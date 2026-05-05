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
	"database/sql"
	"fmt"
	"strings"
	"time"

	lib_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/lib/models/aux_deployments"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/aux_deployments"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
)

func (h *Handler) ReadAuxiliaryDeployment(
	ctx context.Context,
	deploymentId string,
	auxDeploymentId string,
) (aux_deployments.AuxiliaryDeployment, error) {
	auxDeployments, err := h.ReadAuxiliaryDeployments(ctx, deploymentId, lib_aux_deployments.AuxiliaryDeploymentsFilter{
		Ids: []string{auxDeploymentId},
	})
	if err != nil {
		return aux_deployments.AuxiliaryDeployment{}, err
	}
	if len(auxDeploymentId) == 0 {
		return aux_deployments.AuxiliaryDeployment{}, models_error.NotFoundErr
	}
	return auxDeployments[auxDeploymentId], nil
}

func (h *Handler) ReadAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter lib_aux_deployments.AuxiliaryDeploymentsFilter,
) (map[string]aux_deployments.AuxiliaryDeployment, error) {
	fc, val := genAuxiliaryDeploymentsFilter(deploymentId, filter)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT id, dep_id, image, ref, name, enabled, ctr_name, ctr_alias, recreate, command, pseudo_tty, created, updated FROM aux_deployments"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	auxDeps := make(map[string]aux_deployments.AuxiliaryDeployment)
	for rows.Next() {
		var auxDep aux_deployments.AuxiliaryDeployment
		var ct, ut []uint8
		var command sql.NullString
		var pseudoTTY sql.NullBool
		err = rows.Scan(
			&auxDep.Id,
			&auxDep.DeploymentId,
			&auxDep.Image,
			&auxDep.Reference,
			&auxDep.Name,
			&auxDep.Enabled,
			&auxDep.Container.Name,
			&auxDep.Container.Alias,
			&auxDep.Recreate,
			&command,
			&pseudoTTY,
			&ct,
			&ut,
		)
		if err != nil {
			return nil, err
		}
		if auxDep.Created, err = time.Parse(timeLayout, string(ct)); err != nil {
			return nil, err
		}
		if auxDep.Updated, err = time.Parse(timeLayout, string(ut)); err != nil {
			return nil, err
		}
		auxDep.RunConfig.Command = strings.Split(command.String, ",")
		auxDep.RunConfig.PseudoTTY = pseudoTTY.Bool
		auxDeps[auxDep.Id] = auxDep
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return auxDeps, nil
}

func (h *Handler) ReadAuxiliaryDeploymentLabels(ctx context.Context, auxiliaryDeploymentId string) (map[string]string, error) {
	auxDepsLabels, err := h.ReadAuxiliaryDeploymentsLabels(ctx, []string{auxiliaryDeploymentId})
	if err != nil {
		return nil, err
	}
	return auxDepsLabels[auxiliaryDeploymentId], nil
}

func (h *Handler) ReadAuxiliaryDeploymentsLabels(
	ctx context.Context,
	auxiliaryDeploymentsIds []string,
) (map[string]map[string]string, error) {
	fc, val := genAuxiliaryDeploymentsAssetsIdsFilter(auxiliaryDeploymentsIds)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT aux_dep_id, name, value FROM aux_dep_labels"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	auxDepsLabels := make(map[string]map[string]string)
	for rows.Next() {
		var id, name string
		var value sql.NullString
		err = rows.Scan(&id, &name, &value)
		if err != nil {
			return nil, err
		}
		labels, ok := auxDepsLabels[id]
		if !ok {
			labels = make(map[string]string)
			auxDepsLabels[id] = labels
		}
		labels[name] = value.String
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return auxDepsLabels, nil
}

func (h *Handler) ReadAuxiliaryDeploymentConfigs(ctx context.Context, auxiliaryDeploymentId string) (map[string]string, error) {
	auxDepsConfigs, err := h.ReadAuxiliaryDeploymentsConfigs(ctx, []string{auxiliaryDeploymentId})
	if err != nil {
		return nil, err
	}
	return auxDepsConfigs[auxiliaryDeploymentId], nil
}

func (h *Handler) ReadAuxiliaryDeploymentsConfigs(
	ctx context.Context,
	auxiliaryDeploymentsIds []string,
) (map[string]map[string]string, error) {
	fc, val := genAuxiliaryDeploymentsAssetsIdsFilter(auxiliaryDeploymentsIds)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT aux_dep_id, name, value FROM aux_dep_configs"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	auxDepsConfigs := make(map[string]map[string]string)
	for rows.Next() {
		var id, name string
		var value sql.NullString
		err = rows.Scan(&id, &name, &value)
		if err != nil {
			return nil, err
		}
		configs, ok := auxDepsConfigs[id]
		if !ok {
			configs = make(map[string]string)
			auxDepsConfigs[id] = configs
		}
		configs[name] = value.String
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return auxDepsConfigs, nil
}

func (h *Handler) ReadAuxiliaryDeploymentVolumes(
	ctx context.Context,
	deploymentId string,
	refFilter []string,
) (map[string]aux_deployments.AuxiliaryDeploymentVolume, error) {
	fc, val := genAuxiliaryDeploymentVolumesFilter(deploymentId, refFilter)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT id, dep_id, ref, name FROM aux_dep_volumes"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	auxDepVolumes := make(map[string]aux_deployments.AuxiliaryDeploymentVolume)
	for rows.Next() {
		var volume aux_deployments.AuxiliaryDeploymentVolume
		err = rows.Scan(&volume.Id, &volume.DeploymentId, &volume.Reference, &volume.Name)
		if err != nil {
			return nil, err
		}
		auxDepVolumes[volume.Reference] = volume
	}
	return auxDepVolumes, nil
}

const selectAuxiliaryDeploymentVolumesWithMountsStmt = `SELECT aux_dep_volumes.id, aux_dep_volumes.dep_id, aux_dep_volumes.ref, aux_dep_volumes.name, aux_dep_volume_mounts.aux_dep_id
FROM aux_dep_volumes
LEFT JOIN aux_dep_volume_mounts
ON aux_dep_volumes.id = aux_dep_volume_mounts.vol_id`

func (h *Handler) ReadAuxiliaryDeploymentVolumesWithMounts(
	ctx context.Context,
	deploymentId string,
	refFilter []string,
) (map[string]aux_deployments.AuxiliaryDeploymentVolumeWithMounts, error) {
	fc, val := genAuxiliaryDeploymentVolumesFilter(deploymentId, refFilter)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		selectAuxiliaryDeploymentVolumesWithMountsStmt+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	auxDepVolumes := make(map[string]aux_deployments.AuxiliaryDeploymentVolumeWithMounts)
	for rows.Next() {
		var id string
		var depId string
		var reference string
		var volName string
		var auxDepId string
		err = rows.Scan(&id, &depId, &reference, &volName, &auxDepId)
		if err != nil {
			return nil, err
		}
		volume, ok := auxDepVolumes[reference]
		if !ok {
			volume.Id = id
			volume.DeploymentId = depId
			volume.Reference = reference
			volume.Name = volName
		}
		volume.MountedBy = append(volume.MountedBy, auxDepId)
		auxDepVolumes[reference] = volume
	}
	return auxDepVolumes, nil
}

func (h *Handler) ReadAuxiliaryDeploymentVolumeMounts(
	ctx context.Context,
	auxiliaryDeploymentId string,
) ([]aux_deployments.AuxiliaryDeploymentVolumeMount, error) {
	auxDepsVolumeMounts, err := h.ReadAuxiliaryDeploymentsVolumeMounts(ctx, []string{auxiliaryDeploymentId})
	if err != nil {
		return nil, err
	}
	return auxDepsVolumeMounts[auxiliaryDeploymentId], nil
}

const selectAuxiliaryDeploymentsVolumeMountsStmt = `SELECT aux_dep_volume_mounts.aux_dep_id, aux_dep_volume_mounts.vol_id, aux_dep_volumes.name, aux_dep_volumes.ref, aux_dep_volume_mounts.mnt_path
FROM aux_dep_volume_mounts
LEFT JOIN aux_dep_volumes
ON aux_dep_volume_mounts.vol_id = aux_dep_volumes.id`

func (h *Handler) ReadAuxiliaryDeploymentsVolumeMounts(
	ctx context.Context,
	auxiliaryDeploymentsIds []string,
) (map[string][]aux_deployments.AuxiliaryDeploymentVolumeMount, error) {
	var rows *sql.Rows
	var err error
	if len(auxiliaryDeploymentsIds) > 0 {
		auxiliaryDeploymentsIds = helper_slices.RemoveDuplicates(auxiliaryDeploymentsIds)
		rows, err = h.sqlDB.QueryContext(
			ctx,
			selectAuxiliaryDeploymentsVolumeMountsStmt+" WHERE aux_dep_id IN ("+genQuestionMarks(len(auxiliaryDeploymentsIds))+") ORDER BY aux_dep_id, ref;",
		)
	} else {
		rows, err = h.sqlDB.QueryContext(ctx, selectAuxiliaryDeploymentsVolumeMountsStmt+" ORDER BY aux_dep_id, ref;")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	auxDepsVolumeMounts := make(map[string][]aux_deployments.AuxiliaryDeploymentVolumeMount)
	for rows.Next() {
		var mount aux_deployments.AuxiliaryDeploymentVolumeMount
		err = rows.Scan(&mount.AuxiliaryDeploymentId, &mount.VolumeId, &mount.VolumeName, &mount.Reference, &mount.MountPath)
		if err != nil {
			return nil, err
		}
		mounts := auxDepsVolumeMounts[mount.AuxiliaryDeploymentId]
		mounts = append(mounts, mount)
		auxDepsVolumeMounts[mount.AuxiliaryDeploymentId] = mounts
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return auxDepsVolumeMounts, nil
}

const selectAuxDeploymentsByParentStmt = `SELECT deployments.id AS dep_id, deployments.enabled AS dep_enabled, aux_deployments.id, aux_deployments.enabled, aux_deployments.ctr_name, aux_deployments.ctr_alias
FROM aux_deployments 
LEFT JOIN deployments ON aux_deployments.dep_id = deployments.id`

func (h *Handler) ReadAuxDeploymentsByParent(ctx context.Context) (
	map[string]aux_deployments.AuxiliaryDeploymentParent,
	error,
) {
	rows, err := h.sqlDB.QueryContext(
		ctx,
		selectAuxDeploymentsByParentStmt,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	auxDepsByParent := make(map[string]aux_deployments.AuxiliaryDeploymentParent)
	for rows.Next() {
		var parentId string
		var parentEnabled bool
		var auxDep aux_deployments.AuxiliaryDeployment
		err = rows.Scan(
			&parentId,
			&parentEnabled,
			&auxDep.Id,
			&auxDep.Enabled,
			&auxDep.Container.Name,
			&auxDep.Container.Alias,
		)
		if err != nil {
			return nil, err
		}
		auxDepParent, ok := auxDepsByParent[parentId]
		if !ok {
			auxDepParent.Id = parentId
			auxDepParent.Enabled = parentEnabled
			auxDepParent.AuxiliaryDeployments = make(map[string]aux_deployments.AuxiliaryDeployment)
			auxDepsByParent[parentId] = auxDepParent
		}
		auxDepParent.AuxiliaryDeployments[auxDep.Id] = auxDep
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return auxDepsByParent, nil
}

func genAuxiliaryDeploymentsFilter(deploymentId string, filter lib_aux_deployments.AuxiliaryDeploymentsFilter) (string, []any) {
	var fc []string
	var val []any
	if deploymentId != "" {
		fc = append(fc, "dep_id = ?")
		val = append(val, deploymentId)
	}
	if len(filter.Labels) > 0 {
		var tc int
		var str string
		for n, v := range filter.Labels {
			val = append(val, n, v)
			if tc == 0 {
				str = fmt.Sprintf("SELECT t%d.* FROM (SELECT aux_dep_id FROM aux_dep_labels WHERE name = ? AND value = ?) t%d", tc, tc)
			} else {
				str += fmt.Sprintf(" INNER JOIN (SELECT aux_dep_id FROM aux_dep_labels WHERE name = ? AND value = ?) t%d ON t%d.aux_dep_id = t%d.aux_dep_id", tc, tc-1, tc)
			}
			tc += 1
		}
		fc = append(fc, "id IN ("+str+")")
	}
	if len(filter.Ids) > 0 {
		ids := helper_slices.RemoveDuplicates(filter.Ids)
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
	if filter.Recreate < 0 {
		fc = append(fc, "recreate = ?")
		val = append(val, false)
	}
	if filter.Recreate > 0 {
		fc = append(fc, "recreate = ?")
		val = append(val, true)
	}
	if filter.Image != "" {
		fc = append(fc, "image = ?")
		val = append(val, filter.Image)
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}

func genAuxiliaryDeploymentsAssetsIdsFilter(ids []string) (string, []any) {
	if len(ids) > 0 {
		ids = helper_slices.RemoveDuplicates(ids)
		return " WHERE aux_dep_id IN (" + genQuestionMarks(len(ids)) + ")", helper_slices.ToAny(ids)
	}
	return "", nil
}

func genAuxiliaryDeploymentVolumesFilter(deploymentId string, references []string) (string, []any) {
	var fc []string
	var val []any
	if deploymentId != "" {
		fc = append(fc, "dep_id = ?")
		val = append(val, deploymentId)
	}
	if len(references) > 0 {
		references = helper_slices.RemoveDuplicates(references)
		fc = append(fc, "ref IN ("+genQuestionMarks(len(references))+")")
		for _, ref := range references {
			val = append(val, ref)
		}
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}
