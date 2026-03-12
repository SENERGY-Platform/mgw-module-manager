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
	"strings"
	"time"

	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) Module(ctx context.Context, id string) (models_handler_storage.Module, error) {
	modules, err := h.Modules(ctx, models_handler_storage.ModulesFilter{Ids: []string{id}})
	if err != nil {
		return models_handler_storage.Module{}, err
	}
	if len(modules) == 0 {
		return models_handler_storage.Module{}, models_error.NotFoundErr
	}
	return modules[id], nil
}

func (h *Handler) Modules(ctx context.Context, filter models_handler_storage.ModulesFilter) (map[string]models_handler_storage.Module, error) {
	fc, val := genModulesFilter(filter)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT id, dir, source, channel, added, updated FROM modules"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	mods := make(map[string]models_handler_storage.Module)
	for rows.Next() {
		var mod models_handler_storage.Module
		var at, ut []uint8
		err = rows.Scan(&mod.Id, &mod.DirName, &mod.Source, &mod.Channel, &at, &ut)
		if err != nil {
			return nil, err
		}
		if mod.Added, err = time.Parse(timeLayout, string(at)); err != nil {
			return nil, err
		}
		if mod.Updated, err = time.Parse(timeLayout, string(ut)); err != nil {
			return nil, err
		}
		mods[mod.Id] = mod
	}
	return mods, nil
}

func (h *Handler) CreateModule(ctx context.Context, mod models_handler_storage.Module) error {
	_, err := h.sqlDB.ExecContext(
		ctx,
		"INSERT INTO modules (id, dir, source, channel, added, updated) VALUES (?, ?, ?, ?, ?, ?);",
		mod.Id,
		mod.DirName,
		mod.Source,
		mod.Channel,
		mod.Added,
		mod.Updated,
	)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) UpdateModule(ctx context.Context, mod models_handler_storage.Module) error {
	res, err := h.sqlDB.ExecContext(ctx, "UPDATE modules SET dir = ?, source = ?, channel = ?, updated = ? WHERE id = ?;", mod.DirName, mod.Source, mod.Channel, mod.Updated, mod.Id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return models_error.NotFoundErr
	}
	return nil
}

func (h *Handler) DeleteModule(ctx context.Context, id string) error {
	res, err := h.sqlDB.ExecContext(ctx, "DELETE FROM modules WHERE id = ?;", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return models_error.NotFoundErr
	}
	return nil
}

func genModulesFilter(filter models_handler_storage.ModulesFilter) (string, []any) {
	var fc []string
	var val []any
	if len(filter.Ids) > 0 {
		ids := helper_slices.RemoveDuplicates(filter.Ids)
		fc = append(fc, "id IN ("+genQuestionMarks(len(ids))+")")
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
