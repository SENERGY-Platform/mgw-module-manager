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
	"errors"
	"fmt"
	"maps"
	"slices"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_containers "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/containers"
	helper_errors "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/errors"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
)

// RestartDeployments restarts the existing containers of the enabled deployments of the selected modules without
// recreating them. Results are ordered by module ID.
func (h *Handler) RestartDeployments(
	ctx context.Context,
	selectedModules map[string]pkg_models.Module,
) ([]lib_models.DeploymentResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	moduleIds := slices.Sorted(maps.Keys(selectedModules))
	deployments, err := h.databaseHandler.ReadDeployments(ctx, pkg_models.DeploymentsFilter{
		ModuleIds: moduleIds,
	})
	if err != nil {
		logger.ErrorContext(ctx, "restart deployments, read from database", slog_keys.ModuleIds, moduleIds, slog_keys.Error, err)
		return nil, err
	}
	deploymentIds := slices.Collect(maps.Keys(deployments))
	deploymentsContainers, err := h.databaseHandler.ReadDeploymentsContainers(ctx, deploymentIds)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"restart deployments, read containers from database",
			slog_keys.DeploymentIds, deploymentIds,
			slog_keys.Error, err,
		)
		return nil, err
	}
	deploymentsByModule := make(map[string]pkg_models.DeploymentBase, len(deployments))
	for _, deployment := range deployments {
		deploymentsByModule[deployment.ModuleId] = deployment
	}
	results := make([]lib_models.DeploymentResult, 0, len(moduleIds))
	for _, moduleId := range moduleIds {
		result := lib_models.DeploymentResult{ModuleId: moduleId}
		deployment, ok := deploymentsByModule[moduleId]
		if !ok {
			result.ErrorResult = lib_models.NewErrorResult("deployment not found")
			results = append(results, result)
			continue
		}
		result.Id = deployment.Id
		if err = h.restartDeployment(ctx, deployment, deploymentsContainers[deployment.Id]); err != nil {
			result.ErrorResult = lib_models.NewErrorResult(err.Error())
		}
		results = append(results, result)
	}
	return results, nil
}

func (h *Handler) restartDeployment(
	ctx context.Context,
	deployment pkg_models.DeploymentBase,
	containers map[string]pkg_models.DeploymentContainerBase,
) error {
	if !deployment.Enabled {
		return errors.New("deployment is disabled")
	}
	// The runtime monitor must neither start the deployment while its containers are down nor count this restart.
	if !h.runtimeMonitorJobsTryAdd(deployment.Id) {
		return errors.New("deployment is being started or stopped")
	}
	defer h.runtimeMonitorJobsRemove(deployment.Id)
	err := h.restartContainers(ctx, containers)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"restart deployment, restart containers",
			slog_keys.ModuleId, deployment.ModuleId,
			slog_keys.DeploymentId, deployment.Id,
			slog_keys.Error, err,
		)
		return err
	}
	return nil
}

func (h *Handler) restartContainers(
	ctx context.Context,
	deploymentContainers map[string]pkg_models.DeploymentContainerBase,
) error {
	var errs []error
	for _, reference := range slices.Sorted(maps.Keys(deploymentContainers)) {
		container := deploymentContainers[reference]
		err := helper_containers.Restart(ctx, h.containerEngineWrapperClient, container.Name, h.config.JobPollInterval)
		if err != nil {
			errs = append(errs, fmt.Errorf("'%s' %w", container.Reference, err))
		}
	}
	if len(errs) > 0 {
		return helper_errors.Join(errs...)
	}
	return nil
}
