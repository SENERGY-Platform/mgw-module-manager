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

	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_deployment "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployment"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) GetDeploymentsReduced(
	ctx context.Context,
	filter models_handler_deployment.DeploymentsFilter,
) (map[string]models_handler_deployment.DeploymentReduced, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
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
	deployments := make(map[string]models_handler_deployment.DeploymentReduced)
	for id, stgDep := range stgDeps {
		deploymentContainers := deploymentsContainers[id]
		deployment := models_handler_deployment.DeploymentReduced{
			Deployment: stgDep,
			Containers: getContainers(deploymentContainers, cewContainersMap),
		}
		if cewErr != nil || !deployment.Enabled {
			deployment.State = models_handler_deployment.StateNotAvailable
		} else {
			deployment.State = getDeploymentState(getContainersCombinedState(deploymentContainers, cewContainersMap))
		}
		if filter.State > 0 && deployment.State != filter.State {
			continue
		}
		deployments[id] = deployment
	}
	return deployments, nil
}

func (h *Handler) GetDeployment(ctx context.Context, id string) (models_handler_deployment.Deployment, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	deployments, err := h.getDeployments(
		ctx,
		models_handler_deployment.DeploymentsFilter{
			DeploymentsFilter: models_handler_storage.DeploymentsFilter{Ids: []string{id}},
		},
	)
	if err != nil {
		return models_handler_deployment.Deployment{}, err
	}
	if len(deployments) == 0 {
		return models_handler_deployment.Deployment{}, models_error.NotFoundErr
	}
	return deployments[id], nil
}

func (h *Handler) GetDeployments(
	ctx context.Context,
	filter models_handler_deployment.DeploymentsFilter,
) (map[string]models_handler_deployment.Deployment, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.getDeployments(ctx, filter)
}

func (h *Handler) getDeployments(
	ctx context.Context,
	filter models_handler_deployment.DeploymentsFilter,
) (map[string]models_handler_deployment.Deployment, error) {
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
	deployments := make(map[string]models_handler_deployment.Deployment)
	for id, stgDep := range stgDeps {
		deploymentContainers := deploymentsContainers[id]
		deployment := models_handler_deployment.Deployment{
			Deployment:    stgDep,
			Containers:    getContainers(deploymentContainers, cewContainersMap),
			Volumes:       deploymentsVolumes[id],
			HostResources: deploymentsUserData[id].HostResources,
			Secrets:       deploymentsUserData[id].Secrets,
			Configs:       deploymentsUserData[id].Configs,
			GlobalConfigs: deploymentsUserData[id].GlobalConfigs,
			Files:         deploymentsUserData[id].Files,
			FileGroups:    deploymentsUserData[id].FileGroups,
		}
		if cewErr != nil || !deployment.Enabled {
			deployment.State = models_handler_deployment.StateNotAvailable
		} else {
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
	stgDepsContainers map[string]map[string]models_handler_storage.DeploymentContainer,
) (map[string]models_external.Container, error) {
	var ctrNames []string
	for _, stgDepContainers := range stgDepsContainers {
		ctrNames = append(ctrNames, helper_slices.CollectFunc(maps.Values(stgDepContainers), func(item models_handler_storage.DeploymentContainer) string {
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
	stgDepContainers map[string]models_handler_storage.DeploymentContainer,
	cewContainers map[string]models_external.Container,
) map[string]models_handler_deployment.Container {
	containers := make(map[string]models_handler_deployment.Container)
	for reference, stgDepContainer := range stgDepContainers {
		container := models_handler_deployment.Container{DeploymentContainer: stgDepContainer}
		cewContainer, ok := cewContainers[stgDepContainer.Name]
		if ok {
			container.ImageId = cewContainer.ImageID
			container.State = cewContainer.State
		} else {
			logger.Error("missing container") // TODO
		}
		containers[reference] = container
	}
	return containers
}

func getDeploymentState(containersState int) int {
	if containersState == containersStateRunning {
		return models_handler_deployment.StateHealthy
	}
	return models_handler_deployment.StateUnhealthy
}

func getContainersCombinedState(
	deploymentContainers map[string]models_handler_storage.DeploymentContainer,
	existingContainers map[string]models_external.Container,
) int {
	var runningCount int
	for _, deploymentContainer := range deploymentContainers {
		existingContainer, ok := existingContainers[deploymentContainer.Name]
		if !ok {
			return containersStateBroken
		}
		if existingContainer.State == models_external.CewRunningState {
			runningCount++
		}
	}
	switch runningCount {
	case 0:
		return containersStateStopped
	case len(deploymentContainers):
		return containersStateRunning
	}
	return containersStatePartial
}
