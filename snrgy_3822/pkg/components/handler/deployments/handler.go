/*
 * Copyright 2025 InfAI (CC SES)
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
	"os"
	"slices"
	"sync"
	"time"

	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_deployment "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployment"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

type Config struct {
	WorkDirPath     string        `json:"work_dir_path" env_var:"DEPLOYMENTS_HANDLER_WORK_DIR_PATH"`
	PathEscapeDepth int           `json:"path_escape_depth" env_var:"PATH_ESCAPE_DEPTH"`
	JobPollInterval time.Duration `json:"job_poll_interval" env_var:"DEPLOYMENTS_HANDLER_JOB_POLL_INTERVAL"`
}

type Handler struct {
	storageHdl storageHandler
	cewClient  containerEngineWrapperClient
	hmClient   hostManagerClient
	smClient   secretManagerClient
	config     Config
	mu         sync.RWMutex
}

func New(storageHdl storageHandler, cewClient containerEngineWrapperClient, hmClient hostManagerClient, smClient secretManagerClient, config Config) *Handler {
	return &Handler{
		storageHdl: storageHdl,
		cewClient:  cewClient,
		hmClient:   hmClient,
		smClient:   smClient,
		config:     config,
	}
}

func (h *Handler) Init() error {
	return os.MkdirAll(h.config.WorkDirPath, 0775)
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

func (h *Handler) GetDeployments(ctx context.Context, filter models_handler_deployment.DeploymentsFilter) (map[string]models_handler_deployment.Deployment, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.getDeployments(ctx, filter)
}

func (h *Handler) getDeployments(ctx context.Context, filter models_handler_deployment.DeploymentsFilter) (map[string]models_handler_deployment.Deployment, error) {
	stgDeps, err := h.storageHdl.ReadDeployments(ctx, filter.DeploymentsFilter)
	if err != nil {
		return nil, err
	}
	depIds := slices.Collect(maps.Keys(stgDeps))
	stgDepsHostResources, err := h.storageHdl.ReadDeploymentsHostResources(
		ctx,
		models_handler_storage.DeploymentsHostResourcesFilter{DeploymentIds: depIds},
	)
	if err != nil {
		return nil, err
	}
	stgDepsSecrets, err := h.storageHdl.ReadDeploymentsSecrets(
		ctx,
		models_handler_storage.DeploymentsSecretsFilter{DeploymentIds: depIds},
	)
	if err != nil {
		return nil, err
	}
	stgDepsConfigs, err := h.storageHdl.ReadDeploymentsUserConfigs(ctx, depIds)
	if err != nil {
		return nil, err
	}
	stgDepsContainers, err := h.storageHdl.ReadDeploymentsContainers(ctx, depIds)
	if err != nil {
		return nil, err
	}
	cewContainersMap, cewErr := h.getCewContainers(ctx, stgDepsContainers)
	if cewErr != nil {
		logger.Error("error getting containers") // TODO
	}
	deployments := make(map[string]models_handler_deployment.Deployment)
	for _, stgDep := range stgDeps {
		deployment := models_handler_deployment.Deployment{
			Deployment:    stgDep,
			Containers:    newContainers(stgDepsContainers[stgDep.Id], cewContainersMap),
			HostResources: stgDepsHostResources[stgDep.Id],
			Secrets:       stgDepsSecrets[stgDep.Id],
			Configs:       stgDepsConfigs[stgDep.Id],
		}
		if cewErr != nil {
			deployment.State = models_handler_deployment.StateNotAvailable
		} else {
			deployment.State = getHealthState(deployment.Containers)
		}
		deployments[stgDep.Id] = deployment
	}
	return deployments, nil
}

func (h *Handler) getCewContainers(ctx context.Context, stgDepsContainers map[string][]models_handler_storage.DeploymentContainer) (map[string]models_external.Container, error) {
	var ctrIds []string
	for _, stgDepContainers := range stgDepsContainers {
		ctrIds = append(ctrIds, helper_slices.CollectSliceFunc(stgDepContainers, func(item models_handler_storage.DeploymentContainer) string {
			return item.Id
		})...)
	}
	cewContainers, err := h.cewClient.GetContainers(ctx, models_external.ContainersFilter{Ids: ctrIds})
	if err != nil {
		return nil, err
	}
	cewContainersMap := maps.Collect(helper_slices.AllFunc(cewContainers, func(item models_external.Container) string {
		return item.ID
	}))
	return cewContainersMap, nil
}

func newContainers(stgDepContainers []models_handler_storage.DeploymentContainer, cewContainers map[string]models_external.Container) []models_handler_deployment.Container {
	var containers []models_handler_deployment.Container
	for _, stgDepContainer := range stgDepContainers {
		container := models_handler_deployment.Container{DeploymentContainer: stgDepContainer}
		cewContainer, ok := cewContainers[stgDepContainer.Id]
		if ok {
			container.ImageId = cewContainer.ImageID
			container.State = cewContainer.State
		} else {
			logger.Error("missing container") // TODO
		}
		containers = append(containers, container)
	}
	return containers
}

func getHealthState(containers []models_handler_deployment.Container) string {
	for _, container := range containers {
		if container.State != models_external.CewRunningState {
			return models_handler_deployment.StateUnhealthy
		}
	}
	return models_handler_deployment.StateHealthy
}
