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
	"maps"
	"slices"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_containers "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/containers"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
)

func (h *Handler) UpdateDeployment(
	ctx context.Context,
	module pkg_models.Module,
	activeDeployment pkg_models.Deployment,
	dependencies map[string]pkg_models.DeploymentReduced,
	auxDeploymentId string,
	serviceInput lib_models.AuxiliaryDeploymentUpdateInputBase,
) error {
	mu := h.mutexes.Get(activeDeployment.Id)
	mu.Lock()
	defer mu.Unlock()
	currentAuxDeployment, err := h.databaseHandler.ReadAuxiliaryDeployment(ctx, activeDeployment.Id, auxDeploymentId)
	if err != nil {
		logger.Error(
			"update auxiliary deployment, read from database",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, auxDeploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	if serviceInput.Reference == "" {
		serviceInput.Reference = currentAuxDeployment.Reference
	}
	auxService, ok := module.AuxServices[serviceInput.Reference]
	if !ok {
		msg := "reference not found"
		logger.Error(
			"update auxiliary deployment, read from database",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, auxDeploymentId,
			slog_keys.Reference, serviceInput.Reference,
			slog_keys.Error, msg,
		)
		return lib_errors.New[lib_errors.ErrInvalidInput](msg)
	}
	err = validateImage(module.AuxImgSrc, serviceInput.Image)
	if err != nil {
		logger.Error(
			"update auxiliary deployment, validate image",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, auxDeploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	if serviceInput.Incremental {
		serviceInput.Volumes, err = h.updateServiceInputVolumes(ctx, auxDeploymentId, serviceInput.Volumes)
		if err != nil {
			logger.Error(
				"update auxiliary deployment, incremental update volumes",
				slog_keys.ModuleId, module.ID,
				slog_keys.DeploymentId, activeDeployment.Id,
				slog_keys.AuxDeploymentId, auxDeploymentId,
				slog_keys.Error, err,
			)
			return err
		}
		serviceInput.Labels, err = h.updateServiceInputLabels(ctx, auxDeploymentId, serviceInput.Labels)
		if err != nil {
			logger.Error(
				"update auxiliary deployment, incremental update labels",
				slog_keys.ModuleId, module.ID,
				slog_keys.DeploymentId, activeDeployment.Id,
				slog_keys.AuxDeploymentId, auxDeploymentId,
				slog_keys.Error, err,
			)
			return err
		}
		serviceInput.Configs, err = h.updateServiceInputConfigs(ctx, auxDeploymentId, serviceInput.Configs)
		if err != nil {
			logger.Error(
				"update auxiliary deployment, incremental update configs",
				slog_keys.ModuleId, module.ID,
				slog_keys.DeploymentId, activeDeployment.Id,
				slog_keys.AuxDeploymentId, auxDeploymentId,
				slog_keys.Error, err,
			)
			return err
		}
	}
	auxDeploymentVolumes, err := h.databaseHandler.ReadAuxiliaryDeploymentVolumes(ctx, activeDeployment.Id, nil)
	if err != nil {
		logger.Error(
			"update auxiliary deployment, read volumes from database",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, auxDeploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	if serviceInput.Name == "" {
		serviceInput.Name = currentAuxDeployment.Name
	}
	newAuxDeployment, err := getAuxiliaryDeployment(
		auxService.Name,
		auxService.RunConfig,
		activeDeployment.Id,
		currentAuxDeployment.Id,
		currentAuxDeployment.Container.Alias,
		serviceInput.AuxiliaryDeploymentInputBase,
	)
	if err != nil {
		logger.Error(
			"update auxiliary deployment, generate new auxiliary deployment",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, auxDeploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	newAuxDeployment.Updated = helper_time.Now()
	deploymentConfigs, err := getDeploymentConfigs(module.Configs, auxService.Configs, activeDeployment.Configs)
	if err != nil {
		logger.Error(
			"update auxiliary deployment, get deployment configs",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, auxDeploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	newAuxDeploymentVolumes := getNewVolumes(activeDeployment.Id, serviceInput.Volumes, auxDeploymentVolumes)
	err = h.databaseHandler.CreateAuxiliaryDeploymentVolumes(
		ctx,
		activeDeployment.Id,
		slices.Collect(maps.Values(newAuxDeploymentVolumes)),
	)
	if err != nil {
		logger.Error(
			"update auxiliary deployment, write volumes to database",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, auxDeploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	maps.Copy(auxDeploymentVolumes, newAuxDeploymentVolumes)
	volumeMounts := getVolumeMounts(newAuxDeployment.Id, serviceInput.Volumes, auxDeploymentVolumes)
	err = helper_containers.Stop(ctx, h.containerEngineWrapperClient, currentAuxDeployment.Container.Name, h.config.JobPollInterval)
	if err != nil {
		logger.Error(
			"update auxiliary deployments, stop old container",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, currentAuxDeployment.Id,
			slog_keys.ContainerName, currentAuxDeployment.Container.Name,
			slog_keys.Error, err,
		)
		return err
	}
	err = helper_containers.Remove(ctx, h.containerEngineWrapperClient, currentAuxDeployment.Container.Name)
	if err != nil {
		logger.Error(
			"update auxiliary deployments, remove old container",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, currentAuxDeployment.Id,
			slog_keys.ContainerName, currentAuxDeployment.Container.Name,
			slog_keys.Error, err,
		)
		return err
	}
	err = h.databaseHandler.UpdateAuxiliaryDeployment(
		ctx,
		newAuxDeployment,
		serviceInput.Labels,
		serviceInput.Configs,
		volumeMounts,
	)
	if err != nil {
		logger.Error(
			"update auxiliary deployments, write to database",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, currentAuxDeployment.Id,
			slog_keys.Error, err,
		)
		return err
	}
	err = h.ensureAuxDeploymentEnvironment(
		ctx,
		activeDeployment.Id,
		serviceInput.Image,
		serviceInput.PullImage,
		auxDeploymentVolumes,
	)
	if err != nil {
		logger.Error(
			"update auxiliary deployments, ensure environment",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, currentAuxDeployment.Id,
			slog_keys.Error, err,
		)
		return err
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
		logger.Error(
			"update auxiliary deployments, create container",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, currentAuxDeployment.Id,
			slog_keys.Error, err,
		)
		return err
	}
	return nil
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
