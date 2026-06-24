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

package database

import (
	"context"
	"database/sql"
	"strings"
	"time"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
)

const selectDeploymentAdvertisementsStmt = `SELECT dep_advertisements.id, dep_advertisements.dep_id, dep_advertisements.mod_id, dep_advertisements.ref, dep_advertisements.timestamp, dep_adv_items.item_key, dep_adv_items.item_value
FROM dep_advertisements
LEFT JOIN dep_adv_items
ON dep_advertisements.id = dep_adv_items.dep_adv_id`

func (h *Handler) ReadDeploymentAdvertisement(
	ctx context.Context,
	deploymentId string,
	reference string,
) (lib_models.DeploymentAdvertisement, error) {
	rows, err := h.sqlDB.QueryContext(
		ctx,
		selectDeploymentAdvertisementsStmt+" WHERE dep_id = ? AND ref = ?;",
		deploymentId,
		reference,
	)
	if err != nil {
		return lib_models.DeploymentAdvertisement{}, err
	}
	defer rows.Close()
	var depAdv lib_models.DeploymentAdvertisement
	for rows.Next() {
		var ts []uint8
		var itemKey string
		var itemValue sql.NullString
		err = rows.Scan(&depAdv.Id, &depAdv.DeploymentId, &depAdv.ModuleId, &depAdv.Reference, &ts, &itemKey, &itemValue)
		if err != nil {
			return lib_models.DeploymentAdvertisement{}, err
		}
		if depAdv.Timestamp.IsZero() {
			if depAdv.Timestamp, err = time.Parse(timeLayout, string(ts)); err != nil {
				logger.ErrorContext(ctx, "read deployment advertisement", slog_keys.DepAdvertisementId, depAdv.Id, slog_keys.Error, err)
			}
		}
		if depAdv.Items == nil {
			depAdv.Items = make(map[string]string)
		}
		depAdv.Items[itemKey] = itemValue.String
	}
	if err = rows.Err(); err != nil {
		return lib_models.DeploymentAdvertisement{}, err
	}
	return depAdv, nil
}

func (h *Handler) ReadDeploymentAdvertisements(
	ctx context.Context,
	filter lib_models.DeploymentAdvertisementsFilter,
) (map[string]lib_models.DeploymentAdvertisement, error) {
	fc, val := genDeploymentAdvertisementsFilter(filter)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		selectDeploymentAdvertisementsStmt+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	depAdvs := make(map[string]lib_models.DeploymentAdvertisement)
	for rows.Next() {
		var id string
		var depId string
		var modId string
		var reference string
		var ts []uint8
		var itemKey string
		var itemValue sql.NullString
		err = rows.Scan(&id, &depId, &modId, &reference, &ts, &itemKey, &itemValue)
		if err != nil {
			return nil, err
		}
		depAdv, ok := depAdvs[id]
		if !ok {
			if depAdv.Timestamp, err = time.Parse(timeLayout, string(ts)); err != nil {
				logger.ErrorContext(ctx, "read deployment advertisements", slog_keys.DepAdvertisementId, depAdv.Id, slog_keys.Error, err)
			}
			depAdv.Id = id
			depAdv.DeploymentId = depId
			depAdv.ModuleId = modId
			depAdv.Reference = reference
			depAdv.Items = make(map[string]string)
			depAdvs[id] = depAdv
		}
		depAdv.Items[itemKey] = itemValue.String
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return depAdvs, nil
}

func (h *Handler) WriteDeploymentAdvertisements(
	ctx context.Context,
	deploymentId string,
	advertisements []lib_models.DeploymentAdvertisement,
	incremental bool,
) error {
	tx, err := h.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if !incremental {
		_, err = tx.ExecContext(
			ctx,
			"DELETE FROM dep_advertisements WHERE dep_id = ?;",
			deploymentId,
		)
		if err != nil {
			return err
		}
	}
	for _, advertisement := range advertisements {
		if incremental {
			_, err = tx.ExecContext(
				ctx,
				"DELETE FROM dep_advertisements WHERE dep_id = ? AND ref = ?;",
				deploymentId,
				advertisement.Reference,
			)
			if err != nil {
				return err
			}
		}
		err = h.insertDeploymentAdvertisement(ctx, tx, deploymentId, advertisement)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) DeleteDeploymentAdvertisements(
	ctx context.Context,
	deploymentId string,
	filter lib_models.DeploymentAdvertisementsFilterReduced,
) error {
	fc, val := genDeleteDeploymentAdvertisementsFilter(deploymentId, filter)
	_, err := h.sqlDB.ExecContext(
		ctx,
		"DELETE FROM dep_advertisements"+fc+";",
		val...,
	)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) insertDeploymentAdvertisement(
	ctx context.Context,
	tx *sql.Tx,
	deploymentId string,
	advertisement lib_models.DeploymentAdvertisement,
) error {
	_, err := tx.ExecContext(
		ctx,
		"INSERT INTO dep_advertisements (id, dep_id, mod_id, ref, timestamp) VALUES (?, ?, ?, ?, ?)",
		advertisement.Id,
		deploymentId,
		advertisement.ModuleId,
		advertisement.Reference,
		advertisement.Timestamp,
	)
	if err != nil {
		return err
	}
	for key, value := range advertisement.Items {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO dep_adv_items (dep_adv_id, item_key, item_value) VALUES (?, ?, ?)",
			key,
			value,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func genDeploymentAdvertisementsFilter(filter lib_models.DeploymentAdvertisementsFilter) (string, []any) {
	var fc []string
	var val []any
	if filter.DeploymentId != "" {
		fc = append(fc, "dep_id = ?")
		val = append(val, filter.DeploymentId)
	}
	if len(filter.Ids) > 0 {
		ids := helper_slices.RemoveDuplicates(filter.Ids)
		fc = append(fc, "id IN ("+genQuestionMarks(len(ids))+")")
		for _, ref := range ids {
			val = append(val, ref)
		}
	}
	if len(filter.ModuleIds) > 0 {
		ids := helper_slices.RemoveDuplicates(filter.ModuleIds)
		fc = append(fc, "mod_id IN ("+genQuestionMarks(len(ids))+")")
		for _, ref := range ids {
			val = append(val, ref)
		}
	}
	if len(filter.References) > 0 {
		references := helper_slices.RemoveDuplicates(filter.References)
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

func genDeleteDeploymentAdvertisementsFilter(deploymentId string, filter lib_models.DeploymentAdvertisementsFilterReduced) (string, []any) {
	var fc []string
	var val []any
	if deploymentId != "" {
		fc = append(fc, "dep_id = ?")
		val = append(val, deploymentId)
	}
	if len(filter.Ids) > 0 {
		ids := helper_slices.RemoveDuplicates(filter.References)
		fc = append(fc, "id IN ("+genQuestionMarks(len(ids))+")")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if len(filter.References) > 0 {
		references := helper_slices.RemoveDuplicates(filter.References)
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
