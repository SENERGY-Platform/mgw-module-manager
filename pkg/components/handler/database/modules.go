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

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
)

func (h *Handler) ReadModule(ctx context.Context, id string) (pkg_models.DatabaseModule, error) {
	modules, err := h.ReadModules(ctx, pkg_models.ModulesFilter{Ids: []string{id}})
	if err != nil {
		return pkg_models.DatabaseModule{}, err
	}
	if len(modules) == 0 {
		return pkg_models.DatabaseModule{}, lib_errors.New[lib_errors.ErrNotFound]("module not found")
	}
	return modules[id], nil
}

func (h *Handler) ReadModules(ctx context.Context, filter pkg_models.ModulesFilter) (map[string]pkg_models.DatabaseModule, error) {
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
	mods := make(map[string]pkg_models.DatabaseModule)
	for rows.Next() {
		var mod pkg_models.DatabaseModule
		var at, ut []uint8
		err = rows.Scan(&mod.Id, &mod.DirName, &mod.Source, &mod.Channel, &at, &ut)
		if err != nil {
			return nil, err
		}
		if mod.Added, err = time.Parse(timeLayout, string(at)); err != nil {
			logger.Error("read modules", slog_keys.DeploymentId, mod.Id, slog_keys.Error, err)
		}
		if mod.Updated, err = time.Parse(timeLayout, string(ut)); err != nil {
			logger.Error("read modules", slog_keys.DeploymentId, mod.Id, slog_keys.Error, err)
		}
		mods[mod.Id] = mod
	}
	return mods, nil
}

func (h *Handler) CreateModule(ctx context.Context, mod pkg_models.DatabaseModule) error {
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

func (h *Handler) UpdateModule(ctx context.Context, mod pkg_models.DatabaseModule) error {
	_, err := h.sqlDB.ExecContext(ctx, "UPDATE modules SET dir = ?, source = ?, channel = ?, updated = ? WHERE id = ?;", mod.DirName, mod.Source, mod.Channel, mod.Updated, mod.Id)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) DeleteModule(ctx context.Context, id string) error {
	_, err := h.sqlDB.ExecContext(ctx, "DELETE FROM modules WHERE id = ?;", id)
	if err != nil {
		return err
	}
	return nil
}

func genModulesFilter(filter pkg_models.ModulesFilter) (string, []any) {
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
