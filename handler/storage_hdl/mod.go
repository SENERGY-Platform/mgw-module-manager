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
	"github.com/SENERGY-Platform/mgw-module-manager/handler/storage_hdl/dep_util"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/model"
	"strings"
	"time"
)

func (h *Handler) ListMod(ctx context.Context, filter model.ModFilter, dependencyInfo bool) (map[string]model.Module, error) {
	q := "SELECT `id`, `dir`, `modfile`, `added`, `updated` FROM `modules`"
	fc, val := genModFilter(filter)
	if fc != "" {
		q += fc
	}
	rows, err := h.db.QueryContext(ctx, q, val...)
	if err != nil {
		return nil, lib_model.NewInternalError(err)
	}
	defer rows.Close()
	modules := make(map[string]model.Module)
	for rows.Next() {
		var id string
		var mod model.Module
		var at, ut []uint8
		if err = rows.Scan(&id, &mod.Dir, &mod.ModFile, &at, &ut); err != nil {
			return nil, lib_model.NewInternalError(err)
		}
		if dependencyInfo {
			if mod.RequiredMod, err = dep_util.SelectRequiredMod(ctx, h.db, id); err != nil {
				return nil, lib_model.NewInternalError(err)
			}
			if mod.ModRequiring, err = dep_util.SelectModRequiring(ctx, h.db, id); err != nil {
				return nil, lib_model.NewInternalError(err)
			}
		}
		if mod.Added, err = time.Parse(tLayout, string(at)); err != nil {
			return nil, lib_model.NewInternalError(err)
		}
		if mod.Updated, err = time.Parse(tLayout, string(ut)); err != nil {
			return nil, lib_model.NewInternalError(err)
		}
		modules[id] = mod
	}
	return modules, nil
}

func (h *Handler) ReadMod(ctx context.Context, mID string, dependencyInfo bool) (model.Module, error) {
	row := h.db.QueryRowContext(ctx, "SELECT `dir`, `modfile`, `added`, `updated` FROM `modules` WHERE `id` = ?", mID)
	var mod model.Module
	var at, ut []uint8
	err := row.Scan(&mod.Dir, &mod.ModFile, &at, &ut)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Module{}, lib_model.NewNotFoundError(err)
		}
		return model.Module{}, lib_model.NewInternalError(err)
	}
	if dependencyInfo {
		if mod.RequiredMod, err = dep_util.SelectRequiredMod(ctx, h.db, mID); err != nil {
			return model.Module{}, lib_model.NewInternalError(err)
		}
		if mod.ModRequiring, err = dep_util.SelectModRequiring(ctx, h.db, mID); err != nil {
			return model.Module{}, lib_model.NewInternalError(err)
		}
	}
	return mod, nil
}

func (h *Handler) CreateMod(ctx context.Context, txItf driver.Tx, mod model.Module) error {
	execContext := h.db.ExecContext
	if txItf != nil {
		tx := txItf.(*sql.Tx)
		execContext = tx.ExecContext
	}
	res, err := execContext(ctx, "INSERT INTO `modules` (`id`, `dir`, `modfile`, `added`, `updated`) VALUES (?, ?, ?, ?, ?, ?)", mod.ID, mod.Dir, mod.ModFile, mod.Added, mod.Updated)
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

func (h *Handler) UpdateMod(ctx context.Context, txItf driver.Tx, mod model.Module) error {
	execContext := h.db.ExecContext
	if txItf != nil {
		tx := txItf.(*sql.Tx)
		execContext = tx.ExecContext
	}
	res, err := execContext(ctx, "UPDATE `modules` SET `dir` = ?, `modfile` = ?, `updated` = ? WHERE `id` = ?", mod.Dir, mod.ModFile, mod.Updated, mod.ID)
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

func (h *Handler) DeleteMod(ctx context.Context, txItf driver.Tx, mID string) error {
	execContext := h.db.ExecContext
	if txItf != nil {
		tx := txItf.(*sql.Tx)
		execContext = tx.ExecContext
	}
	res, err := execContext(ctx, "DELETE FROM `modules` WHERE `id` = ?", mID)
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

func genModFilter(filter model.ModFilter) (string, []any) {
	var fc []string
	var val []any
	if len(filter.IDs) > 0 {
		ids := removeDuplicates(filter.IDs)
		fc = append(fc, "`id` IN ("+strings.Repeat("?, ", len(ids)-1)+"?)")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}
