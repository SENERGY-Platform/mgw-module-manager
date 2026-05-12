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

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_containers "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/containers"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) RecreateDeployments(
	ctx context.Context,
	module pkg_models.Module,
	activeDeployment pkg_models.Deployment,
	dependencies map[string]pkg_models.DeploymentReduced,
	filter lib_models.AuxiliaryDeploymentsFilterWithState,
) ([]lib_models.AuxiliaryDeploymentBatchResult, error) {
	mu := h.mutexes.Get(activeDeployment.Id)
	mu.Lock()
	defer mu.Unlock()
	auxDeployments, err := h.readAuxiliaryDeploymentsAndFilterByState(ctx, activeDeployment.Id, filter)
	if err != nil {
		logger.Error(
			"recreate auxiliary deployments, read from database",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.Filter, filter,
			slog_keys.Error, err,
		)
		return nil, err
	}
	auxDepIds := slices.Collect(maps.Keys(auxDeployments))
	auxDepConfigs, err := h.databaseHandler.ReadAuxiliaryDeploymentsConfigs(ctx, auxDepIds)
	if err != nil {
		logger.Error(
			"recreate auxiliary deployments, read configs from database",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentIds, auxDepIds,
			slog_keys.Error, err,
		)
		return nil, err
	}
	auxDepVolumeMounts, err := h.databaseHandler.ReadAuxiliaryDeploymentsVolumeMounts(ctx, auxDepIds)
	if err != nil {
		logger.Error(
			"recreate auxiliary deployments, read volume mounts from database",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentIds, auxDepIds,
			slog_keys.Error, err,
		)
		return nil, err
	}
	auxDeploymentVolumes, err := h.databaseHandler.ReadAuxiliaryDeploymentVolumes(ctx, activeDeployment.Id, nil)
	if err != nil {
		logger.Error(
			"recreate auxiliary deployments, read volumes from database",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.Error, err,
		)
		return nil, err
	}
	err = h.ensureContainerVolumes(ctx, auxDeploymentVolumes, activeDeployment.Id)
	if err != nil {
		logger.Error(
			"recreate auxiliary deployments, ensure volumes",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.Volumes, auxDeploymentVolumes,
			slog_keys.Error, err,
		)
		return nil, err
	}
	cacheDeploymentConfigs := make(map[string]map[string]string)
	var results []lib_models.AuxiliaryDeploymentBatchResult
	for id, auxDep := range auxDeployments {
		result := lib_models.AuxiliaryDeploymentBatchResult{Id: id}
		auxService, ok := module.AuxServices[auxDep.Reference]
		if !ok {
			msg := "auxiliary service reference not found"
			logger.Error(
				"recreate auxiliary deployments",
				slog_keys.ModuleId, module.ID,
				slog_keys.DeploymentId, activeDeployment.Id,
				slog_keys.AuxDeploymentId, id,
				slog_keys.Error, msg,
			)
			result.ErrorResult = lib_models.NewErrorResult(msg)
			results = append(results, result)
			continue
		}
		deploymentConfigs, ok := cacheDeploymentConfigs[auxDep.Reference]
		if !ok {
			deploymentConfigs, err = getDeploymentConfigs(module.Configs, auxService.Configs, activeDeployment.Configs)
			if err != nil {
				logger.Error(
					"recreate auxiliary deployments, get deployment configs",
					slog_keys.ModuleId, module.ID,
					slog_keys.DeploymentId, activeDeployment.Id,
					slog_keys.AuxDeploymentId, id,
					slog_keys.Error, err,
				)
				result.ErrorResult = lib_models.NewErrorResult(err.Error())
				results = append(results, result)
				continue
			}
		}
		err = h.recreateAuxiliaryDeployment(
			ctx,
			auxService,
			activeDeployment,
			dependencies,
			deploymentConfigs,
			auxDep,
			auxDepConfigs[id],
			auxDepVolumeMounts[id],
		)
		if err != nil {
			result.ErrorResult = lib_models.NewErrorResult(err.Error())
		}
		results = append(results, result)
	}
	return results, nil
}

func (h *Handler) recreateAuxiliaryDeployment(
	ctx context.Context,
	auxService external_models.ModuleLibAuxService,
	activeDeployment pkg_models.Deployment,
	dependencies map[string]pkg_models.DeploymentReduced,
	deploymentConfigs map[string]string,
	currentAuxDeployment pkg_models.AuxiliaryDeployment,
	configs map[string]string,
	volumeMounts []pkg_models.AuxiliaryDeploymentVolumeMount,
) error {
	ctrName, err := helper_naming.NewContainerName(constants.AuxDeploymentAbbreviation)
	if err != nil {
		logger.Error(
			"recreate auxiliary deployments, generate new container name",
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, currentAuxDeployment.Id,
			slog_keys.Error, err,
		)
		return err
	}
	currentAuxDeployment.Container.Name = ctrName
	err = helper_containers.Stop(ctx, h.containerEngineWrapperClient, currentAuxDeployment.Container.Name, h.config.JobPollInterval)
	if err != nil {
		logger.Error(
			"recreate auxiliary deployments, stop old container",
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
			"recreate auxiliary deployments, remove old container",
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, currentAuxDeployment.Id,
			slog_keys.ContainerName, currentAuxDeployment.Container.Name,
			slog_keys.Error, err,
		)
		return err
	}
	err = h.databaseHandler.UpdateAuxiliaryDeploymentContainerName(ctx, currentAuxDeployment.Id, ctrName)
	if err != nil {
		logger.Error(
			"recreate auxiliary deployments, write new container name to database",
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, currentAuxDeployment.Id,
			slog_keys.Error, err,
		)
		return err
	}
	err = helper_containers.EnsureImage(
		ctx,
		h.containerEngineWrapperClient,
		currentAuxDeployment.Image,
		false,
		h.config.PathEscapeDepth,
		h.config.JobPollInterval,
	)
	if err != nil {
		logger.Error(
			"recreate auxiliary deployments, ensure image",
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, currentAuxDeployment.Id,
			slog_keys.Error, err,
		)
		return err
	}
	err = h.createContainer(
		ctx,
		auxService,
		currentAuxDeployment.Reference,
		activeDeployment,
		dependencies,
		currentAuxDeployment,
		mergeConfigs(deploymentConfigs, configs),
		volumeMounts,
	)
	if err != nil {
		logger.Error(
			"recreate auxiliary deployments, create container",
			slog_keys.DeploymentId, activeDeployment.Id,
			slog_keys.AuxDeploymentId, currentAuxDeployment.Id,
			slog_keys.Error, err,
		)
		return err
	}
	return nil
}

func (h *Handler) readAuxiliaryDeploymentsAndFilterByState(
	ctx context.Context,
	deploymentId string,
	filter lib_models.AuxiliaryDeploymentsFilterWithState,
) (map[string]pkg_models.AuxiliaryDeployment, error) {
	auxDeployments, err := h.databaseHandler.ReadAuxiliaryDeployments(ctx, deploymentId, filter.AuxiliaryDeploymentsFilter)
	if err != nil {
		return nil, err
	}
	if filter.State != "" {
		cewContainers, err := h.getCewContainers(ctx, auxDeployments)
		if err != nil {
			return nil, err
		}
		tmp := make(map[string]pkg_models.AuxiliaryDeployment)
		for id, auxDep := range auxDeployments {
			cewContainer := cewContainers[auxDep.Container.Name]
			if cewContainer.State == filter.State {
				tmp[id] = auxDep
			}
		}
		auxDeployments = tmp
	}
	return auxDeployments, nil
}
