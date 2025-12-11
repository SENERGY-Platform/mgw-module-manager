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
	"fmt"

	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	models_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/storage"
)

func (h *Handler) CreateDeployment(ctx context.Context, deployment models_storage.Deployment, hostResources []models_storage.DeploymentHostResource, secrets []models_storage.DeploymentSecret, configs []models_storage.DeploymentConfig) (err error) {
	tx, err := h.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		err = tx.Rollback()
	}()
	_, err = tx.ExecContext(
		ctx,
		"INSERT INTO deployments (id, mod_id, mod_source, mod_channel, mod_ver, name, dir, enabled, created, updated) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		deployment.Id,
		deployment.Module.Id,
		deployment.Module.Source,
		deployment.Module.Channel,
		deployment.Module.Version,
		deployment.Name,
		deployment.DirName,
		deployment.Enabled,
		deployment.Created,
		deployment.Updated,
	)
	if err != nil {
		return
	}
	err = h.createDeploymentResourcesAndConfigs(ctx, tx, deployment.Id, hostResources, secrets, configs)
	if err != nil {
		return
	}
	err = tx.Commit()
	return
}

func (h *Handler) createDeploymentResourcesAndConfigs(ctx context.Context, tx *sql.Tx, deploymentId string, hostResources []models_storage.DeploymentHostResource, secrets []models_storage.DeploymentSecret, configs []models_storage.DeploymentConfig) error {
	var err error
	for _, hostResource := range hostResources {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO dep_host_resources (dep_id, ref, res_id) VALUES (?, ?, ?)",
			deploymentId,
			hostResource.Reference,
			hostResource.Id,
		)
		if err != nil {
			return err
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
				return err
			}
		}
	}
	for _, config := range configs {
		if config.IsSlice {
			var colName string
			var value any
			switch config.DataType {
			case models_storage.StringType:
				colName = "v_string"
				value = config.String
			case models_storage.Int64Type:
				colName = "v_int"
				value = config.Int64
			case models_storage.Float64Type:
				colName = "v_float"
				value = config.Float64
			case models_storage.BoolType:
				colName = "v_bool"
				value = config.Bool
			}
			_, err = tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO dep_configs (dep_id, ref, %s) VALUES (?, ?, ?)", colName), deploymentId, config.Reference, value)
			if err != nil {
				return err
			}
		} else {
			var colName string
			var values []any
			switch config.DataType {
			case models_storage.StringType:
				colName = "v_string"
				values = helper_slices.ToAny(config.StringSlice)
			case models_storage.Int64Type:
				colName = "v_int"
				values = helper_slices.ToAny(config.Int64Slice)
			case models_storage.Float64Type:
				colName = "v_float"
				values = helper_slices.ToAny(config.Float64Slice)
			case models_storage.BoolType:
				colName = "v_bool"
				values = helper_slices.ToAny(config.BoolSlice)
			}
			stmt := fmt.Sprintf("INSERT INTO dep_list_configs (dep_id, ref, ord, %s) VALUES (?, ?, ?, ?)", colName)
			for i, value := range values {
				_, err = tx.ExecContext(ctx, stmt, deploymentId, config.Reference, i, value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
