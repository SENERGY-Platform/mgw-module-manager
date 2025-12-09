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

	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/storage"
)

func (h *Handler) Deployment(ctx context.Context, id string) (models_storage.Deployment, error) {
	deployments, err := h.Deployments(ctx, models_storage.DeploymentsFilter{IDs: []string{id}})
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
			&dep.ID,
			&dep.Module.ID,
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
		deps[dep.ID] = dep
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return deps, nil
}

func (h *Handler) DeploymentContainers(ctx context.Context, deploymentID string) (map[string]models_storage.DeploymentContainer, error) {
	deploymentsContainers, err := h.DeploymentsContainers(ctx, []string{deploymentID})
	if err != nil {
		return nil, err
	}
	if len(deploymentsContainers) == 0 {
		return nil, nil
	}
	return deploymentsContainers[deploymentID], nil
}

func (h *Handler) DeploymentsContainers(ctx context.Context, deploymentIDs []string) (map[string]map[string]models_storage.DeploymentContainer, error) {
	fc, val := genDeploymentsContainersFilter(deploymentIDs)
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
		err = rows.Scan(&container.DeploymentID, &container.ID, &container.Reference, &container.Alias, &container.Order)
		if err != nil {
			return nil, err
		}
		containers, ok := depContainers[container.DeploymentID]
		if !ok {
			containers = make(map[string]models_storage.DeploymentContainer)
			depContainers[container.DeploymentID] = containers
		}
		containers[container.Reference] = container
	}
	return depContainers, nil
}

func (h *Handler) DeploymentHostResources(ctx context.Context, deploymentID string) (map[string]models_storage.DeploymentHostResource, error) {
	deploymentsHostResources, err := h.DeploymentsHostResources(
		ctx,
		models_storage.DeploymentsHostResourcesFilter{DeploymentIDs: []string{deploymentID}},
	)
	if err != nil {
		return nil, err
	}
	if len(deploymentsHostResources) == 0 {
		return nil, nil
	}
	return deploymentsHostResources[deploymentID], nil
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
		err = rows.Scan(&hostResource.DeploymentID, &hostResource.Reference, &hostResource.ID)
		if err != nil {
			return nil, err
		}
		hostResources, ok := depHostResources[hostResource.DeploymentID]
		if !ok {
			hostResources = make(map[string]models_storage.DeploymentHostResource)
			depHostResources[hostResource.DeploymentID] = hostResources
		}
		hostResources[hostResource.Reference] = hostResource
	}
	return depHostResources, nil
}

func (h *Handler) DeploymentSecrets(ctx context.Context, deploymentID string) (map[string]models_storage.DeploymentSecret, error) {
	deploymentsSecrets, err := h.DeploymentsSecrets(ctx, models_storage.DeploymentsSecretsFilter{DeploymentIDs: []string{deploymentID}})
	if err != nil {
		return nil, err
	}
	if len(deploymentsSecrets) == 0 {
		return nil, nil
	}
	return deploymentsSecrets[deploymentID], nil
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
		var depID string
		var ref string
		var secID string
		var item models_storage.DeploymentSecretItem
		err = rows.Scan(&depID, &ref, &secID, &item.Name, &item.AsMount, &item.AsEnv)
		if err != nil {
			return nil, err
		}
		secrets, ok := depSecrets[depID]
		if !ok {
			secrets = make(map[string]models_storage.DeploymentSecret)
			depSecrets[depID] = secrets
		}
		secret, ok := secrets[ref]
		if !ok {
			secret.ID = secID
			secret.Reference = ref
			secret.DeploymentID = depID
		}
		secret.Items = append(secret.Items, item)
		secrets[secret.Reference] = secret
	}
	return depSecrets, nil
}

func (h *Handler) DeploymentConfigs(ctx context.Context, deploymentID string) (map[string]models_storage.DeploymentConfig, error) {
	deploymentsConfigs, err := h.DeploymentsConfigs(ctx, []string{deploymentID})
	if err != nil {
		return nil, err
	}
	if len(deploymentsConfigs) == 0 {
		return nil, nil
	}
	return deploymentsConfigs[deploymentID], nil
}

func (h *Handler) DeploymentsConfigs(ctx context.Context, deploymentIDs []string) (map[string]map[string]models_storage.DeploymentConfig, error) {
	fc, val := genDeploymentsConfigsFilter(deploymentIDs)
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
		var depID string
		var ref string
		var vString sql.NullString
		var vInt sql.NullInt64
		var vFloat sql.NullFloat64
		var vBool sql.NullBool
		err = rows.Scan(&depID, &ref, &vString, &vInt, &vFloat, &vBool)
		if err != nil {
			return nil, err
		}
		var config models_storage.DeploymentConfig
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
		configs, ok := depConfigs[depID]
		if !ok {
			configs = make(map[string]models_storage.DeploymentConfig)
			depConfigs[depID] = configs
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
		var depID string
		var ref string
		var ord int
		var vString sql.NullString
		var vInt sql.NullInt64
		var vFloat sql.NullFloat64
		var vBool sql.NullBool
		err = rows.Scan(&depID, &ref, &ord, &vString, &vInt, &vFloat, &vBool)
		if err != nil {
			return nil, err
		}
		configs, ok := depConfigs[depID]
		if !ok {
			configs = make(map[string]models_storage.DeploymentConfig)
			depConfigs[depID] = configs
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

func genDeploymentsConfigsFilter(IDs []string) (string, []any) {
	var fc []string
	var val []any
	if len(IDs) > 0 {
		ids := removeDuplicates(IDs)
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
	if len(filter.IDs) > 0 {
		ids := removeDuplicates(filter.IDs)
		fc = append(fc, "res_id IN ("+genQuestionMarks(len(ids))+")")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if len(filter.DeploymentIDs) > 0 {
		ids := removeDuplicates(filter.DeploymentIDs)
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
	if len(filter.IDs) > 0 {
		ids := removeDuplicates(filter.IDs)
		fc = append(fc, "res_id IN ("+genQuestionMarks(len(ids))+")")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if len(filter.DeploymentIDs) > 0 {
		ids := removeDuplicates(filter.DeploymentIDs)
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

func genDeploymentsContainersFilter(IDs []string) (string, []any) {
	var fc []string
	var val []any
	if len(IDs) > 0 {
		ids := removeDuplicates(IDs)
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
	if len(filter.IDs) > 0 {
		ids := removeDuplicates(filter.IDs)
		fc = append(fc, "id IN ("+genQuestionMarks(len(ids))+")")
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
