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
	"maps"
	"slices"

	lib_models_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/lib/models/aux_deployments"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/lib/models/results"
	helper_containers "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/containers"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	models_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/aux_deployments"
	models_constants "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	models_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/deployments"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/modules"
)

func (h *Handler) RecreateDeployments(
	ctx context.Context,
	module models_module.Module,
	activeDeployment models_deployments.Deployment,
	dependencies map[string]models_deployments.DeploymentReduced,
	filter lib_models_aux_deployments.AuxiliaryDeploymentsFilterWithState,
) ([]lib_models_aux_deployments.BatchResult, error) {
	mu := h.mutexes.Get(activeDeployment.Id)
	mu.Lock()
	defer mu.Unlock()
	auxDeployments, err := h.readAuxiliaryDeploymentsAndFilterByState(ctx, activeDeployment.Id, filter)
	if err != nil {
		return nil, err
	}
	auxDepIds := slices.Collect(maps.Keys(auxDeployments))
	auxDepConfigs, err := h.databaseHandler.ReadAuxiliaryDeploymentsConfigs(ctx, auxDepIds)
	if err != nil {
		return nil, err
	}
	auxDepVolumeMounts, err := h.databaseHandler.ReadAuxiliaryDeploymentsVolumeMounts(ctx, auxDepIds)
	if err != nil {
		return nil, err
	}
	auxDeploymentVolumes, err := h.databaseHandler.ReadAuxiliaryDeploymentVolumes(ctx, activeDeployment.Id, nil)
	if err != nil {
		return nil, err
	}
	err = h.ensureContainerVolumes(ctx, auxDeploymentVolumes, activeDeployment.Id)
	if err != nil {
		return nil, err
	}
	cacheDeploymentConfigs := make(map[string]map[string]string)
	var results []lib_models_aux_deployments.BatchResult
	for id, auxDep := range auxDeployments {
		result := lib_models_aux_deployments.BatchResult{Id: id}
		auxService, ok := module.AuxServices[auxDep.Reference]
		if !ok {
			result.ErrorResult = models_error.NewErrorResult("auxiliary service reference not found")
			results = append(results, result)
			continue
		}
		deploymentConfigs, ok := cacheDeploymentConfigs[auxDep.Reference]
		if !ok {
			deploymentConfigs, err = getDeploymentConfigs(module.Configs, auxService.Configs, activeDeployment.Configs)
			if err != nil {
				result.ErrorResult = models_error.NewErrorResult(err.Error())
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
			result.ErrorResult = models_error.NewErrorResult(err.Error())
		}
		results = append(results, result)
	}
	return results, nil
}

func (h *Handler) recreateAuxiliaryDeployment(
	ctx context.Context,
	auxService models_external.ModuleLibAuxService,
	activeDeployment models_deployments.Deployment,
	dependencies map[string]models_deployments.DeploymentReduced,
	deploymentConfigs map[string]string,
	currentAuxDeployment models_aux_deployments.AuxiliaryDeployment,
	configs map[string]string,
	volumeMounts []models_aux_deployments.AuxiliaryDeploymentVolumeMount,
) error {
	ctrName, err := helper_naming.NewContainerName(models_constants.AuxDeploymentAbbreviation)
	if err != nil {
		return err
	}
	currentAuxDeployment.Container.Name = ctrName
	err = helper_containers.Stop(ctx, h.containerEngineWrapperClient, currentAuxDeployment.Container.Name, h.config.JobPollInterval)
	if err != nil {
		return err
	}
	err = helper_containers.Remove(ctx, h.containerEngineWrapperClient, currentAuxDeployment.Container.Name)
	if err != nil {
		return err
	}
	err = h.databaseHandler.UpdateAuxiliaryDeploymentContainerName(ctx, currentAuxDeployment.Id, ctrName)
	if err != nil {
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
		return err
	}
	return h.createContainer(
		ctx,
		auxService,
		currentAuxDeployment.Reference,
		activeDeployment,
		dependencies,
		currentAuxDeployment,
		mergeConfigs(deploymentConfigs, configs),
		volumeMounts,
	)
}

func (h *Handler) readAuxiliaryDeploymentsAndFilterByState(
	ctx context.Context,
	deploymentId string,
	filter lib_models_aux_deployments.AuxiliaryDeploymentsFilterWithState,
) (map[string]models_aux_deployments.AuxiliaryDeployment, error) {
	auxDeployments, err := h.databaseHandler.ReadAuxiliaryDeployments(ctx, deploymentId, filter.AuxiliaryDeploymentsFilter)
	if err != nil {
		return nil, err
	}
	if filter.State != "" {
		cewContainers, err := h.getCewContainers(ctx, auxDeployments)
		if err != nil {
			return nil, err
		}
		tmp := make(map[string]models_aux_deployments.AuxiliaryDeployment)
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
