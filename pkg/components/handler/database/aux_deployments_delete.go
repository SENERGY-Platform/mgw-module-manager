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

package handler_database

import (
	"context"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

func (h *Handler) DeleteAuxiliaryDeployment(ctx context.Context, deploymentId, auxDeploymentId string) error {
	return h.DeleteAuxiliaryDeployments(ctx, deploymentId, models_handler_database.AuxiliaryDeploymentsFilter{
		Ids: []string{auxDeploymentId},
	})
}

func (h *Handler) DeleteAuxiliaryDeployments(ctx context.Context,
	deploymentId string,
	filter models_handler_database.AuxiliaryDeploymentsFilter,
) error {
	fc, val := genAuxiliaryDeploymentsFilter(deploymentId, filter)
	_, err := h.sqlDB.ExecContext(
		ctx,
		"DELETE FROM aux_deployments"+fc+";",
		val...,
	)
	if err != nil {
		return err
	}
	return nil
}
