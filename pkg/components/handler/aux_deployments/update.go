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

package handler_aux_deployments

import (
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/aux_deployments"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployments"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/modules"
)

func (h *Handler) UpdateAuxiliaryDeployment(
	ctx context.Context,
	module models_handler_modules.Module,
	activeDeployment models_handler_deployments.Deployment,
	dependencies map[string]models_handler_deployments.DeploymentReduced,
	auxDeploymentId string,
	serviceInput models_handler_aux_deployments.UpdateServiceInput,
) error {
	auxService, ok := module.AuxServices[serviceInput.Reference]
	if !ok {
		return errors.New("auxiliary service reference not found") // TODO
	}
	currentAuxDeployment, err := h.databaseHandler.ReadAuxiliaryDeployment(ctx, activeDeployment.Id, auxDeploymentId)
	if err != nil {
		return err
	}
	err = validateImage(module.AuxImgSrc, serviceInput.Image)
	if err != nil {
		return err
	}
	if serviceInput.Incremental {
		serviceInput.Volumes, err = h.updateServiceInputVolumes(ctx, auxDeploymentId, serviceInput.Volumes)
		if err != nil {
			return err
		}
		serviceInput.Labels, err = h.updateServiceInputLabels(ctx, auxDeploymentId, serviceInput.Labels)
		if err != nil {
			return err
		}
		serviceInput.Configs, err = h.updateServiceInputConfigs(ctx, auxDeploymentId, serviceInput.Configs)
		if err != nil {
			return err
		}
	}
	auxDeploymentVolumes, err := h.databaseHandler.ReadAuxiliaryDeploymentVolumes(ctx, activeDeployment.Id)
	if err != nil {
		return err
	}
	newAuxDeployment, err := getAuxiliaryDeployment(
		auxService.Name,
		auxService.RunConfig,
		activeDeployment.Id,
		currentAuxDeployment.Id,
		currentAuxDeployment.Container.Alias,
		serviceInput.ServiceInput,
	)
	if err != nil {
		return err
	}
	newAuxDeployment.Updated = helper_time.Now()
	deploymentConfigs, err := getDeploymentConfigs(module.Configs, auxService.Configs, activeDeployment.Configs)
	if err != nil {
		return err
	}
	newAuxDeploymentVolumes := getNewVolumes(activeDeployment.Id, serviceInput.Volumes, auxDeploymentVolumes)
	err = h.databaseHandler.CreateAuxiliaryDeploymentVolumes(
		ctx,
		activeDeployment.Id,
		slices.Collect(maps.Values(newAuxDeploymentVolumes)),
	)
	if err != nil {
		return err
	}
	maps.Copy(auxDeploymentVolumes, newAuxDeploymentVolumes)
	volumeMounts := getVolumeMounts(newAuxDeployment.Id, serviceInput.Volumes, auxDeploymentVolumes)
	err = h.databaseHandler.UpdateAuxiliaryDeployment(
		ctx,
		newAuxDeployment,
		serviceInput.Labels,
		serviceInput.Configs,
		volumeMounts,
	)
	err = h.ensureAuxDeploymentEnvironment(
		ctx,
		activeDeployment.Id,
		serviceInput.Image,
		serviceInput.PullImage,
		auxDeploymentVolumes,
	)
	if err != nil {
		return err
	}
	mergedConfigs := mergeConfigs(deploymentConfigs, serviceInput.Configs)
	return h.createContainer(
		ctx,
		auxService,
		serviceInput.Reference,
		activeDeployment,
		dependencies,
		newAuxDeployment,
		mergedConfigs,
		volumeMounts,
	)
}

func (h *Handler) updateServiceInputVolumes(
	ctx context.Context,
	auxDeploymentId string,
	serviceInputVolumes map[string]string,
) (map[string]string, error) {
	currentVolumeMounts, err := h.databaseHandler.ReadAuxiliaryDeploymentVolumeMounts(ctx, auxDeploymentId)
	if err != nil {
		return nil, err
	}
	tmp := make(map[string]string)
	for _, volumeMount := range currentVolumeMounts {
		tmp[volumeMount.MountPath] = volumeMount.Reference
	}
	maps.Copy(tmp, serviceInputVolumes)
	return tmp, nil
}

func (h *Handler) updateServiceInputLabels(
	ctx context.Context,
	auxDeploymentId string,
	serviceInputLabels map[string]string,
) (map[string]string, error) {
	currentLabels, err := h.databaseHandler.ReadAuxiliaryDeploymentLabels(ctx, auxDeploymentId)
	if err != nil {
		return nil, err
	}
	maps.Copy(currentLabels, serviceInputLabels)
	return currentLabels, nil
}

func (h *Handler) updateServiceInputConfigs(
	ctx context.Context,
	auxDeploymentId string,
	serviceInputConfigs map[string]string,
) (map[string]string, error) {
	currentConfigs, err := h.databaseHandler.ReadAuxiliaryDeploymentConfigs(ctx, auxDeploymentId)
	if err != nil {
		return nil, err
	}
	maps.Copy(currentConfigs, serviceInputConfigs)
	return currentConfigs, nil
}
