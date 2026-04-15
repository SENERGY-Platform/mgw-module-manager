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
		auxDep.RunConfig.Command = command.String
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

const selectAuxiliaryDeploymentsVolumesStmt = `SELECT * FROM (SELECT aux_dep_volumes.id, aux_dep_volumes.dep_id, aux_dep_volumes.ref, aux_dep_volumes.name, aux_dep_volume_mounts.aux_dep_id, aux_dep_volume_mounts.mnt_path
FROM aux_dep_volumes
LEFT JOIN aux_dep_volume_mounts
ON aux_dep_volumes.id = aux_dep_volume_mounts.vol_id ORDER BY dep_id, ref, aux_dep_id) AS SUB WHERE SUB.dep_id = ?`

func (h *Handler) ReadAuxiliaryDeploymentsVolumes(
	ctx context.Context,
	deploymentId string,
) (map[string]models_handler_database.AuxiliaryDeploymentVolume, error) {
	rows, err := h.sqlDB.QueryContext(
		ctx,
		selectAuxiliaryDeploymentsVolumesStmt+";",
		deploymentId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	auxDepVolumes := make(map[string]models_handler_database.AuxiliaryDeploymentVolume)
	for rows.Next() {
		var id string
		var depId string
		var ref string
		var name string
		var auxDepId string
		var mntPath string
		err = rows.Scan(&id, &depId, &ref, &name, &auxDepId, &mntPath)
		if err != nil {
			return nil, err
		}
		volume, ok := auxDepVolumes[ref]
		if !ok {
			volume.Id = id
			volume.DeploymentId = depId
			volume.Reference = ref
			volume.Name = name
		}
		volume.Mounts = append(volume.Mounts, models_handler_database.AuxiliaryDeploymentVolumeMount{
			VolumeId:              id,
			Reference:             ref,
			AuxiliaryDeploymentId: auxDepId,
			MountPath:             mntPath,
		})
		auxDepVolumes[ref] = volume
	}
	return auxDepVolumes, nil
}

func (h *Handler) ReadAuxiliaryDeploymentsVolumeMounts(
	ctx context.Context,
	auxiliaryDeploymentsIds []string,
) (map[string][]models_handler_database.AuxiliaryDeploymentVolumeMount, error) {
	fc, val := genAuxiliaryDeploymentsIdsFilter(auxiliaryDeploymentsIds)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT vol_id, ref, aux_dep_id, mnt_path FROM aux_dep_volume_mounts"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	auxDepsVolumeMounts := make(map[string][]models_handler_database.AuxiliaryDeploymentVolumeMount)
	for rows.Next() {
		var mount models_handler_database.AuxiliaryDeploymentVolumeMount
		err = rows.Scan(&mount.VolumeId, &mount.Reference, &mount.AuxiliaryDeploymentId, &mount.MountPath)
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
