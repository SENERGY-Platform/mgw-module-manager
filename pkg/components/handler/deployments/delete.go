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
	"maps"
	"slices"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_job "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/job"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) DeleteDeployments(
	ctx context.Context,
	filter pkg_models.DeploymentsFilterWithState,
	allowAll bool,
) ([]lib_models.DeploymentResult, error) {
	if !allowAll && filterEmpty(filter) {
		return nil, nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if allowAll {
		logger.WarnContext(ctx, "delete deployments", slog_keys.Filter, filter, slog_keys.AllowAll, allowAll)
	}
	deployments, err := h.databaseHandler.ReadDeployments(ctx, filter.DeploymentsFilter)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"delete deployments, read from database",
			slog_keys.Filter, filter,
			slog_keys.AllowAll, allowAll,
			slog_keys.Error, err,
		)
		return nil, err
	}
	deploymentIds := slices.Collect(maps.Keys(deployments))
	deploymentsVolumes, deploymentsContainers, err := h.getDeploymentsVolumesAndContainersFromDB(ctx, deploymentIds)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"delete deployments, read volume and container data from database",
			slog_keys.DeploymentIds, deploymentIds,
			slog_keys.Error, err,
		)
		return nil, err
	}
	var results []lib_models.DeploymentResult
	for id, deployment := range deployments {
		result := lib_models.DeploymentResult{ModuleId: deployment.ModuleId, Id: deployment.Id}
		err = h.deleteDeployment(
			ctx,
			deployment.Id,
			deployment.DirName,
			deployment.FilesDirName,
			deploymentsContainers[id],
			deploymentsVolumes[id],
		)
		if err != nil {
			result.ErrorResult = lib_models.NewErrorResult(err.Error())
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
	containers map[string]pkg_models.DeploymentContainerBase,
	volumes map[string]pkg_models.DeploymentVolume,
) error {
	err := h.removeDeploymentEnvironment(ctx, deploymentId, deploymentDirName, deploymentFilesDirName, containers)
	if err != nil {
		logger.ErrorContext(ctx, "delete deployment, remove environment", slog_keys.DeploymentId, deploymentId, slog_keys.Error, err)
		return err
	}
	err = h.removeContainerVolumes(ctx, volumes)
	if err != nil {
		logger.ErrorContext(ctx, "delete deployment, remove volumes", slog_keys.DeploymentId, deploymentId, slog_keys.Error, err)
		return err
	}
	err = h.removeHttpEndpoints(ctx, deploymentId)
	if err != nil {
		logger.ErrorContext(ctx, "delete deployment, remove http endpoints", slog_keys.DeploymentId, deploymentId, slog_keys.Error, err)
		return err
	}
	err = h.databaseHandler.DeleteDeployment(ctx, deploymentId)
	if err != nil {
		logger.ErrorContext(ctx, "delete deployment, remove from database", slog_keys.DeploymentId, deploymentId, slog_keys.Error, err)
		return err
	}
	logger.InfoContext(ctx, "delete deployment", slog_keys.DeploymentId, deploymentId)
	return nil
}

func (h *Handler) removeHttpEndpoints(ctx context.Context, deploymentId string) error {
	jobId, err := h.coreManagerClient.RemoveEndpoints(ctx, external_models.CmEndpointFiler{Ref: deploymentId}, false)
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

func filterEmpty(filter pkg_models.DeploymentsFilterWithState) bool {
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
