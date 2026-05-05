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

	"github.com/SENERGY-Platform/mgw-module-manager/lib/models/aux_deployments"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/lib/models/results"
	helper_containers "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/containers"
)

func (h *Handler) DeleteDeployments(
	ctx context.Context,
	deploymentId string,
	filter aux_deployments.AuxiliaryDeploymentsFilterWithState,
	allowAll bool,
) ([]aux_deployments.BatchResult, error) {
	if !allowAll && filterEmpty(filter) {
		return nil, nil
	}
	mu := h.mutexes.Get(deploymentId)
	mu.Lock()
	defer mu.Unlock()
	auxDeployments, err := h.readAuxiliaryDeploymentsAndFilterByState(ctx, deploymentId, filter)
	if err != nil {
		return nil, err
	}
	var deleted []string
	var results []aux_deployments.BatchResult
	for id, auxDep := range auxDeployments {
		result := aux_deployments.BatchResult{Id: id}
		err = helper_containers.Remove(ctx, h.containerEngineWrapperClient, auxDep.Container.Name)
		if err != nil {
			result.ErrorResult = models_error.NewErrorResult(err.Error())
		} else {
			deleted = append(deleted, id)
		}
		results = append(results, result)
	}
	err = h.databaseHandler.DeleteAuxiliaryDeployments(ctx, deleted)
	if err != nil {
		return results, err
	}
	return results, nil
}

func (h *Handler) DeleteMutex(deploymentId string) {
	h.mutexes.Delete(deploymentId)
}

func filterEmpty(filter aux_deployments.AuxiliaryDeploymentsFilterWithState) bool {
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
