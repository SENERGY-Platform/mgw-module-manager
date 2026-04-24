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
	"maps"
	"slices"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/aux_deployments"
)

func (h *Handler) EnableDeployments(
	ctx context.Context,
	deploymentId string,
	filter models_handler_aux_deployments.AuxiliaryDeploymentsFilter,
) ([]string, error) {
	mu := h.mutexes.Get(deploymentId)
	mu.RLock()
	defer mu.RUnlock()
	auxDeployments, err := h.readAuxiliaryDeploymentsAndFilterByState(ctx, deploymentId, filter)
	if err != nil {
		return nil, err
	}
	ids := slices.Collect(maps.Keys(auxDeployments))
	err = h.databaseHandler.UpdateAuxiliaryDeploymentsEnabledState(ctx, ids, true)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (h *Handler) DisableDeployments(
	ctx context.Context,
	deploymentId string,
	filter models_handler_aux_deployments.AuxiliaryDeploymentsFilter,
) ([]string, error) {
	mu := h.mutexes.Get(deploymentId)
	mu.RLock()
	defer mu.RUnlock()
	auxDeployments, err := h.readAuxiliaryDeploymentsAndFilterByState(ctx, deploymentId, filter)
	if err != nil {
		return nil, err
	}
	ids := slices.Collect(maps.Keys(auxDeployments))
	err = h.databaseHandler.UpdateAuxiliaryDeploymentsEnabledState(ctx, ids, false)
	if err != nil {
		return nil, err
	}
	return ids, nil
}
