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

package handler_deployments

import (
	"context"
	"maps"
	"slices"

	helper_maps "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/maps"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	models_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/deployments"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) GetReducedDeployments(
	ctx context.Context,
	filter models_deployments.DeploymentsFilterWithState,
) (map[string]models_deployments.DeploymentReduced, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.getDeploymentsReduced(ctx, filter)
}

func (h *Handler) GetReducedDeploymentsByModuleIds(
	ctx context.Context,
	filter models_deployments.DeploymentsFilterWithState,
) (map[string]models_deployments.DeploymentReduced, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	deployments, err := h.getDeploymentsReduced(ctx, filter)
	if err != nil {
		return nil, err
	}
	deployments = helper_maps.CollectFunc(maps.Values(deployments), func(value models_deployments.DeploymentReduced) string {
		return value.ModuleId
	})
	return deployments, nil
}

func (h *Handler) GetDeployment(ctx context.Context, id string) (models_deployments.Deployment, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	deployments, err := h.getDeployments(
		ctx,
		models_deployments.DeploymentsFilterWithState{
			DeploymentsFilter: models_deployments.DeploymentsFilter{Ids: []string{id}},
		},
	)
	if err != nil {
		return models_deployments.Deployment{}, err
	}
	if len(deployments) == 0 {
		return models_deployments.Deployment{}, models_error.NotFoundErr
	}
	return deployments[id], nil
}

func (h *Handler) GetDeploymentByModuleId(ctx context.Context, moduleId string) (models_deployments.Deployment, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	deployments, err := h.getDeployments(
		ctx,
		models_deployments.DeploymentsFilterWithState{
			DeploymentsFilter: models_deployments.DeploymentsFilter{ModuleIds: []string{moduleId}},
		},
	)
	if err != nil {
		return models_deployments.Deployment{}, err
	}
	if len(deployments) == 0 {
		return models_deployments.Deployment{}, models_error.NotFoundErr
	}
	deployments = helper_maps.CollectFunc(maps.Values(deployments), func(value models_deployments.Deployment) string {
		return value.ModuleId
	})
	return deployments[moduleId], nil
}

func (h *Handler) GetDeploymentIds(
	ctx context.Context,
	filter models_deployments.DeploymentsFilter,
) (map[string]string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	deployments, err := h.databaseHandler.ReadDeployments(ctx, filter)
	if err != nil {
		return nil, err
	}

	ids := make(map[string]string)
	for id, deployment := range deployments {
		ids[id] = deployment.ModuleId
	}
	return ids, nil
}

func (h *Handler) GetDeployments(
	ctx context.Context,
	filter models_deployments.DeploymentsFilterWithState,
) (map[string]models_deployments.Deployment, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.getDeployments(ctx, filter)
}

func (h *Handler) GetDeploymentsByModuleIds(
	ctx context.Context,
	filter models_deployments.DeploymentsFilterWithState,
) (map[string]models_deployments.Deployment, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	deployments, err := h.getDeployments(ctx, filter)
	if err != nil {
		return nil, err
	}
	deployments = helper_maps.CollectFunc(maps.Values(deployments), func(value models_deployments.Deployment) string {
		return value.ModuleId
	})
	return deployments, nil
}

func (h *Handler) getDeploymentsReduced(
	ctx context.Context,
	filter models_deployments.DeploymentsFilterWithState,
) (map[string]models_deployments.DeploymentReduced, error) {
	stgDeps, err := h.databaseHandler.ReadDeployments(ctx, filter.DeploymentsFilter)
	if err != nil {
		return nil, err
	}
	depIds := slices.Collect(maps.Keys(stgDeps))
	deploymentsContainers, err := h.databaseHandler.ReadDeploymentsContainers(ctx, depIds)
	if err != nil {
		return nil, err
	}
	cewContainersMap, cewErr := h.getCewContainers(ctx, deploymentsContainers)
	if cewErr != nil {
		logger.Error("error getting containers") // TODO
	}
	deployments := make(map[string]models_deployments.DeploymentReduced)
	for id, stgDep := range stgDeps {
		deploymentContainers := deploymentsContainers[id]
		deployment := models_deployments.DeploymentReduced{
			DeploymentBase: stgDep,
			Containers:     getContainers(deploymentContainers, cewContainersMap),
		}
		if cewErr == nil && deployment.Enabled {
			deployment.State = getDeploymentState(getContainersCombinedState(deploymentContainers, cewContainersMap))
		}
		if filter.State > 0 && deployment.State != filter.State {
			continue
		}
		deployments[id] = deployment
	}
	return deployments, nil
}

