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

	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
)

func (h *Handler) EnableDeployments(ctx context.Context, moduleIds []string) ([]string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	deployments, err := h.databaseHandler.ReadDeployments(ctx, pkg_models.DeploymentsFilter{
		ModuleIds: moduleIds,
	})
	if err != nil {
		logger.Error(
			"enable deployments, read from database",
			slog_keys.Filter, moduleIds,
			slog_keys.Error, err,
		)
		return nil, err
	}
	ids := slices.Collect(maps.Keys(deployments))
	err = h.databaseHandler.UpdateDeploymentsEnabledState(ctx, ids, true)
	if err != nil {
		logger.Error(
			"enable deployments, write to database",
			slog_keys.DeploymentIds, ids,
			slog_keys.Error, err,
		)
		return nil, err
	}
	return ids, nil
}

func (h *Handler) DisableDeployments(ctx context.Context, moduleIds []string) ([]string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	deployments, err := h.databaseHandler.ReadDeployments(ctx, pkg_models.DeploymentsFilter{
		ModuleIds: moduleIds,
	})
	if err != nil {
		logger.Error(
			"disable deployments, read from database",
			slog_keys.Filter, moduleIds,
			slog_keys.Error, err,
		)
		return nil, err
	}
	ids := slices.Collect(maps.Keys(deployments))
	err = h.databaseHandler.UpdateDeploymentsEnabledState(ctx, ids, false)
	if err != nil {
		logger.Error(
			"disable deployments, write to database",
			slog_keys.DeploymentIds, ids,
			slog_keys.Error, err,
		)
		return nil, err
	}
	return ids, nil
}
