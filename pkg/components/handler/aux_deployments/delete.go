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

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_containers "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/containers"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
)

func (h *Handler) DeleteDeployments(
	ctx context.Context,
	deploymentId string,
	filter lib_models.AuxiliaryDeploymentsFilterWithState,
	allowAll bool,
) ([]lib_models.AuxiliaryDeploymentBatchResult, error) {
	if !allowAll && filterEmpty(filter) {
		return nil, nil
	}
	mu := h.mutexes.Get(deploymentId)
	mu.Lock()
	defer mu.Unlock()
	if allowAll {
		logger.Warn(
			"delete auxiliary deployments",
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Filter, filter,
			slog_keys.AllowAll, allowAll,
		)
	}
	auxDeployments, err := h.readAuxiliaryDeploymentsAndFilterByState(ctx, deploymentId, filter)
	if err != nil {
		logger.Error(
			"delete auxiliary deployments, read from database",
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Filter, filter,
			slog_keys.Error, err,
		)
		return nil, err
	}
	var deleted []string
	var results []lib_models.AuxiliaryDeploymentBatchResult
	for id, auxDep := range auxDeployments {
		result := lib_models.AuxiliaryDeploymentBatchResult{Id: id}
		err = helper_containers.Remove(ctx, h.containerEngineWrapperClient, auxDep.Container.Name)
		if err != nil {
			logger.Error(
				"delete auxiliary deployments, remove container",
				slog_keys.DeploymentId, deploymentId,
				slog_keys.AuxDeploymentId, id,
				slog_keys.ContainerName, auxDep.Container.Name,
				slog_keys.Error, err,
			)
			result.ErrorResult = lib_models.NewErrorResult(err.Error())
		} else {
			deleted = append(deleted, id)
		}
		results = append(results, result)
	}
	err = h.databaseHandler.DeleteAuxiliaryDeployments(ctx, deleted)
	if err != nil {
		logger.Error(
			"delete auxiliary deployments, remove from database",
			slog_keys.DeploymentId, deploymentId,
			slog_keys.AuxDeploymentIds, deleted,
			slog_keys.Error, err,
		)
		return results, err
	}
	return results, nil
}

func (h *Handler) DeleteMutex(deploymentId string) {
	h.mutexes.Delete(deploymentId)
}

func filterEmpty(filter lib_models.AuxiliaryDeploymentsFilterWithState) bool {
	switch {
	case filter.State != "":
		return false
	case filter.Enabled != 0:
		return false
	case filter.Image != "":
		return false
	case filter.Recreate != 0:
		return false
	case len(filter.Ids) > 0:
		return false
	case len(filter.Labels) > 0:
		return false
	}
	return true
}
