/*
 * Copyright 2023 InfAI (CC SES)
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

package storage_hdl

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"strings"
	"time"
)

func (h *Handler) ListAuxDep(ctx context.Context, dID string, filter lib_model.AuxDepFilter, assets bool) (map[string]lib_model.AuxDeployment, error) {
	q := "SELECT `id`, `dep_id`, `image`, `created`, `updated`, `ref`, `name`, `enabled`, `command`, `pseudo_tty` FROM `aux_deployments`"
	fc, val := genAuxDepFilter(dID, filter)
	if fc != "" {
		q += fc
	}
	q += " ORDER BY `created`"
	rows, err := h.db.QueryContext(ctx, q, val...)
	if err != nil {
		return nil, lib_model.NewInternalError(err)
	}
	defer rows.Close()
	auxDeployments := make(map[string]lib_model.AuxDeployment)
	for rows.Next() {
		var auxDep lib_model.AuxDeployment
		var ct, ut []uint8
		if err = rows.Scan(&auxDep.ID, &auxDep.DepID, &auxDep.Image, &ct, &ut, &auxDep.Ref, &auxDep.Name, &auxDep.Enabled, &auxDep.RunConfig.Command, &auxDep.RunConfig.PseudoTTY); err != nil {
			return nil, lib_model.NewInternalError(err)
		}
		auxDep.Created, err = time.Parse(tLayout, string(ct))
		if err != nil {
			return nil, lib_model.NewInternalError(err)
		}
		auxDep.Updated, err = time.Parse(tLayout, string(ut))
		if err != nil {
			return nil, lib_model.NewInternalError(err)
		}
		auxDep.Labels, err = selectAuxDepLabels(ctx, h.db, auxDep.ID)
		if err != nil {
			return nil, err
		}
		auxDep.Container, err = selectAuxDepContainer(ctx, h.db, auxDep.ID)
		if err != nil {
			return nil, err
		}
		if assets {
			if auxDep.Configs, err = selectAuxDepConfigs(ctx, h.db, auxDep.ID); err != nil {
				return nil, err
			}
			if auxDep.Volumes, err = selectAuxDepVolumes(ctx, h.db, auxDep.ID); err != nil {
				return nil, err
			}
		}
		auxDeployments[auxDep.ID] = auxDep
	}
	if err = rows.Err(); err != nil {
		return nil, lib_model.NewInternalError(err)
	}
	return auxDeployments, nil
}

func (h *Handler) ReadAuxDep(ctx context.Context, aID string, assets bool) (lib_model.AuxDeployment, error) {
	row := h.db.QueryRowContext(ctx, "SELECT `id`, `dep_id`, `image`, `created`, `updated`, `ref`, `name`, `enabled`, `command`, `pseudo_tty` FROM `aux_deployments` WHERE `id` = ?", aID)
	var auxDep lib_model.AuxDeployment
	var ct, ut []uint8
	err := row.Scan(&auxDep.ID, &auxDep.DepID, &auxDep.Image, &ct, &ut, &auxDep.Ref, &auxDep.Name, &auxDep.Enabled, &auxDep.RunConfig.Command, &auxDep.RunConfig.PseudoTTY)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return lib_model.AuxDeployment{}, lib_model.NewNotFoundError(err)
		}
		return lib_model.AuxDeployment{}, lib_model.NewInternalError(err)
	}
	auxDep.Created, err = time.Parse(tLayout, string(ct))
	if err != nil {
		return lib_model.AuxDeployment{}, lib_model.NewInternalError(err)
	}
	auxDep.Updated, err = time.Parse(tLayout, string(ut))
	if err != nil {
		return lib_model.AuxDeployment{}, lib_model.NewInternalError(err)
	}
	auxDep.Labels, err = selectAuxDepLabels(ctx, h.db, auxDep.ID)
	if err != nil {
		return lib_model.AuxDeployment{}, err
	}
	auxDep.Container, err = selectAuxDepContainer(ctx, h.db, auxDep.ID)
	if err != nil {
		return lib_model.AuxDeployment{}, err
	}
	if assets {
		if auxDep.Configs, err = selectAuxDepConfigs(ctx, h.db, auxDep.ID); err != nil {
			return lib_model.AuxDeployment{}, err
		}
		if auxDep.Volumes, err = selectAuxDepVolumes(ctx, h.db, auxDep.ID); err != nil {
			return lib_model.AuxDeployment{}, err
		}
	}
	return auxDep, nil
}

func (h *Handler) CreateAuxDep(ctx context.Context, txItf driver.Tx, auxDep lib_model.AuxDepBase) (string, error) {
	var tx *sql.Tx
	if txItf != nil {
		tx = txItf.(*sql.Tx)
	} else {
		var e error
		if tx, e = h.db.BeginTx(ctx, nil); e != nil {
			return "", lib_model.NewInternalError(e)
		}
		defer tx.Rollback()
	}
	res, err := tx.ExecContext(ctx, "INSERT INTO `aux_deployments` (`id`, `dep_id`, `image`, `created`, `updated`, `ref`, `name`, `enabled`, `command`, `pseudo_tty`) VALUES (UUID(), ?, ?, ?, ?, ?, ?, ?, ?, ?)", auxDep.DepID, auxDep.Image, auxDep.Created, auxDep.Updated, auxDep.Ref, auxDep.Name, auxDep.Enabled, auxDep.RunConfig.Command, auxDep.RunConfig.PseudoTTY)
	if err != nil {
		return "", lib_model.NewInternalError(err)
	}
	i, err := res.LastInsertId()
	if err != nil {
		return "", lib_model.NewInternalError(err)
	}
	row := tx.QueryRowContext(ctx, "SELECT `id` FROM `aux_deployments` WHERE `index` = ?", i)
	var id string
	if err = row.Scan(&id); err != nil {
		return "", lib_model.NewInternalError(err)
	}
	if id == "" {
		return "", lib_model.NewInternalError(errors.New("generating id failed"))
	}
	if len(auxDep.Labels) > 0 {
		if err = insertAuxDepLabels(ctx, tx.PrepareContext, id, auxDep.Labels); err != nil {
			return "", err
		}
	}
	if len(auxDep.Configs) > 0 {
		if err = insertAuxDepConfigs(ctx, tx.PrepareContext, id, auxDep.Configs); err != nil {
			return "", err
		}
	}
	if len(auxDep.Volumes) > 0 {
		if err = insertAuxDepVolumes(ctx, tx.PrepareContext, id, auxDep.Volumes); err != nil {
			return "", err
		}
	}
	if txItf == nil {
		if err = tx.Commit(); err != nil {
			return "", lib_model.NewInternalError(err)
		}
	}
	return id, nil
}

func (h *Handler) UpdateAuxDep(ctx context.Context, txItf driver.Tx, auxDep lib_model.AuxDepBase) error {
	var tx *sql.Tx
	if txItf != nil {
		tx = txItf.(*sql.Tx)
	} else {
		var e error
		if tx, e = h.db.BeginTx(ctx, nil); e != nil {
			return lib_model.NewInternalError(e)
		}
		defer tx.Rollback()
	}
	res, err := tx.ExecContext(ctx, "UPDATE `aux_deployments` SET `image` = ?, `updated` = ?, `name` = ?, `enabled` = ?, `command` = ?, `pseudo_tty` = ? WHERE `id` = ?", auxDep.Image, auxDep.Updated, auxDep.Name, auxDep.Enabled, auxDep.RunConfig.Command, auxDep.RunConfig.PseudoTTY, auxDep.ID)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	if n < 1 {
		return lib_model.NewNotFoundError(errors.New("no rows affected"))
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM `aux_labels` WHERE `aux_id` = ?", auxDep.ID)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM `aux_configs` WHERE `aux_id` = ?", auxDep.ID)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM `aux_volumes` WHERE `aux_id` = ?", auxDep.ID)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	if len(auxDep.Labels) > 0 {
		if err = insertAuxDepLabels(ctx, tx.PrepareContext, auxDep.ID, auxDep.Labels); err != nil {
			return err
		}
	}
	if len(auxDep.Configs) > 0 {
		if err = insertAuxDepConfigs(ctx, tx.PrepareContext, auxDep.ID, auxDep.Configs); err != nil {
			return err
		}
	}
	if len(auxDep.Volumes) > 0 {
		if err = insertAuxDepVolumes(ctx, tx.PrepareContext, auxDep.ID, auxDep.Volumes); err != nil {
			return err
		}
	}
	if txItf == nil {
		if err = tx.Commit(); err != nil {
			return lib_model.NewInternalError(err)
		}
	}
	return nil
}

func (h *Handler) DeleteAuxDep(ctx context.Context, txItf driver.Tx, aID string) error {
	execContext := h.db.ExecContext
	if txItf != nil {
		tx := txItf.(*sql.Tx)
		execContext = tx.ExecContext
	}
	res, err := execContext(ctx, "DELETE FROM `aux_deployments` WHERE `id` = ?", aID)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	if n < 1 {
		return lib_model.NewNotFoundError(errors.New("no rows affected"))
	}
	return nil
}

func (h *Handler) CreateAuxDepContainer(ctx context.Context, txItf driver.Tx, aID string, auxDepContainer lib_model.AuxDepContainer) error {
	execContext := h.db.ExecContext
	if txItf != nil {
		tx := txItf.(*sql.Tx)
		execContext = tx.ExecContext
	}
	_, err := execContext(ctx, "INSERT INTO `aux_containers` (`aux_id`, `ctr_id`, `alias`) VALUES (?, ?, ?)", aID, auxDepContainer.ID, auxDepContainer.Alias)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) DeleteAuxDepContainer(ctx context.Context, txItf driver.Tx, aID string) error {
	execContext := h.db.ExecContext
	if txItf != nil {
		tx := txItf.(*sql.Tx)
		execContext = tx.ExecContext
	}
	_, err := execContext(ctx, "DELETE FROM `aux_containers` WHERE `aux_id` = ?", aID)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	return nil
}

func selectAuxDepContainer(ctx context.Context, db *sql.DB, id string) (lib_model.AuxDepContainer, error) {
	row := db.QueryRowContext(ctx, "SELECT `ctr_id`, `alias` FROM `aux_containers` WHERE `aux_id` = ?", id)
	var auxDepCtr lib_model.AuxDepContainer
	err := row.Scan(&auxDepCtr.ID, &auxDepCtr.Alias)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return lib_model.AuxDepContainer{}, lib_model.NewNotFoundError(err)
		}
		return lib_model.AuxDepContainer{}, lib_model.NewInternalError(err)
	}
	return auxDepCtr, nil
}

func insertAuxDepLabels(ctx context.Context, pf func(ctx context.Context, query string) (*sql.Stmt, error), id string, m map[string]string) error {
	return insertStrMap(ctx, pf, "INSERT INTO `aux_labels` (`aux_id`, `name`, `value`) VALUES (?, ?, ?)", id, m)
}

func insertAuxDepConfigs(ctx context.Context, pf func(ctx context.Context, query string) (*sql.Stmt, error), id string, m map[string]string) error {
	return insertStrMap(ctx, pf, "INSERT INTO `aux_configs` (`aux_id`, `ref`, `value`) VALUES (?, ?, ?)", id, m)
}

func insertAuxDepVolumes(ctx context.Context, pf func(ctx context.Context, query string) (*sql.Stmt, error), id string, m map[string]string) error {
	return insertStrMap(ctx, pf, "INSERT INTO `aux_volumes` (`aux_id`, `name`, `mnt_point`) VALUES (?, ?, ?)", id, m)
}

func genAuxDepFilter(dID string, filter lib_model.AuxDepFilter) (string, []any) {
	var str string
	var val []any
	tc := 0
	if len(filter.Labels) > 0 {
		for n, v := range filter.Labels {
			var fl []string
			fl = append(fl, "`name` = ?")
			val = append(val, n)
			fl = append(fl, "`value` = ?")
			val = append(val, v)
			if tc == 0 {
				str = fmt.Sprintf("SELECT t%d.* FROM (SELECT `aux_id` FROM `aux_labels` WHERE %s) t%d", tc, strings.Join(fl, " AND "), tc)
			} else {
				str += fmt.Sprintf(" INNER JOIN (SELECT `aux_id` FROM `aux_labels` WHERE %s) t%d ON t%d.aux_id = t%d.aux_id", strings.Join(fl, " AND "), tc, tc-1, tc)
			}
			tc += 1
		}
		str = " `id` IN (" + str + ")"
	}
	if str != "" {
		str += " AND"
	}
	str += " `dep_id` = ?"
	val = append(val, dID)
	if filter.Image != "" {
		str += " AND `image` = ?"
		val = append(val, filter.Image)
	}
	if filter.Enabled != 0 {
		str += " AND `enabled` = ?"
		if filter.Enabled > 0 {
			val = append(val, true)
		} else {
			val = append(val, false)
		}
	}
	if len(val) > 0 {
		return " WHERE" + str, val
	}
	return "", nil
}

func selectAuxDepLabels(ctx context.Context, db *sql.DB, id string) (map[string]string, error) {
	return selectStrMap(ctx, db.QueryContext, "SELECT `name`, `value` FROM `aux_labels` WHERE `aux_id` = ?", id)
}

func selectAuxDepConfigs(ctx context.Context, db *sql.DB, id string) (map[string]string, error) {
	return selectStrMap(ctx, db.QueryContext, "SELECT `ref`, `value` FROM `aux_configs` WHERE `aux_id` = ?", id)
}

func selectAuxDepVolumes(ctx context.Context, db *sql.DB, id string) (map[string]string, error) {
	return selectStrMap(ctx, db.QueryContext, "SELECT `name`, `mnt_point` FROM `aux_volumes` WHERE `aux_id` = ?", id)
}

func selectStrMap(ctx context.Context, qf func(ctx context.Context, query string, args ...any) (*sql.Rows, error), query, id string) (map[string]string, error) {
	rows, err := qf(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]string)
	for rows.Next() {
		var key string
		var val string
		if err = rows.Scan(&key, &val); err != nil {
			return nil, lib_model.NewInternalError(err)
		}
		m[key] = val
	}
	if err = rows.Err(); err != nil {
		return nil, lib_model.NewInternalError(err)
	}
	return m, nil
}

func insertStrMap(ctx context.Context, pf func(ctx context.Context, query string) (*sql.Stmt, error), query, id string, m map[string]string) error {
	stmt, err := pf(ctx, query)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	defer stmt.Close()
	for key, val := range m {
		if _, err = stmt.ExecContext(ctx, id, key, val); err != nil {
			return lib_model.NewInternalError(err)
		}
	}
	return nil
}
