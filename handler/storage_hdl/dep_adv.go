/*
 * Copyright 2024 InfAI (CC SES)
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
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/model"
	"strings"
	"time"
)

func (h *Handler) ListDepAdv(ctx context.Context, filter model.DepAdvFilter) (map[string]model.DepAdvertisement, error) {
	q := "SELECT `id`, `dep_id`, `mod_id`, `origin`, `ref`, `timestamp` FROM `dep_advertisements`"
	fc, val := genDepAdvFilter(filter)
	if fc != "" {
		q += fc
	}
	rows, err := h.db.QueryContext(ctx, q, val...)
	if err != nil {
		return nil, lib_model.NewInternalError(err)
	}
	defer rows.Close()
	depAdvertisements := make(map[string]model.DepAdvertisement)
	for rows.Next() {
		var adv model.DepAdvertisement
		var ts []uint8
		if err = rows.Scan(&adv.ID, &adv.DepID, &adv.ModuleID, &adv.Origin, &adv.Ref, &ts); err != nil {
			return nil, lib_model.NewInternalError(err)
		}
		adv.Timestamp, err = time.Parse(tLayout, string(ts))
		if err != nil {
			return nil, lib_model.NewInternalError(err)
		}
		if adv.Items, err = selectDepAdvItems(ctx, h.db, adv.ID); err != nil {
			return nil, err
		}
		depAdvertisements[adv.ID] = adv
	}
	if err = rows.Err(); err != nil {
		return nil, lib_model.NewInternalError(err)
	}
	return depAdvertisements, nil
}

func (h *Handler) ReadDepAdv(ctx context.Context, dID, ref string) (model.DepAdvertisement, error) {
	row := h.db.QueryRowContext(ctx, "SELECT `id`, `dep_id`, `mod_id`, `origin`, `ref`, `timestamp` FROM `dep_advertisements` WHERE `dep_id` = ? AND `ref` = ?", dID, ref)
	var adv model.DepAdvertisement
	var ts []uint8
	err := row.Scan(&adv.ID, &adv.DepID, &adv.ModuleID, &adv.Origin, &adv.Ref, &ts)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.DepAdvertisement{}, lib_model.NewNotFoundError(err)
		}
		return model.DepAdvertisement{}, lib_model.NewInternalError(err)
	}
	adv.Timestamp, err = time.Parse(tLayout, string(ts))
	if err != nil {
		return model.DepAdvertisement{}, lib_model.NewInternalError(err)
	}
	if adv.Items, err = selectDepAdvItems(ctx, h.db, adv.ID); err != nil {
		return model.DepAdvertisement{}, err
	}
	return adv, nil
}

func (h *Handler) CreateDepAdv(ctx context.Context, txItf driver.Tx, adv model.DepAdvertisement) (string, error) {
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
	res, err := tx.ExecContext(ctx, "INSERT INTO `dep_advertisements` (`id`, `dep_id`, `mod_id`, `origin`, `ref`, `timestamp`) VALUES (UUID(), ?, ?, ?, ?, ?)", adv.DepID, adv.ModuleID, adv.Origin, adv.Ref, adv.Timestamp)
	if err != nil {
		return "", lib_model.NewInternalError(err)
	}
	i, err := res.LastInsertId()
	if err != nil {
		return "", lib_model.NewInternalError(err)
	}
	row := tx.QueryRowContext(ctx, "SELECT `id` FROM `dep_advertisements` WHERE `index` = ?", i)
	var id string
	if err = row.Scan(&id); err != nil {
		return "", lib_model.NewInternalError(err)
	}
	if id == "" {
		return "", lib_model.NewInternalError(errors.New("generating id failed"))
	}
	if len(adv.Items) > 0 {
		if err = insertDepAdvItems(ctx, tx.PrepareContext, id, adv.Items); err != nil {
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

func (h *Handler) DeleteDepAdv(ctx context.Context, txItf driver.Tx, dID, ref string) error {
	execContext := h.db.ExecContext
	if txItf != nil {
		tx := txItf.(*sql.Tx)
		execContext = tx.ExecContext
	}
	res, err := execContext(ctx, "DELETE FROM `dep_advertisements` WHERE `dep_id` = ? AND `ref` = ?", dID, ref)
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

func (h *Handler) DeleteAllDepAdv(ctx context.Context, txItf driver.Tx, dID string) error {
	execContext := h.db.ExecContext
	if txItf != nil {
		tx := txItf.(*sql.Tx)
		execContext = tx.ExecContext
	}
	res, err := execContext(ctx, "DELETE FROM `dep_advertisements` WHERE `dep_id` = ?", dID)
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

func selectDepAdvItems(ctx context.Context, db *sql.DB, id string) (map[string]string, error) {
	return selectStrMap(ctx, db.QueryContext, "SELECT `key`, `value` FROM `dep_adv_items` WHERE `adv_id` = ?", id)
}

func insertDepAdvItems(ctx context.Context, pf func(ctx context.Context, query string) (*sql.Stmt, error), id string, m map[string]string) error {
	return insertStrMap(ctx, pf, "INSERT INTO `dep_adv_items` (`adv_id`, `key`, `value`) VALUES (?, ?, ?)", id, m)
}

func genDepAdvFilter(filter model.DepAdvFilter) (string, []any) {
	var fc []string
	var val []any
	if filter.DepID != "" {
		fc = append(fc, "`dep_id` = ?")
		val = append(val, filter.DepID)
	}
	if filter.ModuleID != "" {
		fc = append(fc, "`mod_id` = ?")
		val = append(val, filter.ModuleID)
	}
	if filter.Origin != "" {
		fc = append(fc, "`origin` = ?")
		val = append(val, filter.Origin)
	}
	if filter.Ref != "" {
		fc = append(fc, "`ref` = ?")
		val = append(val, filter.Ref)
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}
