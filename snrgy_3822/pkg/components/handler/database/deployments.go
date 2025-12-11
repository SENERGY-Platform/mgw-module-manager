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
	"errors"
	"strings"
	"time"

	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/storage"
)

func (h *Handler) Deployment(ctx context.Context, id string) (models_storage.Deployment, error) {
	deployments, err := h.Deployments(ctx, models_storage.DeploymentsFilter{Ids: []string{id}})
	if err != nil {
		return models_storage.Deployment{}, err
	}
	if len(deployments) == 0 {
		return models_storage.Deployment{}, models_error.NotFoundErr
	}
	return deployments[id], nil
}

func (h *Handler) Deployments(ctx context.Context, filter models_storage.DeploymentsFilter) (map[string]models_storage.Deployment, error) {
	fc, val := genDeploymentsFilter(filter)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT id, mod_id, mod_source, mod_channel, mod_ver, name, dir, enabled, created, updated FROM deployments"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deps := make(map[string]models_storage.Deployment)
	for rows.Next() {
		var dep models_storage.Deployment
		var ct, ut []uint8
		err = rows.Scan(
			&dep.Id,
			&dep.Module.Id,
			&dep.Module.Source,
			&dep.Module.Channel,
			&dep.Module.Version,
			&dep.Name,
			&dep.DirName,
			&dep.Enabled,
			&ct,
			&ut,
		)
		if err != nil {
			return nil, err
		}
		if dep.Created, err = time.Parse(timeLayout, string(ct)); err != nil {
			return nil, err
		}
		if dep.Updated, err = time.Parse(timeLayout, string(ut)); err != nil {
			return nil, err
		}
		deps[dep.Id] = dep
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return deps, nil
}

func (h *Handler) CreateDeployment(ctx context.Context, deployment models_storage.Deployment) error {
	return h.CreateDeployments(ctx, []models_storage.Deployment{deployment})
}

func (h *Handler) CreateDeployments(ctx context.Context, deployments []models_storage.Deployment) (err error) {
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
	for _, deployment := range deployments {
		_, err = db.ExecContext(
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
	}
	if tx != nil {
		err = tx.Commit()
	}
	return
}

func (h *Handler) UpdateDeployment(ctx context.Context, deployment models_storage.Deployment) error {
	return h.UpdateDeployments(ctx, []models_storage.Deployment{deployment})
}

func (h *Handler) UpdateDeployments(ctx context.Context, deployments []models_storage.Deployment) (err error) {
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
	for _, deployment := range deployments {
		_, err = db.ExecContext(
			ctx,
			"UPDATE deployments SET mod_source = ?, mod_channel = ?, mod_ver = ?, name = ?, dir = ?, enabled = ?, updated = ? WHERE id = ?",
			deployment.Module.Source,
			deployment.Module.Channel,
			deployment.Module.Version,
			deployment.Name,
			deployment.DirName,
			deployment.Enabled,
			deployment.Updated,
			deployment.Id,
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

func (h *Handler) DeploymentContainers(ctx context.Context, deploymentId string) (map[string]models_storage.DeploymentContainer, error) {
	deploymentsContainers, err := h.DeploymentsContainers(ctx, []string{deploymentId})
	if err != nil {
		return nil, err
	}
	if len(deploymentsContainers) == 0 {
		return nil, nil
	}
	return deploymentsContainers[deploymentId], nil
}

func (h *Handler) DeploymentsContainers(ctx context.Context, deploymentIds []string) (map[string]map[string]models_storage.DeploymentContainer, error) {
	fc, val := genDeploymentsContainersFilter(deploymentIds)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT dep_id, ctr_id, srv_ref, alias, 'order' FROM dep_containers"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	depContainers := make(map[string]map[string]models_storage.DeploymentContainer)
	for rows.Next() {
		var container models_storage.DeploymentContainer
		err = rows.Scan(&container.DeploymentId, &container.Id, &container.Reference, &container.Alias, &container.Order)
		if err != nil {
			return nil, err
		}
		containers, ok := depContainers[container.DeploymentId]
		if !ok {
			containers = make(map[string]models_storage.DeploymentContainer)
			depContainers[container.DeploymentId] = containers
		}
		containers[container.Reference] = container
	}
	return depContainers, nil
}

func (h *Handler) DeploymentHostResources(ctx context.Context, deploymentId string) (map[string]models_storage.DeploymentHostResource, error) {
	deploymentsHostResources, err := h.DeploymentsHostResources(
		ctx,
		models_storage.DeploymentsHostResourcesFilter{DeploymentIds: []string{deploymentId}},
	)
	if err != nil {
		return nil, err
	}
	if len(deploymentsHostResources) == 0 {
		return nil, nil
	}
	return deploymentsHostResources[deploymentId], nil
}

func (h *Handler) DeploymentsHostResources(ctx context.Context, filter models_storage.DeploymentsHostResourcesFilter) (map[string]map[string]models_storage.DeploymentHostResource, error) {
	fc, val := genDeploymentsHostResourcesFilter(filter)
	rows, err := h.sqlDB.QueryContext(ctx,
		"SELECT dep_id, ref, res_id FROM dep_host_resources"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	depHostResources := make(map[string]map[string]models_storage.DeploymentHostResource)
	for rows.Next() {
		var hostResource models_storage.DeploymentHostResource
		err = rows.Scan(&hostResource.DeploymentId, &hostResource.Reference, &hostResource.Id)
		if err != nil {
			return nil, err
		}
		hostResources, ok := depHostResources[hostResource.DeploymentId]
		if !ok {
			hostResources = make(map[string]models_storage.DeploymentHostResource)
			depHostResources[hostResource.DeploymentId] = hostResources
		}
		hostResources[hostResource.Reference] = hostResource
	}
	return depHostResources, nil
}

func (h *Handler) DeploymentSecrets(ctx context.Context, deploymentId string) (map[string]models_storage.DeploymentSecret, error) {
	deploymentsSecrets, err := h.DeploymentsSecrets(ctx, models_storage.DeploymentsSecretsFilter{DeploymentIds: []string{deploymentId}})
	if err != nil {
		return nil, err
	}
	if len(deploymentsSecrets) == 0 {
		return nil, nil
	}
	return deploymentsSecrets[deploymentId], nil
}

func (h *Handler) DeploymentsSecrets(ctx context.Context, filter models_storage.DeploymentsSecretsFilter) (map[string]map[string]models_storage.DeploymentSecret, error) {
	fc, val := genDeploymentsSecretsFilter(filter)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT dep_id, ref, sec_id, item, as_mount, as_env FROM dep_secrets"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	depSecrets := make(map[string]map[string]models_storage.DeploymentSecret)
	for rows.Next() {
		var depId string
		var ref string
		var secId string
		var item models_storage.DeploymentSecretItem
		err = rows.Scan(&depId, &ref, &secId, &item.Name, &item.AsMount, &item.AsEnv)
		if err != nil {
			return nil, err
		}
		secrets, ok := depSecrets[depId]
		if !ok {
			secrets = make(map[string]models_storage.DeploymentSecret)
			depSecrets[depId] = secrets
		}
		secret, ok := secrets[ref]
		if !ok {
			secret.Id = secId
			secret.Reference = ref
			secret.DeploymentId = depId
		}
		secret.Items = append(secret.Items, item)
		secrets[secret.Reference] = secret
	}
	return depSecrets, nil
}

func (h *Handler) DeploymentConfigs(ctx context.Context, deploymentId string) (map[string]models_storage.DeploymentConfig, error) {
	deploymentsConfigs, err := h.DeploymentsConfigs(ctx, []string{deploymentId})
	if err != nil {
		return nil, err
	}
	if len(deploymentsConfigs) == 0 {
		return nil, nil
	}
	return deploymentsConfigs[deploymentId], nil
}

func (h *Handler) DeploymentsConfigs(ctx context.Context, deploymentIds []string) (map[string]map[string]models_storage.DeploymentConfig, error) {
	fc, val := genDeploymentsConfigsFilter(deploymentIds)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT dep_id, ref, v_string, v_int, v_float, v_bool FROM dep_configs"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	depConfigs := make(map[string]map[string]models_storage.DeploymentConfig)
	for rows.Next() {
		var depId string
		var ref string
		var vString sql.NullString
		var vInt sql.NullInt64
		var vFloat sql.NullFloat64
		var vBool sql.NullBool
		err = rows.Scan(&depId, &ref, &vString, &vInt, &vFloat, &vBool)
		if err != nil {
			return nil, err
		}
		config := models_storage.DeploymentConfig{
			DeploymentId: depId,
			Reference:    ref,
		}
		switch {
		case vString.Valid:
			config.String = vString.String
			config.DataType = models_storage.StringType
		case vInt.Valid:
			config.Int64 = vInt.Int64
			config.DataType = models_storage.Int64Type
		case vFloat.Valid:
			config.Float64 = vFloat.Float64
			config.DataType = models_storage.Float64Type
		case vBool.Valid:
			config.Bool = vBool.Bool
			config.DataType = models_storage.BoolType
		}
		configs, ok := depConfigs[depId]
		if !ok {
			configs = make(map[string]models_storage.DeploymentConfig)
			depConfigs[depId] = configs
		}
		configs[ref] = config
	}
	rows2, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT dep_id, ref, ord, v_string, v_int, v_float, v_bool FROM dep_list_configs"+fc+" ORDER BY dep_id, ref, ord;",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var depId string
		var ref string
		var ord int
		var vString sql.NullString
		var vInt sql.NullInt64
		var vFloat sql.NullFloat64
		var vBool sql.NullBool
		err = rows.Scan(&depId, &ref, &ord, &vString, &vInt, &vFloat, &vBool)
		if err != nil {
			return nil, err
		}
		configs, ok := depConfigs[depId]
		if !ok {
			configs = make(map[string]models_storage.DeploymentConfig)
			depConfigs[depId] = configs
		}
		config, ok := configs[ref]
		var dt int
		switch {
		case vString.Valid:
			config.StringSlice = append(config.StringSlice, vString.String)
			dt = models_storage.StringType
		case vInt.Valid:
			config.Int64Slice = append(config.Int64Slice, vInt.Int64)
			dt = models_storage.Int64Type
		case vFloat.Valid:
			config.Float64Slice = append(config.Float64Slice, vFloat.Float64)
			dt = models_storage.Float64Type
		case vBool.Valid:
			config.BoolSlice = append(config.BoolSlice, vBool.Bool)
			dt = models_storage.BoolType
		}
		if !ok {
			config.DeploymentId = depId
			config.Reference = ref
			config.DataType = dt
			config.IsSlice = true
		} else {
			if !config.IsSlice {
				return nil, errors.New("invalid config type")
			}
			if config.DataType != dt {
				return nil, errors.New("invalid data type")
			}
		}
		configs[ref] = config
	}
	return depConfigs, nil
}

func genDeploymentsConfigsFilter(ids []string) (string, []any) {
	var fc []string
	var val []any
	if len(ids) > 0 {
		ids = helper_slices.RemoveDuplicates(ids)
		fc = append(fc, "id IN ("+genQuestionMarks(len(ids))+")")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}

func genDeploymentsSecretsFilter(filter models_storage.DeploymentsSecretsFilter) (string, []any) {
	var fc []string
	var val []any
	if len(filter.Ids) > 0 {
		ids := helper_slices.RemoveDuplicates(filter.Ids)
		fc = append(fc, "res_id IN ("+genQuestionMarks(len(ids))+")")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if len(filter.DeploymentIds) > 0 {
		ids := helper_slices.RemoveDuplicates(filter.DeploymentIds)
		fc = append(fc, "dep_id IN ("+genQuestionMarks(len(ids))+")")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if filter.AsMount < 0 {
		fc = append(fc, "as_mount = ?")
		val = append(val, false)
	}
	if filter.AsMount > 0 {
		fc = append(fc, "as_mount = ?")
		val = append(val, true)
	}
	if filter.AsEnv < 0 {
		fc = append(fc, "as_env = ?")
		val = append(val, false)
	}
	if filter.AsEnv > 0 {
		fc = append(fc, "as_env = ?")
		val = append(val, true)
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}

func genDeploymentsHostResourcesFilter(filter models_storage.DeploymentsHostResourcesFilter) (string, []any) {
	var fc []string
	var val []any
	if len(filter.Ids) > 0 {
		ids := helper_slices.RemoveDuplicates(filter.Ids)
		fc = append(fc, "res_id IN ("+genQuestionMarks(len(ids))+")")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if len(filter.DeploymentIds) > 0 {
		ids := helper_slices.RemoveDuplicates(filter.DeploymentIds)
		fc = append(fc, "dep_id IN ("+genQuestionMarks(len(ids))+")")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}

func genDeploymentsContainersFilter(ids []string) (string, []any) {
	var fc []string
	var val []any
	if len(ids) > 0 {
		ids = helper_slices.RemoveDuplicates(ids)
		fc = append(fc, "id IN ("+genQuestionMarks(len(ids))+")")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}

func genDeploymentsFilter(filter models_storage.DeploymentsFilter) (string, []any) {
	var fc []string
	var val []any
	if len(filter.Ids) > 0 {
		ids := helper_slices.RemoveDuplicates(filter.Ids)
		fc = append(fc, "id IN ("+genQuestionMarks(len(ids))+")")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if len(filter.ModuleIds) > 0 {
		ids := helper_slices.RemoveDuplicates(filter.ModuleIds)
		fc = append(fc, "mod_id IN ("+genQuestionMarks(len(ids))+")")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if filter.Enabled < 0 {
		fc = append(fc, "enabled = ?")
		val = append(val, false)
	}
	if filter.Enabled > 0 {
		fc = append(fc, "enabled = ?")
		val = append(val, true)
	}
	if filter.Name != "" {
		fc = append(fc, "name LIKE '%?%'")
		val = append(val, filter.Name)
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}
