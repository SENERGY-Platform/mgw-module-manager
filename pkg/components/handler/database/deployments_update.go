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
	"time"

	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) UpdateDeploymentsEnabledState(ctx context.Context, deployments map[string]bool, timestamp time.Time) (err error) {
	var db sqlDatabase = h.sqlDB
	var tx *sql.Tx
	if len(deployments) > 0 {
		tx, err = h.sqlDB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() {
			err = tx.Rollback()
		}()
		db = tx
	}
	for id, enabled := range deployments {
		_, err = db.ExecContext(
			ctx,
			"UPDATE deployments SET enabled = ?, updated = ? WHERE id = ?",
			enabled,
			timestamp,
			id,
		)
		if err != nil {
			return
		}
	}
	if tx != nil {
		err = tx.Commit()
	}
	return
}

func (h *Handler) UpdateDeploymentEnabledState(ctx context.Context, id string, enabled bool, timestamp time.Time) error {
	return h.UpdateDeploymentsEnabledState(ctx, map[string]bool{id: enabled}, timestamp)
}

func (h *Handler) UpdateDeploymentName(ctx context.Context, id, name string, timestamp time.Time) error {
	_, err := h.sqlDB.ExecContext(
		ctx,
		"UPDATE deployments SET name = ?, updated = ? WHERE id = ?",
		name,
		timestamp,
		id,
	)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) UpdateDeploymentContainerIds(ctx context.Context, containers []models_handler_storage.DeploymentContainer) (err error) {
	var db sqlDatabase = h.sqlDB
	var tx *sql.Tx
	if len(containers) > 0 {
		tx, err = h.sqlDB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() {
			err = tx.Rollback()
		}()
		db = tx
	}
	for _, container := range containers {
		_, err = db.ExecContext(
			ctx,
			"UPDATE dep_containers SET ctr_id = ? WHERE dep_id = ? AND srv_ref = ?",
			container.Id,
			container.DeploymentId,
			container.Reference,
		)
		if err != nil {
			return
		}
	}
	if tx != nil {
		err = tx.Commit()
	}
	return
}

func (h *Handler) UpdateDeployment(
	ctx context.Context,
	deployment models_handler_storage.Deployment,
	hostResources []models_handler_storage.DeploymentHostResource,
	secrets []models_handler_storage.DeploymentSecret,
	userConfigs []models_handler_storage.DeploymentUserConfig,
	globalConfigs []models_handler_storage.DeploymentGlobalConfig,
	files []models_handler_storage.DeploymentFile,
	fileGroups []models_handler_storage.DeploymentFileGroup,
	volumes []models_handler_storage.DeploymentVolume,
	containers []models_handler_storage.DeploymentContainer,
) (err error) {
	tx, err := h.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		err = tx.Rollback()
	}()
	_, err = tx.ExecContext(
		ctx,
		"UPDATE deployments SET mod_source = ?, mod_channel = ?, mod_ver = ?, name = ?, dir = ?, enabled = ?, updated = ? WHERE id = ?",
		deployment.ModuleSource,
		deployment.ModuleChannel,
		deployment.ModuleVersion,
		deployment.Name,
		deployment.DirName,
		deployment.Enabled,
		deployment.Updated,
		deployment.Id,
	)
	if err != nil {
		return
	}
	err = h.deleteDeploymentResourcesAndConfigs(ctx, tx, deployment.Id)
	if err != nil {
		return
	}
	err = h.createDeploymentResourcesAndConfigs(
		ctx,
		tx,
		deployment.Id,
		hostResources,
		secrets,
		userConfigs,
		globalConfigs,
		files,
		fileGroups,
		volumes,
		containers,
	)
	if err != nil {
		return
	}
	err = tx.Commit()
	return
}
