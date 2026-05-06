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

package deployments

import (
	"context"
	"maps"
	"slices"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/models/constants"
	lib_external "github.com/SENERGY-Platform/mgw-module-manager/lib/models/external"
	helper_maps "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/maps"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) GetReducedDeployments(
	ctx context.Context,
	filter pkg_models.DeploymentsFilterWithState,
) (map[string]pkg_models.DeploymentReduced, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.getDeploymentsReduced(ctx, filter)
}

func (h *Handler) GetReducedDeploymentsByModuleIds(
	ctx context.Context,
	filter pkg_models.DeploymentsFilterWithState,
) (map[string]pkg_models.DeploymentReduced, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	deployments, err := h.getDeploymentsReduced(ctx, filter)
	if err != nil {
		return nil, err
	}
	deployments = helper_maps.CollectFunc(maps.Values(deployments), func(value pkg_models.DeploymentReduced) string {
		return value.ModuleId
	})
	return deployments, nil
}

func (h *Handler) GetDeployment(ctx context.Context, id string) (pkg_models.Deployment, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	deployments, err := h.getDeployments(
		ctx,
		pkg_models.DeploymentsFilterWithState{
			DeploymentsFilter: pkg_models.DeploymentsFilter{Ids: []string{id}},
		},
	)
	if err != nil {
		return pkg_models.Deployment{}, err
	}
	if len(deployments) == 0 {
		return pkg_models.Deployment{}, pkg_models.NotFoundErr
	}
	return deployments[id], nil
}

func (h *Handler) GetDeploymentByModuleId(ctx context.Context, moduleId string) (pkg_models.Deployment, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	deployments, err := h.getDeployments(
		ctx,
		pkg_models.DeploymentsFilterWithState{
			DeploymentsFilter: pkg_models.DeploymentsFilter{ModuleIds: []string{moduleId}},
		},
	)
	if err != nil {
		return pkg_models.Deployment{}, err
	}
	if len(deployments) == 0 {
		return pkg_models.Deployment{}, pkg_models.NotFoundErr
	}
	deployments = helper_maps.CollectFunc(maps.Values(deployments), func(value pkg_models.Deployment) string {
		return value.ModuleId
	})
	return deployments[moduleId], nil
}

func (h *Handler) GetDeploymentIds(
	ctx context.Context,
	filter pkg_models.DeploymentsFilter,
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
	filter pkg_models.DeploymentsFilterWithState,
) (map[string]pkg_models.Deployment, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.getDeployments(ctx, filter)
}

func (h *Handler) GetDeploymentsByModuleIds(
	ctx context.Context,
	filter pkg_models.DeploymentsFilterWithState,
) (map[string]pkg_models.Deployment, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	deployments, err := h.getDeployments(ctx, filter)
	if err != nil {
		return nil, err
	}
	deployments = helper_maps.CollectFunc(maps.Values(deployments), func(value pkg_models.Deployment) string {
		return value.ModuleId
	})
	return deployments, nil
}

func (h *Handler) getDeploymentsReduced(
	ctx context.Context,
	filter pkg_models.DeploymentsFilterWithState,
) (map[string]pkg_models.DeploymentReduced, error) {
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
	deployments := make(map[string]pkg_models.DeploymentReduced)
	for id, stgDep := range stgDeps {
		deploymentContainers := deploymentsContainers[id]
		deployment := pkg_models.DeploymentReduced{
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
	filter pkg_models.DeploymentsFilterWithState,
) (map[string]pkg_models.Deployment, error) {
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
	deployments := make(map[string]pkg_models.Deployment)
	for id, stgDep := range stgDeps {
		deploymentContainers := deploymentsContainers[id]
		deployment := pkg_models.Deployment{
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
	stgDepsContainers map[string]map[string]pkg_models.DeploymentContainerBase,
) (map[string]external_models.CewContainer, error) {
	var ctrNames []string
	for _, stgDepContainers := range stgDepsContainers {
		ctrNames = append(ctrNames, helper_slices.CollectFunc(maps.Values(stgDepContainers), func(item pkg_models.DeploymentContainerBase) string {
			return item.Name
		})...)
	}
	cewContainers, err := h.containerEngineWrapperClient.GetContainers(ctx, external_models.CewContainersFilter{Names: ctrNames})
	if err != nil {
		return nil, err
	}
	cewContainersMap := maps.Collect(helper_slices.AllFunc(cewContainers, func(item external_models.CewContainer) string {
		return item.Name
	}))
	return cewContainersMap, nil
}

func getContainers(
	stgDepContainers map[string]pkg_models.DeploymentContainerBase,
	cewContainers map[string]external_models.CewContainer,
) map[string]pkg_models.DeploymentContainer {
	containers := make(map[string]pkg_models.DeploymentContainer)
	for reference, stgDepContainer := range stgDepContainers {
		container := pkg_models.DeploymentContainer{DeploymentContainerBase: stgDepContainer}
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
		return lib_constants.DeploymentStateHealthy
	}
	return lib_constants.DeploymentStateUnhealthy
}

func getContainersCombinedState(
	deploymentContainers map[string]pkg_models.DeploymentContainerBase,
	existingContainers map[string]external_models.CewContainer,
) int {
	var runningCount int
	var unhealthyCount int
	for _, deploymentContainer := range deploymentContainers {
		existingContainer, ok := existingContainers[deploymentContainer.Name]
		if !ok {
			return containersStateBroken
		}
		if existingContainer.State == lib_external.CewRunningState {
			runningCount++
		}
		if existingContainer.Health != nil && *existingContainer.Health == lib_external.CewUnhealthyState {
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
