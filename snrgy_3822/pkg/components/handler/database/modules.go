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
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/storage"
	"strings"
	"time"
)

const selectFromModulesStatement = "SELECT id, dir, source, channel, added, updated FROM modules"

func (h *Handler) ListMod(ctx context.Context, filter models_storage.ModuleFilter) (map[string]models_storage.Module, error) {
	fc, val := genModFilter(filter)
	rows, err := h.sqlDB.QueryContext(ctx, selectFromModulesStatement+fc+";", val)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	mods := make(map[string]models_storage.Module)
	for rows.Next() {
		var mod models_storage.Module
		var at, ut []uint8
		err = rows.Scan(&mod.ID, &mod.DirName, &mod.Source, &mod.Channel, &at, &ut)
		if err != nil {
			return nil, err
		}
		if mod.Added, err = time.Parse(timeLayout, string(at)); err != nil {
			return nil, err
		}
		if mod.Updated, err = time.Parse(timeLayout, string(ut)); err != nil {
			return nil, err
		}
		mods[mod.ID] = mod
	}
	return mods, nil
}

func (h *Handler) ReadMod(ctx context.Context, id string) (models_storage.Module, error) {
	row := h.sqlDB.QueryRowContext(ctx, selectFromModulesStatement+"WHERE id = ?;", id)
	var mod models_storage.Module
	var at, ut []uint8
	err := row.Scan(&mod.ID, &mod.DirName, &mod.Source, &mod.Channel, &at, &ut)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models_storage.Module{}, models_error.NotFoundErr
		}
		return models_storage.Module{}, err
	}
	if mod.Added, err = time.Parse(timeLayout, string(at)); err != nil {
		return models_storage.Module{}, err
	}
	if mod.Updated, err = time.Parse(timeLayout, string(ut)); err != nil {
		return models_storage.Module{}, err
	}
	return mod, nil
}

func (h *Handler) CreateMod(ctx context.Context, mod models_storage.Module) error {
	_, err := h.sqlDB.ExecContext(
		ctx,
		"INSERT INTO modules (id, dir, source, channel, added, updated) VALUES (?, ?, ?, ?, ?, ?)",
		mod.ID,
		mod.DirName,
		mod.Channel,
		mod.Source,
		mod.Added,
		mod.Updated,
	)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) UpdateMod(ctx context.Context, mod models_storage.Module) error {
	_, err := h.sqlDB.ExecContext(ctx, "UPDATE modules SET dir = ?, source = ?, channel = ?, updated = ? WHERE id = ?", mod.DirName, mod.Source, mod.Channel, mod.Updated, mod.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models_error.NotFoundErr
		}
		return err
	}
	return nil
}

func (h *Handler) DeleteMod(ctx context.Context, id string) error {
	_, err := h.sqlDB.ExecContext(ctx, "DELETE FROM modules WHERE id = ?", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models_error.NotFoundErr
		}
		return err
	}
	return nil
}

func genModFilter(filter models_storage.ModuleFilter) (string, []any) {
	var fc []string
	var val []any
	if len(filter.IDs) > 0 {
		ids := removeDuplicates(filter.IDs)
		fc = append(fc, "id IN ("+genQuestionMarks(len(filter.IDs))+")")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if filter.Source != "" {
		fc = append(fc, "source = ?")
		val = append(val, filter.Source)
	}
	if filter.Channel != "" {
		fc = append(fc, "channel = ?")
		val = append(val, filter.Channel)
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}
