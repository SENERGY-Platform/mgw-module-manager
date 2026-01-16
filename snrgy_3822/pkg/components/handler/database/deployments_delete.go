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

package database

import (
	"context"
	"database/sql"

	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
)

func (h *Handler) DeleteDeployment(ctx context.Context, id string) error {
	return h.DeleteDeployments(ctx, []string{id})
}

func (h *Handler) DeleteDeployments(ctx context.Context, ids []string) error {
	ids = helper_slices.RemoveDuplicates(ids)
	_, err := h.sqlDB.ExecContext(
		ctx,
		"DELETE FROM deployments WHERE id IN ("+genQuestionMarks(len(ids))+")",
		helper_slices.ToAny(ids)...,
	)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) deleteDeploymentResourcesAndConfigs(ctx context.Context, tx *sql.Tx, deploymentId string) (err error) {
	_, err = tx.ExecContext(ctx, "DELETE FROM dep_host_resources WHERE dep_id = ?", deploymentId)
	if err != nil {
		return
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM dep_secrets WHERE dep_id = ?", deploymentId)
	if err != nil {
		return
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM dep_configs WHERE dep_id = ?", deploymentId)
	if err != nil {
		return
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM dep_global_configs WHERE dep_id = ?", deploymentId)
	return
}
