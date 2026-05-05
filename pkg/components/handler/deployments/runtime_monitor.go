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
	"time"

	models_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/deployments"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) RuntimeMonitor(ctx context.Context) {
	timer := time.NewTimer(h.config.RuntimeMonitorStartupDelay)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			h.checkDeployments(ctx)
			timer.Reset(h.config.RuntimeMonitorLoopDelay)
		case <-ctx.Done():
			return
		}
	}
}

func (h *Handler) checkDeployments(ctx context.Context) {
	deployments, deploymentsContainers, deploymentsMountSecrets, cewContainersMap, err := h.getCurrentRuntimeData(ctx)
	if err != nil {
		logger.Error(err.Error()) // TODO
		return
	}
	filteredDeployments := h.runtimeMonitorJobsFilter(deployments)
	for id, deployment := range filteredDeployments {
		deploymentContainers := deploymentsContainers[id]
		state := getContainersCombinedState(deploymentContainers, cewContainersMap)
		if state == containersStateBroken || state == containersStateUnhealthy {
			continue
		}
		if deployment.Enabled {
			if state == containersStateRunning {
				continue
			}
			h.runtimeMonitorJobsAdd(id)
			go h.startDeployment(ctx, id, deploymentContainers, deploymentsMountSecrets[id])
		} else {
			if state == containersStateStopped {
				continue
			}
			h.runtimeMonitorJobsAdd(id)
			go h.stopDeployment(ctx, id, deploymentContainers, len(deploymentsMountSecrets[id]) > 0)
		}
	}
}

func (h *Handler) getCurrentRuntimeData(ctx context.Context) (
	map[string]models_deployments.DeploymentBase,
	map[string]map[string]models_deployments.ContainerBase,
	map[string]map[string]models_deployments.DeploymentSecret,
	map[string]models_external.Container,
	error,
) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	deployments, err := h.databaseHandler.ReadDeployments(ctx, models_deployments.DeploymentsFilter{})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	deploymentsContainers, err := h.databaseHandler.ReadDeploymentsContainers(ctx, slices.Collect(maps.Keys(deployments)))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	var enabledDeploymentIds []string
	for id, deployment := range deployments {
		if deployment.Enabled {
			enabledDeploymentIds = append(enabledDeploymentIds, id)
		}
	}
	deploymentsMountSecrets, err := h.databaseHandler.ReadDeploymentsSecrets(ctx, models_deployments.DeploymentsSecretsFilter{
		DeploymentIds: enabledDeploymentIds,
		AsMount:       1,
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	cewContainersMap, err := h.getCewContainers(ctx, deploymentsContainers)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return deployments, deploymentsContainers, deploymentsMountSecrets, cewContainersMap, nil
}

func (h *Handler) startDeployment(
	ctx context.Context,
	deploymentId string,
	deploymentContainers map[string]models_deployments.ContainerBase,
	deploymentMountSecrets map[string]models_deployments.DeploymentSecret,
) {
	var err error
	defer func() {
		if err != nil && len(deploymentMountSecrets) > 0 {
			e, _ := h.secretManagerClient.CleanPathVariants(context.Background(), deploymentId)
			if e != nil {
				logger.Error(e.Error()) // TODO
			}
		}
		h.runtimeMonitorJobsRemove(deploymentId)
	}()
	err = h.loadDeploymentMountSecrets(ctx, deploymentId, deploymentMountSecrets)
	if err != nil {
		logger.Error(err.Error()) // TODO
		return
	}
	err = h.startContainers(ctx, deploymentContainers)
	if err != nil {
		logger.Error(err.Error()) // TODO
		return
	}
}

func (h *Handler) loadDeploymentMountSecrets(
	ctx context.Context,
	deploymentId string,
	deploymentMountSecrets map[string]models_deployments.DeploymentSecret,
) error {
	for _, secret := range deploymentMountSecrets {
		for _, item := range secret.Items {
			if item.AsEnv {
				continue
			}
			req := models_external.SecretVariantRequest{
				ID:        secret.Id,
				Item:      nil,
				Reference: deploymentId,
			}
			if item.Name != "" {
				req.Item = &item.Name
			}
			err, _ := h.secretManagerClient.LoadPathVariant(ctx, req)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) stopDeployment(
	ctx context.Context,
	deploymentId string,
	deploymentContainers map[string]models_deployments.ContainerBase,
	hasMountSecrets bool,
) {
	defer h.runtimeMonitorJobsRemove(deploymentId)
	if hasMountSecrets {
		err, _ := h.secretManagerClient.CleanPathVariants(ctx, deploymentId)
		if err != nil {
			logger.Error(err.Error()) // TODO
		}
	}
	err := h.stopContainers(ctx, deploymentContainers)
	if err != nil {
		logger.Error(err.Error()) // TODO
	}
}

func (h *Handler) runtimeMonitorJobsFilter(deployments map[string]models_deployments.DeploymentBase) map[string]models_deployments.DeploymentBase {
	h.runtimeMonitorJobsMu.RLock()
	defer h.runtimeMonitorJobsMu.RUnlock()
	filteredDeployments := make(map[string]models_deployments.DeploymentBase)
	for id, deployment := range deployments {
		_, ok := h.runtimeMonitorJobs[id]
		if !ok {
			filteredDeployments[id] = deployment
		}
	}
	return filteredDeployments
}

func (h *Handler) runtimeMonitorJobsAdd(id string) {
	h.runtimeMonitorJobsMu.Lock()
	defer h.runtimeMonitorJobsMu.Unlock()
	h.runtimeMonitorJobs[id] = struct{}{}
}

func (h *Handler) runtimeMonitorJobsRemove(id string) {
	h.runtimeMonitorJobsMu.Lock()
	defer h.runtimeMonitorJobsMu.Unlock()
	delete(h.runtimeMonitorJobs, id)
}
