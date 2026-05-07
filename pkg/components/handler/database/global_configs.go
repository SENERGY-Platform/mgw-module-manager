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

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
)

func (h *Handler) CreateGlobalConfig(ctx context.Context, config pkg_models.Config) (err error) {
	tx, err := h.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(
		ctx,
		"INSERT INTO global_configs (id, name, data_type, is_list) VALUES (?, ?, ?, ?)",
		config.Id,
		config.Name,
		config.DataType,
		config.IsSlice,
	)
	if err != nil {
		return
	}
	err = createConfigValues(ctx, tx, "global_config_values", config.Id, config.Value)
	if err != nil {
		return
	}
	err = tx.Commit()
	return
}

func (h *Handler) ReadGlobalConfig(ctx context.Context, id string) (pkg_models.Config, error) {
	globalConfigs, err := h.ReadGlobalConfigs(ctx, []string{id})
	if err != nil {
		return pkg_models.Config{}, err
	}
	if len(globalConfigs) == 0 {
		return pkg_models.Config{}, lib_errors.New[lib_errors.ErrNotFound]("global config not found")
	}
	return globalConfigs[id], nil
}

func (h *Handler) ReadGlobalConfigs(ctx context.Context, ids []string) (map[string]pkg_models.Config, error) {
	rows, err := h.queryConfigs(ctx, ids, "global_configs", "global_config_values", "id", "name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	globalConfigs := make(map[string]pkg_models.Config)
	for rows.Next() {
		var id string
		var isList bool
		var dataType int
		var vString sql.NullString
		var vInt sql.NullInt64
		var vFloat sql.NullFloat64
		var vBool sql.NullBool
		var ord int
		var name string
		err = rows.Scan(&id, &dataType, &isList, &vString, &vInt, &vFloat, &vBool, &ord, &name)
		if err != nil {
			return nil, err
		}
		config, ok := globalConfigs[id]
		if !ok {
			config.Id = id
			config.IsSlice = isList
			config.DataType = dataType
			config.Name = name
		}
		if isList {
			switch dataType {
			case constants.ValueDataTypeString:
				config.StringSlice = append(config.StringSlice, vString.String)
			case constants.ValueDataTypeInt64:
				config.Int64Slice = append(config.Int64Slice, vInt.Int64)
			case constants.ValueDataTypeFloat64:
				config.Float64Slice = append(config.Float64Slice, vFloat.Float64)
			case constants.ValueDataTypeBool:
				config.BoolSlice = append(config.BoolSlice, vBool.Bool)
			}
		} else {
			switch dataType {
			case constants.ValueDataTypeString:
				config.String = vString.String
			case constants.ValueDataTypeInt64:
				config.Int64 = vInt.Int64
			case constants.ValueDataTypeFloat64:
				config.Float64 = vFloat.Float64
			case constants.ValueDataTypeBool:
				config.Bool = vBool.Bool
			}
		}
		globalConfigs[id] = config
	}
	return globalConfigs, nil
}

func (h *Handler) UpdateGlobalConfig(ctx context.Context, config pkg_models.Config) (err error) {
	tx, err := h.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(
		ctx,
		"UPDATE global_configs SET name = ?, data_type = ?, is_list = ? WHERE id = ?",
		config.Name,
		config.DataType,
		config.IsSlice,
	)
	if err != nil {
		return
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM global_config_values WHERE c_id = ?", config.Id)
	if err != nil {
		return
	}
	err = createConfigValues(ctx, tx, "global_config_values", config.Id, config.Value)
	if err != nil {
		return
	}
	err = tx.Commit()
	return
}

func (h *Handler) DeleteGlobalConfig(ctx context.Context, id string) error {
	return h.DeleteGlobalConfigs(ctx, []string{id})
}

func (h *Handler) DeleteGlobalConfigs(ctx context.Context, ids []string) error {
	fc, val := genDeleteGlobalConfigsFilter(ids)
	_, err := h.sqlDB.ExecContext(
		ctx,
		"DELETE FROM global_configs"+fc+";",
		val...,
	)
	if err != nil {
		return err
	}
	return nil
}

func genDeleteGlobalConfigsFilter(ids []string) (string, []any) {
	if len(ids) > 0 {
		ids = helper_slices.RemoveDuplicates(ids)
		return " WHERE id IN (" + genQuestionMarks(len(ids)) + ")", helper_slices.ToAny(ids)
	}
	return "", nil
}
