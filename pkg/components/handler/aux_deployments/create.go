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
	"time"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_containers "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/containers"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	helper_uuid "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/uuid"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) CreateDeployment(
	ctx context.Context,
	module pkg_models.Module,
	activeDeployment pkg_models.Deployment,
	dependencies map[string]pkg_models.DeploymentReduced,
	serviceInput lib_models.AuxiliaryDeploymentInputBase,
) (lib_models.AuxiliaryDeploymentResult, error) {
	mu := h.mutexes.Get(activeDeployment.Id)
	mu.Lock()
	defer mu.Unlock()
	auxService, ok := module.AuxServices[serviceInput.Reference]
	if !ok {
		msg := "reference not found"
		logger.ErrorContext(
			ctx,
			"create auxiliary deployment",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.Reference, serviceInput.Reference,
			slog_keys.Error, msg,
		)
		return lib_models.AuxiliaryDeploymentResult{}, lib_errors.New[lib_errors.ErrInvalidInput](msg)
	}
	auxDeploymentVolumes, err := h.databaseHandler.ReadAuxiliaryDeploymentVolumes(ctx, activeDeployment.Id, nil)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"create auxiliary deployment, read volumes from database",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.Reference, serviceInput.Reference,
			slog_keys.Error, err,
		)
		return lib_models.AuxiliaryDeploymentResult{}, err
	}
	err = validateImage(module.AuxImgSrc, serviceInput.Image)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"create auxiliary deployment, validate image",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.Reference, serviceInput.Reference,
			slog_keys.Error, err,
		)
		return lib_models.AuxiliaryDeploymentResult{}, err
	}
	id, err := helper_uuid.New()
	if err != nil {
		logger.ErrorContext(
			ctx,
			"create auxiliary deployment, generate id",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.Reference, serviceInput.Reference,
			slog_keys.Error, err,
		)
		return lib_models.AuxiliaryDeploymentResult{}, err
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
		logger.ErrorContext(
			ctx,
			"create auxiliary deployment, generate new auxiliary deployment",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.Reference, serviceInput.Reference,
			slog_keys.Error, err,
		)
		return lib_models.AuxiliaryDeploymentResult{}, err
	}
	newAuxDeployment.Created = helper_time.Now()
	newAuxDeployment.Updated = newAuxDeployment.Created
	deploymentConfigs, err := getDeploymentConfigs(module.Configs, auxService.Configs, activeDeployment.Configs)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"create auxiliary deployment, get configs",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.Reference, serviceInput.Reference,
			slog_keys.Error, err,
		)
		return lib_models.AuxiliaryDeploymentResult{}, err
	}
	newAuxDeploymentVolumes := getNewVolumes(activeDeployment.Id, serviceInput.Volumes, auxDeploymentVolumes)
	err = h.databaseHandler.CreateAuxiliaryDeploymentVolumes(
		ctx,
		activeDeployment.Id,
		slices.Collect(maps.Values(newAuxDeploymentVolumes)),
	)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"create auxiliary deployment, write volumes to database",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.Reference, serviceInput.Reference,
			slog_keys.Error, err,
		)
		return lib_models.AuxiliaryDeploymentResult{}, err
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
		logger.ErrorContext(
			ctx,
			"create auxiliary deployment, write to database",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.Reference, serviceInput.Reference,
			slog_keys.Error, err,
		)
		return lib_models.AuxiliaryDeploymentResult{}, err
	}
	defer func() {
		if err != nil {
			e := h.databaseHandler.DeleteAuxiliaryDeployment(ctx, id)
			if e != nil {
				logger.ErrorContext(
					ctx,
					"create auxiliary deployment, remove from database",
					slog_keys.ModuleId, module.ID,
					slog_keys.DeploymentId, activeDeployment.Id,
					slog_keys.Reference, serviceInput.Reference,
					slog_keys.Error, e,
				)
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
		logger.ErrorContext(
			ctx,
			"create auxiliary deployment, ensure environment",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.Reference, serviceInput.Reference,
			slog_keys.Error, err,
		)
		return lib_models.AuxiliaryDeploymentResult{}, err
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
		logger.ErrorContext(
			ctx,
			"create auxiliary deployment, create container",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.Reference, serviceInput.Reference,
			slog_keys.Error, err,
		)
		return lib_models.AuxiliaryDeploymentResult{}, err
	}
	return lib_models.AuxiliaryDeploymentResult{
		Id:             newAuxDeployment.Id,
		ContainerAlias: newAuxDeployment.Container.Alias,
	}, nil
}

func (h *Handler) ensureAuxDeploymentEnvironment(
	ctx context.Context,
	deploymentId string,
	imageName string,
	forceImagePull bool,
	volumes map[string]lib_models.AuxiliaryDeploymentVolume,
) error {
	err := helper_containers.EnsureImage(
		ctx,
		h.containerEngineWrapperClient,
		imageName,
		forceImagePull,
		h.config.PathEscapeDepth,
		time.Duration(h.config.JobPollInterval),
	)
	if err != nil {
		return err
	}
	return h.ensureContainerVolumes(ctx, volumes, deploymentId)
}

func getAuxiliaryDeployment(
	moduleAuxServiceName string,
	moduleAuxServiceRunConfig external_models.ModuleLibRunConfig,
	deploymentId string,
	auxDeploymentId string,
	containerAlias string,
	serviceInput lib_models.AuxiliaryDeploymentInputBase,
) (pkg_models.AuxiliaryDeployment, error) {
	ctrName, err := helper_naming.NewContainerName(constants.AuxDeploymentAbbreviation)
	if err != nil {
		return pkg_models.AuxiliaryDeployment{}, err
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
	return pkg_models.AuxiliaryDeployment{
		Id:           auxDeploymentId,
		DeploymentId: deploymentId,
		Reference:    serviceInput.Reference,
		Name:         name,
		Image:        serviceInput.Image,
		Enabled:      serviceInput.Enabled,
		Container: pkg_models.AuxiliaryDeploymentContainer{
			Name:  ctrName,
			Alias: containerAlias,
		},
		RunConfig: lib_models.AuxiliaryDeploymentRunConfig{
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
			return errors.New(fmt.Sprintf("regex pattern '%s' invalid", s))
		}
		if re.MatchString(image) {
			return nil
		}
	}
	return errors.New("image invalid")
}
