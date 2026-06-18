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

	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

func (h *Handler) UpdateDeploymentsEnabledState(ctx context.Context, deploymentIds []string, state bool) error {
	var db sqlDatabase = h.sqlDB
	var tx *sql.Tx
	var err error
	if len(deploymentIds) > 0 {
		tx, err = h.sqlDB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		db = tx
	}
	for _, id := range deploymentIds {
		_, err = db.ExecContext(
			ctx,
			"UPDATE deployments SET enabled = ? WHERE id = ?",
			state,
			id,
		)
		if err != nil {
			return err
		}
	}
	if tx != nil {
		err = tx.Commit()
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) UpdateDeploymentEnabledState(ctx context.Context, id string, state bool) error {
	return h.UpdateDeploymentsEnabledState(ctx, []string{id}, state)
}

func (h *Handler) UpdateDeployment(
	ctx context.Context,
	deployment pkg_models.DeploymentBase,
	hostResources []pkg_models.DeploymentHostResource,
	secrets []pkg_models.DeploymentSecret,
	userConfigs []pkg_models.DeploymentUserConfig,
	globalConfigs []pkg_models.DeploymentGlobalConfig,
	files []pkg_models.DeploymentFile,
	fileGroups []pkg_models.DeploymentFileGroup,
	volumes []pkg_models.DeploymentVolume,
	containers []pkg_models.DeploymentContainerBase,
) error {
	tx, err := h.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(
		ctx,
		"UPDATE deployments SET mod_source = ?, mod_channel = ?, mod_ver = ?, dir = ?, files_dir = ?, enabled = ?, updated = ? WHERE id = ?",
		deployment.ModuleSource,
		deployment.ModuleChannel,
		deployment.ModuleVersion,
		deployment.DirName,
		deployment.FilesDirName,
		deployment.Enabled,
		deployment.Updated,
		deployment.Id,
	)
	if err != nil {
		return err
	}
	err = h.deleteDeploymentResourcesAndConfigs(ctx, tx, deployment.Id)
	if err != nil {
		return err
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
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) UpdateDeploymentContainerNames(ctx context.Context, containers []pkg_models.DeploymentContainerBase) error {
	var db sqlDatabase = h.sqlDB
	var tx *sql.Tx
	var err error
	if len(containers) > 0 {
		tx, err = h.sqlDB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		db = tx
	}
	for _, container := range containers {
		_, err = db.ExecContext(
			ctx,
			"UPDATE dep_containers SET name = ? WHERE dep_id = ? AND srv_ref = ?",
			container.Name,
			container.DeploymentId,
			container.Reference,
		)
		if err != nil {
			return err
		}
	}
	if tx != nil {
		err = tx.Commit()
		if err != nil {
			return err
		}
	}
	return nil
}
