/*
 * Copyright 2026 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package aux_deployments

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	lib_models_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/lib/models/aux_deployments"
	helper_containers "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/containers"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	helper_uuid "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/uuid"
	models_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/aux_deployments"
	models_constants "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	models_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/deployments"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/modules"
)

func (h *Handler) CreateDeployment(
	ctx context.Context,
	module models_module.Module,
	activeDeployment models_deployments.Deployment,
	dependencies map[string]models_deployments.DeploymentReduced,
	serviceInput lib_models_aux_deployments.ServiceInput,
) (lib_models_aux_deployments.Result, error) {
	mu := h.mutexes.Get(activeDeployment.Id)
	mu.Lock()
	defer mu.Unlock()
	auxService, ok := module.AuxServices[serviceInput.Reference]
	if !ok {
		return lib_models_aux_deployments.Result{}, errors.New("auxiliary service reference not found") // TODO
	}
	auxDeploymentVolumes, err := h.databaseHandler.ReadAuxiliaryDeploymentVolumes(ctx, activeDeployment.Id, nil)
	if err != nil {
		return lib_models_aux_deployments.Result{}, err
	}
	err = validateImage(module.AuxImgSrc, serviceInput.Image)
	if err != nil {
		return lib_models_aux_deployments.Result{}, err
	}
	id, err := helper_uuid.New()
	if err != nil {
		return lib_models_aux_deployments.Result{}, err
	}
	newAuxDeployment, err := getAuxiliaryDeployment(
		auxService.Name,
		auxService.RunConfig,
		activeDeployment.Id,
		id,
		helper_naming.NewContainerAlias(activeDeployment.Id, id),
		serviceInput,
	)
	if err != nil {
		return lib_models_aux_deployments.Result{}, err
	}
	newAuxDeployment.Created = helper_time.Now()
	newAuxDeployment.Updated = newAuxDeployment.Created
	deploymentConfigs, err := getDeploymentConfigs(module.Configs, auxService.Configs, activeDeployment.Configs)
	if err != nil {
		return lib_models_aux_deployments.Result{}, err
	}
	newAuxDeploymentVolumes := getNewVolumes(activeDeployment.Id, serviceInput.Volumes, auxDeploymentVolumes)
	err = h.databaseHandler.CreateAuxiliaryDeploymentVolumes(
		ctx,
		activeDeployment.Id,
		slices.Collect(maps.Values(newAuxDeploymentVolumes)),
	)
	if err != nil {
		return lib_models_aux_deployments.Result{}, err
	}
	maps.Copy(auxDeploymentVolumes, newAuxDeploymentVolumes)
	volumeMounts := getVolumeMounts(newAuxDeployment.Id, serviceInput.Volumes, auxDeploymentVolumes)
	err = h.databaseHandler.CreateAuxiliaryDeployment(
		ctx,
		newAuxDeployment,
		serviceInput.Labels,
		serviceInput.Configs,
		volumeMounts,
	)
	if err != nil {
		return lib_models_aux_deployments.Result{}, err
	}
	defer func() {
		if err != nil {
			e := h.databaseHandler.DeleteAuxiliaryDeployment(ctx, id)
			if e != nil {
				logger.Error(e.Error()) // TODO
			}
		}
	}()
	err = h.ensureAuxDeploymentEnvironment(
		ctx,
		activeDeployment.Id,
		serviceInput.Image,
		serviceInput.PullImage,
		auxDeploymentVolumes,
	)
	if err != nil {
		return lib_models_aux_deployments.Result{}, err
	}
	err = h.createContainer(
		ctx,
		auxService,
		serviceInput.Reference,
		activeDeployment,
		dependencies,
		newAuxDeployment,
		mergeConfigs(deploymentConfigs, serviceInput.Configs),
		volumeMounts,
	)
	if err != nil {
		return lib_models_aux_deployments.Result{}, err
	}
	return lib_models_aux_deployments.Result{
		Id:             newAuxDeployment.Id,
		ContainerAlias: newAuxDeployment.Container.Alias,
	}, nil
}

func (h *Handler) ensureAuxDeploymentEnvironment(
	ctx context.Context,
	deploymentId string,
	imageName string,
	forceImagePull bool,
	volumes map[string]lib_models_aux_deployments.AuxiliaryDeploymentVolume,
) error {
	err := helper_containers.EnsureImage(
		ctx,
		h.containerEngineWrapperClient,
		imageName,
		forceImagePull,
		h.config.PathEscapeDepth,
		h.config.JobPollInterval,
	)
	if err != nil {
		return err
	}
	return h.ensureContainerVolumes(ctx, volumes, deploymentId)
}

func getAuxiliaryDeployment(
	moduleAuxServiceName string,
	moduleAuxServiceRunConfig models_external.ModuleLibRunConfig,
	deploymentId string,
	auxDeploymentId string,
	containerAlias string,
	serviceInput lib_models_aux_deployments.ServiceInput,
) (models_aux_deployments.AuxiliaryDeployment, error) {
	ctrName, err := helper_naming.NewContainerName(models_constants.AuxDeploymentAbbreviation)
	if err != nil {
		return models_aux_deployments.AuxiliaryDeployment{}, err
	}
	name := moduleAuxServiceName
	if serviceInput.Name != "" {
		name = serviceInput.Name
	}
	command := moduleAuxServiceRunConfig.Command
	if len(serviceInput.RunConfig.Command) > 0 {
		command = serviceInput.RunConfig.Command
	}
	pseudoTTY := moduleAuxServiceRunConfig.PseudoTTY
	if serviceInput.RunConfig.PseudoTTY < 0 {
		pseudoTTY = false
	}
	if serviceInput.RunConfig.PseudoTTY > 0 {
		pseudoTTY = true
	}
	return models_aux_deployments.AuxiliaryDeployment{
		Id:           auxDeploymentId,
		DeploymentId: deploymentId,
		Reference:    serviceInput.Reference,
		Name:         name,
		Image:        serviceInput.Image,
		Enabled:      serviceInput.Enabled,
		Container: models_aux_deployments.AuxiliaryDeploymentContainer{
			Name:  ctrName,
			Alias: containerAlias,
		},
		RunConfig: lib_models_aux_deployments.AuxiliaryDeploymentRunConfig{
			Command:   command,
			PseudoTTY: pseudoTTY,
		},
		Recreate: serviceInput.Recreate,
	}, nil
}

func validateImage(auxImgSrc map[string]struct{}, image string) error {
	for src := range auxImgSrc {
		s := strings.ReplaceAll(src, ".", "\\.")
		if strings.Contains(src, "*") {
			s = strings.ReplaceAll(s, "*", ".+")
		} else {
			s = s + "(?:$|:.+$)"
		}
		s = "^" + s
		re, err := regexp.Compile(s)
		if err != nil {
			return fmt.Errorf("invalid regex pattern '%s'", s) // TODO
		}
		if re.MatchString(image) {
			return nil
		}
	}
	return errors.New("invalid image") // TODO
}
