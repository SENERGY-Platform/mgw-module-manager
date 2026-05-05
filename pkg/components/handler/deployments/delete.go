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
	"errors"
	"maps"
	"slices"

	models_error "github.com/SENERGY-Platform/mgw-module-manager/lib/models/results"
	lib_models_service "github.com/SENERGY-Platform/mgw-module-manager/lib/models/service"
	helper_job "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/job"
	models_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/deployments"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) DeleteDeployments(
	ctx context.Context,
	filter models_deployments.DeploymentsFilterWithState,
	allowAll bool,
) ([]lib_models_service.DeploymentResult, error) {
	if !allowAll && filterEmpty(filter) {
		return nil, nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	deployments, err := h.databaseHandler.ReadDeployments(ctx, filter.DeploymentsFilter)
	if err != nil {
		return nil, err
	}
	deploymentsVolumes, deploymentsContainers, err := h.getDeploymentsVolumesAndContainersFromDB(ctx, slices.Collect(maps.Keys(deployments)))
	if err != nil {
		return nil, err
	}
	var results []lib_models_service.DeploymentResult
	for id, deployment := range deployments {
		result := lib_models_service.DeploymentResult{ModuleId: deployment.ModuleId, Id: deployment.Id}
		err = h.deleteDeployment(
			ctx,
			deployment.Id,
			deployment.DirName,
			deployment.FilesDirName,
			deploymentsContainers[id],
			deploymentsVolumes[id],
		)
		if err != nil {
			result.ErrorResult = models_error.NewErrorResult(err.Error())
		}
		results = append(results, result)
	}
	return results, nil
}

func (h *Handler) deleteDeployment(
	ctx context.Context,
	deploymentId string,
	deploymentDirName string,
	deploymentFilesDirName string,
	containers map[string]models_deployments.ContainerBase,
	volumes map[string]models_deployments.DeploymentVolume,
) error {
	err := h.removeDeploymentEnvironment(ctx, deploymentId, deploymentDirName, deploymentFilesDirName, containers)
	if err != nil {
		return err
	}
	err = h.removeContainerVolumes(ctx, volumes)
	if err != nil {
		return err
	}
	err = h.removeHttpEndpoints(ctx, deploymentId)
	if err != nil {
		return err
	}
	return h.databaseHandler.DeleteDeployment(ctx, deploymentId)
}

func (h *Handler) removeHttpEndpoints(ctx context.Context, deploymentId string) error {
	jobId, err := h.coreManagerClient.RemoveEndpoints(ctx, models_external.CmEndpointFiler{Ref: deploymentId}, false)
	if err != nil {
		return err
	}
	job, err := helper_job.Await(ctx, h.coreManagerClient, jobId, h.config.JobPollInterval)
	if err != nil {
		return err
	}
	if job.Error != nil {
		return errors.New(job.Error.Message)
	}
	return nil
}

func filterEmpty(filter models_deployments.DeploymentsFilterWithState) bool {
	switch {
	case len(filter.Ids) > 0:
		return false
	case len(filter.ModuleIds) > 0:
		return false
	case filter.Enabled != 0:
		return false
	case filter.State != 0:
		return false
	}
	return true
}
