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
	"strings"
	"time"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

func (h *Handler) ReadDeployment(ctx context.Context, id string) (pkg_models.DeploymentBase, error) {
	deployments, err := h.ReadDeployments(ctx, pkg_models.DeploymentsFilter{Ids: []string{id}})
	if err != nil {
		return pkg_models.DeploymentBase{}, err
	}
	if len(deployments) == 0 {
		return pkg_models.DeploymentBase{}, pkg_models.NotFoundErr
	}
	return deployments[id], nil
}

func (h *Handler) ReadDeployments(ctx context.Context, filter pkg_models.DeploymentsFilter) (map[string]pkg_models.DeploymentBase, error) {
	fc, val := genDeploymentsFilter(filter)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT id, mod_id, mod_source, mod_channel, mod_ver, dir, files_dir, enabled, created, updated FROM deployments"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deps := make(map[string]pkg_models.DeploymentBase)
	for rows.Next() {
		var dep pkg_models.DeploymentBase
		var ct, ut []uint8
		err = rows.Scan(
			&dep.Id,
			&dep.ModuleId,
			&dep.ModuleSource,
			&dep.ModuleChannel,
			&dep.ModuleVersion,
			&dep.DirName,
			&dep.FilesDirName,
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

func (h *Handler) ReadDeploymentContainers(ctx context.Context, deploymentId string) (map[string]pkg_models.DeploymentContainerBase, error) {
	deploymentsContainers, err := h.ReadDeploymentsContainers(ctx, []string{deploymentId})
	if err != nil {
		return nil, err
	}
	if len(deploymentsContainers) == 0 {
		return nil, nil
	}
	return deploymentsContainers[deploymentId], nil
}

func (h *Handler) ReadDeploymentsContainers(ctx context.Context, deploymentIds []string) (map[string]map[string]pkg_models.DeploymentContainerBase, error) {
	fc, val := genDeploymentsContainersFilter(deploymentIds)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT dep_id, name, srv_ref, alias FROM dep_containers"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	depContainers := make(map[string]map[string]pkg_models.DeploymentContainerBase)
	for rows.Next() {
		var container pkg_models.DeploymentContainerBase
		err = rows.Scan(&container.DeploymentId, &container.Name, &container.Reference, &container.Alias)
		if err != nil {
			return nil, err
		}
		containers, ok := depContainers[container.DeploymentId]
		if !ok {
			containers = make(map[string]pkg_models.DeploymentContainerBase)
			depContainers[container.DeploymentId] = containers
		}
		containers[container.Reference] = container
	}
	return depContainers, nil
}

func (h *Handler) ReadDeploymentsVolumes(ctx context.Context, deploymentIds []string) (map[string]map[string]pkg_models.DeploymentVolume, error) {
	fc, val := genDeploymentsVolumesFilter(deploymentIds)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT dep_id, ref, name FROM dep_volumes"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	depVolumes := make(map[string]map[string]pkg_models.DeploymentVolume)
	for rows.Next() {
		var volume pkg_models.DeploymentVolume
		err = rows.Scan(&volume.DeploymentId, &volume.Reference, &volume.Name)
		if err != nil {
			return nil, err
		}
		volumes, ok := depVolumes[volume.DeploymentId]
		if !ok {
			volumes = make(map[string]pkg_models.DeploymentVolume)
			depVolumes[volume.DeploymentId] = volumes
		}
		volumes[volume.Reference] = volume
	}
	return depVolumes, nil
}

func (h *Handler) ReadDeploymentHostResources(ctx context.Context, deploymentId string) (map[string]pkg_models.DeploymentHostResource, error) {
	deploymentsHostResources, err := h.ReadDeploymentsHostResources(
		ctx,
		pkg_models.DeploymentsHostResourcesFilter{DeploymentIds: []string{deploymentId}},
	)
	if err != nil {
		return nil, err
	}
	if len(deploymentsHostResources) == 0 {
		return nil, nil
	}
	return deploymentsHostResources[deploymentId], nil
}

func (h *Handler) ReadDeploymentsHostResources(
	ctx context.Context,
	filter pkg_models.DeploymentsHostResourcesFilter,
) (map[string]map[string]pkg_models.DeploymentHostResource, error) {
	fc, val := genDeploymentsHostResourcesFilter(filter)
	rows, err := h.sqlDB.QueryContext(ctx,
		"SELECT dep_id, ref, res_id FROM dep_host_resources"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	depHostResources := make(map[string]map[string]pkg_models.DeploymentHostResource)
	for rows.Next() {
		var hostResource pkg_models.DeploymentHostResource
		err = rows.Scan(&hostResource.DeploymentId, &hostResource.Reference, &hostResource.Id)
		if err != nil {
			return nil, err
		}
		hostResources, ok := depHostResources[hostResource.DeploymentId]
		if !ok {
			hostResources = make(map[string]pkg_models.DeploymentHostResource)
			depHostResources[hostResource.DeploymentId] = hostResources
		}
		hostResources[hostResource.Reference] = hostResource
	}
	return depHostResources, nil
}

func (h *Handler) ReadDeploymentSecrets(ctx context.Context, deploymentId string) (map[string]pkg_models.DeploymentSecret, error) {
	deploymentsSecrets, err := h.ReadDeploymentsSecrets(ctx, pkg_models.DeploymentsSecretsFilter{DeploymentIds: []string{deploymentId}})
	if err != nil {
		return nil, err
	}
	if len(deploymentsSecrets) == 0 {
		return nil, nil
	}
	return deploymentsSecrets[deploymentId], nil
}

func (h *Handler) ReadDeploymentsSecrets(ctx context.Context, filter pkg_models.DeploymentsSecretsFilter) (map[string]map[string]pkg_models.DeploymentSecret, error) {
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
	depSecrets := make(map[string]map[string]pkg_models.DeploymentSecret)
	for rows.Next() {
		var depId string
		var ref string
		var secId string
		var item lib_models.DeploymentSecretItem
		err = rows.Scan(&depId, &ref, &secId, &item.Name, &item.AsMount, &item.AsEnv)
		if err != nil {
			return nil, err
		}
		secrets, ok := depSecrets[depId]
		if !ok {
			secrets = make(map[string]pkg_models.DeploymentSecret)
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

func (h *Handler) ReadDeploymentUserConfigs(ctx context.Context, deploymentId string) (map[string]pkg_models.DeploymentUserConfig, error) {
	deploymentsConfigs, err := h.ReadDeploymentsConfigs(ctx, []string{deploymentId})
	if err != nil {
		return nil, err
	}
	if len(deploymentsConfigs) == 0 {
		return nil, nil
	}
	return deploymentsConfigs[deploymentId], nil
}

func (h *Handler) ReadDeploymentsConfigs(ctx context.Context, deploymentIds []string) (map[string]map[string]pkg_models.DeploymentUserConfig, error) {
	rows, err := h.queryConfigs(ctx, deploymentIds, "dep_configs", "dep_config_values", "dep_id", "dep_id", "ref")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	depConfigs := make(map[string]map[string]pkg_models.DeploymentUserConfig) // {depID:{reference:config}}
	for rows.Next() {
		var id string
		var isList bool
		var dataType int
		var vString sql.NullString
		var vInt sql.NullInt64
		var vFloat sql.NullFloat64
		var vBool sql.NullBool
		var ord int
		var depId string
		var ref string
		err = rows.Scan(&id, &dataType, &isList, &vString, &vInt, &vFloat, &vBool, &ord, &depId, &ref)
		if err != nil {
			return nil, err
		}
		configs, ok := depConfigs[depId]
		if !ok {
			configs = make(map[string]pkg_models.DeploymentUserConfig)
			depConfigs[depId] = configs
		}
		config, ok := configs[ref]
		if !ok {
			config.Id = id
			config.IsSlice = isList
			config.DataType = dataType
			config.DeploymentId = depId
			config.Reference = ref
		}
		if isList {
			switch dataType {
			case pkg_models.DataTypeString:
				config.StringSlice = append(config.StringSlice, vString.String)
			case pkg_models.DataTypeInt64:
				config.Int64Slice = append(config.Int64Slice, vInt.Int64)
			case pkg_models.DataTypeFloat64:
				config.Float64Slice = append(config.Float64Slice, vFloat.Float64)
			case pkg_models.DataTypeBool:
				config.BoolSlice = append(config.BoolSlice, vBool.Bool)
			}
		} else {
			switch dataType {
			case pkg_models.DataTypeString:
				config.String = vString.String
			case pkg_models.DataTypeInt64:
				config.Int64 = vInt.Int64
			case pkg_models.DataTypeFloat64:
				config.Float64 = vFloat.Float64
			case pkg_models.DataTypeBool:
				config.Bool = vBool.Bool
			}
		}
		configs[ref] = config
	}
	return depConfigs, nil
}

func (h *Handler) ReadDeploymentGlobalConfigs(ctx context.Context, deploymentId string) (map[string]pkg_models.DeploymentGlobalConfig, error) {
	deploymentsGlobalConfigs, err := h.ReadDeploymentsGlobalConfigs(
		ctx,
		pkg_models.DeploymentGlobalConfigsFilter{DeploymentIds: []string{deploymentId}},
	)
	if err != nil {
		return nil, err
	}
	if len(deploymentsGlobalConfigs) == 0 {
		return nil, nil
	}
	return deploymentsGlobalConfigs[deploymentId], nil
}

func (h *Handler) ReadDeploymentsGlobalConfigs(ctx context.Context, filter pkg_models.DeploymentGlobalConfigsFilter) (map[string]map[string]pkg_models.DeploymentGlobalConfig, error) {
	fc, val := genDeploymentGlobalConfigsFilter(filter)
	rows, err := h.sqlDB.QueryContext(ctx,
		"SELECT dep_id, ref, c_id FROM dep_global_configs"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	depGlobalConfigs := make(map[string]map[string]pkg_models.DeploymentGlobalConfig)
	for rows.Next() {
		var globalConfig pkg_models.DeploymentGlobalConfig
		err = rows.Scan(&globalConfig.DeploymentId, &globalConfig.Reference, &globalConfig.Id)
		if err != nil {
			return nil, err
		}
		globalConfigs, ok := depGlobalConfigs[globalConfig.DeploymentId]
		if !ok {
			globalConfigs = make(map[string]pkg_models.DeploymentGlobalConfig)
			depGlobalConfigs[globalConfig.DeploymentId] = globalConfigs
		}
		globalConfigs[globalConfig.Reference] = globalConfig
	}
	return depGlobalConfigs, nil
}

func (h *Handler) ReadDeploymentFiles(ctx context.Context, deploymentId string) (map[string]pkg_models.DeploymentFile, error) {
	depFiles, err := h.ReadDeploymentsFiles(ctx, []string{deploymentId})
	if err != nil {
		return nil, err
	}
	if len(depFiles) == 0 {
		return nil, nil
	}
	return depFiles[deploymentId], nil
}

func (h *Handler) ReadDeploymentsFiles(ctx context.Context, deploymentIds []string) (map[string]map[string]pkg_models.DeploymentFile, error) {
	fc, val := genDeploymentsFilesFilter(deploymentIds)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT dep_id, ref, data FROM dep_files"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	depFiles := make(map[string]map[string]pkg_models.DeploymentFile)
	for rows.Next() {
		var depFile pkg_models.DeploymentFile
		err = rows.Scan(&depFile.DeploymentId, &depFile.Reference, &depFile.Data)
		if err != nil {
			return nil, err
		}
		files, ok := depFiles[depFile.DeploymentId]
		if !ok {
			files = make(map[string]pkg_models.DeploymentFile)
			depFiles[depFile.DeploymentId] = files
		}
		files[depFile.Reference] = depFile
	}
	return depFiles, nil
}

func (h *Handler) ReadDeploymentFileGroups(ctx context.Context, deploymentId string) (map[string]pkg_models.DeploymentFileGroup, error) {
	depFileGroups, err := h.ReadDeploymentsFileGroups(ctx, []string{deploymentId})
	if err != nil {
		return nil, err
	}
	if len(depFileGroups) == 0 {
		return nil, nil
	}
	return depFileGroups[deploymentId], nil
}

const selectFileGroupsStmt = `SELECT dep_file_groups.id, dep_file_groups.dep_id, dep_file_groups.ref, dep_file_group_files.path, dep_file_group_files.format, dep_file_group_files.data
FROM dep_file_groups
LEFT JOIN dep_file_group_files
ON dep_file_groups.id = dep_file_group_files.g_id`

func (h *Handler) ReadDeploymentsFileGroups(ctx context.Context, deploymentIds []string) (map[string]map[string]pkg_models.DeploymentFileGroup, error) {
	var rows *sql.Rows
	var err error
	if len(deploymentIds) > 0 {
		deploymentIds = helper_slices.RemoveDuplicates(deploymentIds)
		rows, err = h.sqlDB.QueryContext(
			ctx,
			selectFileGroupsStmt+" WHERE dep_id IN ("+genQuestionMarks(len(deploymentIds))+") ORDER BY dep_id, path;",
			helper_slices.ToAny(deploymentIds)...,
		)
	} else {
		rows, err = h.sqlDB.QueryContext(ctx, selectFileGroupsStmt+" ORDER BY dep_id, path;")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	depFileGroups := make(map[string]map[string]pkg_models.DeploymentFileGroup) // {depID:{reference:DeploymentFileGroup}}
	for rows.Next() {
		var id string
		var depId string
		var ref string
		var path string
		var format int
		var data []byte
		err = rows.Scan(&id, &depId, &ref, &path, &format, &data)
		if err != nil {
			return nil, err
		}
		fileGroups, ok := depFileGroups[depId]
		if !ok {
			fileGroups = make(map[string]pkg_models.DeploymentFileGroup)
			depFileGroups[depId] = fileGroups
		}
		fileGroup, ok := fileGroups[ref]
		if !ok {
			fileGroup.Id = id
			fileGroup.DeploymentId = depId
			fileGroup.Reference = ref
		}
		fileGroup.Files = append(fileGroup.Files, pkg_models.DeploymentFileGroupFile{
			Path:   path,
			Format: format,
			Data:   data,
		})
		fileGroups[ref] = fileGroup
	}
	return depFileGroups, nil
}

func genDeploymentGlobalConfigsFilter(filter pkg_models.DeploymentGlobalConfigsFilter) (string, []any) {
	var fc []string
	var val []any
	if len(filter.Ids) > 0 {
		ids := helper_slices.RemoveDuplicates(filter.Ids)
		fc = append(fc, "c_id IN ("+genQuestionMarks(len(ids))+")")
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

func genDeploymentsSecretsFilter(filter pkg_models.DeploymentsSecretsFilter) (string, []any) {
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

func genDeploymentsHostResourcesFilter(filter pkg_models.DeploymentsHostResourcesFilter) (string, []any) {
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

func genDeploymentsVolumesFilter(ids []string) (string, []any) {
	var fc []string
	var val []any
	if len(ids) > 0 {
		ids = helper_slices.RemoveDuplicates(ids)
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

func genDeploymentsFilesFilter(ids []string) (string, []any) {
	var fc []string
	var val []any
	if len(ids) > 0 {
		ids = helper_slices.RemoveDuplicates(ids)
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

func genDeploymentsFilter(filter pkg_models.DeploymentsFilter) (string, []any) {
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
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}
