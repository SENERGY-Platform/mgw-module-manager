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

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

func (h *Handler) ReadAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models_handler_database.AuxiliaryDeploymentsFilter,
) (map[string]models_handler_database.AuxiliaryDeployment, error) {
	fc, val := genAuxiliaryDeploymentsFilter(deploymentId, filter)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT id, dep_id, image, ref, name, enabled, ctr_name, ctr_alias, command, pseudo_tty, created, updated FROM aux_deployments"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	auxDeps := make(map[string]models_handler_database.AuxiliaryDeployment)
	for rows.Next() {
		var auxDep models_handler_database.AuxiliaryDeployment
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

func (h *Handler) ReadAuxiliaryDeploymentsLabels(
	ctx context.Context,
	auxiliaryDeploymentsIds []string,
) (map[string]map[string]string, error) {
	fc, val := genAuxiliaryDeploymentsIdsFilter(auxiliaryDeploymentsIds)
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

func (h *Handler) ReadAuxiliaryDeploymentsConfigs(
	ctx context.Context,
	auxiliaryDeploymentsIds []string,
) (map[string]map[string]string, error) {
	fc, val := genAuxiliaryDeploymentsIdsFilter(auxiliaryDeploymentsIds)
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
) (map[string]models_handler_database.AuxiliaryDeploymentVolume, error) {
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT id, dep_id, ref, name FROM aux_dep_volumes WHERE dep_id = ?;",
		deploymentId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	auxDepVolumes := make(map[string]models_handler_database.AuxiliaryDeploymentVolume)
	for rows.Next() {
		var volume models_handler_database.AuxiliaryDeploymentVolume
		err = rows.Scan(&volume.Id, &volume.DeploymentId, &volume.Reference, &volume.Name)
		if err != nil {
			return nil, err
		}
		auxDepVolumes[volume.Reference] = volume
	}
	return auxDepVolumes, nil
}

const selectAuxiliaryDeploymentsVolumeMountsStmt = `SELECT aux_dep_volume_mounts.aux_dep_id, aux_dep_volume_mounts.vol_id, aux_dep_volumes.name, aux_dep_volumes.ref, aux_dep_volume_mounts.mnt_path
FROM aux_dep_volume_mounts
LEFT JOIN aux_dep_volumes
ON aux_dep_volume_mounts.vol_id = aux_dep_volumes.id ORDER BY aux_dep_id, ref`

func (h *Handler) ReadAuxiliaryDeploymentsVolumeMounts(
	ctx context.Context,
	auxiliaryDeploymentsIds []string,
) (map[string][]models_handler_database.AuxiliaryDeploymentVolumeMount, error) {
	var rows *sql.Rows
	var err error
	if len(auxiliaryDeploymentsIds) > 0 {
		auxiliaryDeploymentsIds = helper_slices.RemoveDuplicates(auxiliaryDeploymentsIds)
		rows, err = h.sqlDB.QueryContext(
			ctx,
			"SELECT * FROM ("+selectAuxiliaryDeploymentsVolumeMountsStmt+") AS SUB WHERE SUB.aux_dep_id IN ("+genQuestionMarks(len(auxiliaryDeploymentsIds))+");",
		)
	} else {
		rows, err = h.sqlDB.QueryContext(ctx, selectAuxiliaryDeploymentsVolumeMountsStmt+";")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	auxDepsVolumeMounts := make(map[string][]models_handler_database.AuxiliaryDeploymentVolumeMount)
	for rows.Next() {
		var mount models_handler_database.AuxiliaryDeploymentVolumeMount
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

func genAuxiliaryDeploymentsFilter(deploymentId string, filter models_handler_database.AuxiliaryDeploymentsFilter) (string, []any) {
	fc := []string{"dep_id = ?"}
	val := []any{deploymentId}
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
	if filter.Image != "" {
		fc = append(fc, "image = ?")
		val = append(val, filter.Image)
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}

func genAuxiliaryDeploymentsIdsFilter(ids []string) (string, []any) {
	ids = helper_slices.RemoveDuplicates(ids)
	if len(ids) > 0 {
		return " WHERE aux_dep_id IN (" + genQuestionMarks(len(ids)) + ")", helper_slices.ToAny(ids)
	}
	return "", nil
}