func (h *Handler) getDeployments(
	ctx context.Context,
	filter models_deployments.DeploymentsFilterWithState,
) (map[string]models_deployments.Deployment, error) {
	stgDeps, err := h.databaseHandler.ReadDeployments(ctx, filter.DeploymentsFilter)
	if err != nil {
		return nil, err
	}
	depIds := slices.Collect(maps.Keys(stgDeps))
	deploymentsUserData, err := h.getDeploymentsUserDataFromDB(ctx, depIds)
	if err != nil {
		return nil, err
	}
	deploymentsVolumes, deploymentsContainers, err := h.getDeploymentsVolumesAndContainersFromDB(ctx, depIds)
	if err != nil {
		return nil, err
	}
	cewContainersMap, cewErr := h.getCewContainers(ctx, deploymentsContainers)
	if cewErr != nil {
		logger.Error("error getting containers") // TODO
	}
	deployments := make(map[string]models_deployments.Deployment)
	for id, stgDep := range stgDeps {
		deploymentContainers := deploymentsContainers[id]
		deployment := models_deployments.Deployment{
			DeploymentBase: stgDep,
			Containers:     getContainers(deploymentContainers, cewContainersMap),
			Volumes:        deploymentsVolumes[id],
			HostResources:  deploymentsUserData[id].HostResources,
			Secrets:        deploymentsUserData[id].Secrets,
			Configs:        deploymentsUserData[id].Configs,
			GlobalConfigs:  deploymentsUserData[id].GlobalConfigs,
			Files:          deploymentsUserData[id].Files,
			FileGroups:     deploymentsUserData[id].FileGroups,
		}
		if cewErr == nil && deployment.Enabled {
			deployment.State = getDeploymentState(getContainersCombinedState(deploymentContainers, cewContainersMap))
		}
		if filter.State > 0 && deployment.State != filter.State {
			continue
		}
		deployments[id] = deployment
	}
	return deployments, nil
}

func (h *Handler) getCewContainers(
	ctx context.Context,
	stgDepsContainers map[string]map[string]models_deployments.ContainerBase,
) (map[string]models_external.Container, error) {
	var ctrNames []string
	for _, stgDepContainers := range stgDepsContainers {
		ctrNames = append(ctrNames, helper_slices.CollectFunc(maps.Values(stgDepContainers), func(item models_deployments.ContainerBase) string {
			return item.Name
		})...)
	}
	cewContainers, err := h.containerEngineWrapperClient.GetContainers(ctx, models_external.ContainersFilter{Names: ctrNames})
	if err != nil {
		return nil, err
	}
	cewContainersMap := maps.Collect(helper_slices.AllFunc(cewContainers, func(item models_external.Container) string {
		return item.Name
	}))
	return cewContainersMap, nil
}

func getContainers(
	stgDepContainers map[string]models_deployments.ContainerBase,
	cewContainers map[string]models_external.Container,
) map[string]models_deployments.Container {
	containers := make(map[string]models_deployments.Container)
	for reference, stgDepContainer := range stgDepContainers {
		container := models_deployments.Container{ContainerBase: stgDepContainer}
		cewContainer, ok := cewContainers[stgDepContainer.Name]
		if ok {
			container.ImageId = cewContainer.ImageID
			container.State = cewContainer.State
			if cewContainer.Health != nil {
				container.Health = *cewContainer.Health
			}
		} else {
			logger.Error("missing container") // TODO
		}
		containers[reference] = container
	}
	return containers
}

func getDeploymentState(containersState int) int {
	if containersState == containersStateRunning {
		return models_deployments.StateHealthy
	}
	return models_deployments.StateUnhealthy
}

func getContainersCombinedState(
	deploymentContainers map[string]models_deployments.ContainerBase,
	existingContainers map[string]models_external.Container,
) int {
	var runningCount int
	var unhealthyCount int
	for _, deploymentContainer := range deploymentContainers {
		existingContainer, ok := existingContainers[deploymentContainer.Name]
		if !ok {
			return containersStateBroken
		}
		if existingContainer.State == models_external.CewRunningState {
			runningCount++
		}
		if existingContainer.Health != nil && *existingContainer.Health == models_external.CewUnhealthyState {
			unhealthyCount++
		}
	}
	switch runningCount {
	case 0:
		return containersStateStopped
	case len(deploymentContainers):
		if unhealthyCount > 0 {
			return containersStateUnhealthy
		}
		return containersStateRunning
	}
	return containersStatePartial
}
