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

package handler_database

import (
	"context"
	"database/sql"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

func (h *Handler) CreateDeployment(
	ctx context.Context,
	deployment models_handler_database.Deployment,
	hostResources []models_handler_database.DeploymentHostResource,
	secrets []models_handler_database.DeploymentSecret,
	userConfigs []models_handler_database.DeploymentUserConfig,
	globalConfigs []models_handler_database.DeploymentGlobalConfig,
	files []models_handler_database.DeploymentFile,
	fileGroups []models_handler_database.DeploymentFileGroup,
	volumes []models_handler_database.DeploymentVolume,
	containers []models_handler_database.DeploymentContainer,
) (err error) {
	tx, err := h.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(
		ctx,
		"INSERT INTO deployments (id, mod_id, mod_source, mod_channel, mod_ver, dir, files_dir, enabled, created, updated) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		deployment.Id,
		deployment.ModuleId,
		deployment.ModuleSource,
		deployment.ModuleChannel,
		deployment.ModuleVersion,
		deployment.DirName,
		deployment.FilesDirName,
		deployment.Enabled,
		deployment.Created,
		deployment.Updated,
	)
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

func (h *Handler) createDeploymentResourcesAndConfigs(
	ctx context.Context,
	tx *sql.Tx,
	deploymentId string,
	hostResources []models_handler_database.DeploymentHostResource,
	secrets []models_handler_database.DeploymentSecret,
	userConfigs []models_handler_database.DeploymentUserConfig,
	globalConfigs []models_handler_database.DeploymentGlobalConfig,
	files []models_handler_database.DeploymentFile,
	fileGroups []models_handler_database.DeploymentFileGroup,
	volumes []models_handler_database.DeploymentVolume,
	containers []models_handler_database.DeploymentContainer,
) (err error) {
	for _, hostResource := range hostResources {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO dep_host_resources (dep_id, ref, res_id) VALUES (?, ?, ?)",
			deploymentId,
			hostResource.Reference,
			hostResource.Id,
		)
		if err != nil {
			return
		}
	}
	for _, secret := range secrets {
		for _, item := range secret.Items {
			_, err = tx.ExecContext(
				ctx,
				"INSERT INTO dep_secrets (dep_id, ref, sec_id, item, as_mount, as_env) VALUES (?, ?, ?, ?, ?, ?)",
				deploymentId,
				secret.Reference,
				secret.Id,
				item.Name,
				item.AsMount,
				item.AsEnv,
			)
			if err != nil {
				return
			}
		}
	}
	for _, config := range userConfigs {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO dep_configs (id, dep_id, ref, data_type, is_list) VALUES (?, ?, ?, ?, ?)",
			config.Id,
			config.DeploymentId,
			config.Reference,
			config.DataType,
			config.IsSlice,
		)
		if err != nil {
			return
		}
		err = createConfigValues(ctx, tx, "dep_config_values", config.Id, config.Value)
		if err != nil {
			return
		}
	}
	for _, globalConfig := range globalConfigs {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO dep_global_configs (dep_id, ref, c_id) VALUES (?, ?, ?)",
			deploymentId,
			globalConfig.Reference,
			globalConfig.Id,
		)
		if err != nil {
			return
		}
	}
	for _, file := range files {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO dep_files (dep_id, ref, data) VALUES (?, ?, ?)",
			deploymentId,
			file.Reference,
			file.Data,
		)
		if err != nil {
			return
		}
	}
	for _, fileGroup := range fileGroups {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO dep_file_groups (id, dep_id, ref) VALUES (?, ?, ?)",
			fileGroup.Id,
			deploymentId,
			fileGroup.Reference,
		)
		if err != nil {
			return
		}
		err = createFileGroupFiles(ctx, tx, fileGroup.Id, fileGroup.Files)
		if err != nil {
			return
		}
	}
	for _, volume := range volumes {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO dep_volumes (dep_id, ref, name) VALUES (?, ?, ?)",
			deploymentId,
			volume.Reference,
			volume.Name,
		)
		if err != nil {
			return
		}
	}
	for _, container := range containers {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO dep_containers (dep_id, name, srv_ref, alias) VALUES (?, ?, ?, ?)",
			deploymentId,
			container.Name,
			container.Reference,
			container.Alias,
		)
		if err != nil {
			return
		}
	}
	return
}

func createFileGroupFiles(ctx context.Context, tx *sql.Tx, groupId string, files []models_handler_database.DeploymentFileGroupFile) (err error) {
	for _, file := range files {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO dep_file_group_files (g_id, path, format, data) VALUES (?, ?, ?, ?)",
			groupId,
			file.Path,
			file.Format,
			file.Data,
		)
		if err != nil {
			return
		}
	}
	return
}
