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
	"errors"
	"fmt"
	"strings"

	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) CreateGlobalConfig(ctx context.Context, config models_handler_storage.GlobalConfig) error {
	if config.IsSlice {
		colName, values := getListConfigValsAndCol(config.ConfigValue)
		stmt := fmt.Sprintf("INSERT INTO global_configs (id, name, ord, is_list, %s) VALUES (?, ?, ?, ?, ?)", colName)
		for i, value := range values {
			_, err := h.sqlDB.ExecContext(ctx, stmt, config.Id, config.Name, i, true, value)
			if err != nil {
				return err
			}
		}
	} else {
		colName, value := getConfigValAndCol(config.ConfigValue)
		_, err := h.sqlDB.ExecContext(ctx, fmt.Sprintf("INSERT INTO global_configs (id, name, ord, is_list, %s) VALUES (?, ?, ?, ?, ?)", colName), config.Id, config.Name, 0, false, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) ReadGlobalConfig(ctx context.Context, id string) (models_handler_storage.GlobalConfig, error) {
	globalConfigs, err := h.ReadGlobalConfigs(ctx, []string{id})
	if err != nil {
		return models_handler_storage.GlobalConfig{}, err
	}
	if len(globalConfigs) == 0 {
		return models_handler_storage.GlobalConfig{}, models_error.NotFoundErr
	}
	return globalConfigs[id], nil
}

func (h *Handler) ReadGlobalConfigs(ctx context.Context, ids []string) (map[string]models_handler_storage.GlobalConfig, error) {
	fc, val := genGlobalConfigsFilter(ids)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT id, name, is_list, v_string, v_int, v_float, v_bool, ord FROM global_configs"+fc+" ORDER BY is_list, id, ord;",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	globalConfigs := make(map[string]models_handler_storage.GlobalConfig)
	for rows.Next() {
		var id string
		var name string
		var isList bool
		var vString sql.NullString
		var vInt sql.NullInt64
		var vFloat sql.NullFloat64
		var vBool sql.NullBool
		var ord int
		err = rows.Scan(&id, &name, &isList, &vString, &vInt, &vFloat, &vBool, &ord)
		if err != nil {
			return nil, err
		}
		config, ok := globalConfigs[id]
		var dataType int
		if isList {
			switch {
			case vString.Valid:
				config.StringSlice = append(config.StringSlice, vString.String)
				dataType = models_handler_storage.StringType
			case vInt.Valid:
				config.Int64Slice = append(config.Int64Slice, vInt.Int64)
				dataType = models_handler_storage.Int64Type
			case vFloat.Valid:
				config.Float64Slice = append(config.Float64Slice, vFloat.Float64)
				dataType = models_handler_storage.Float64Type
			case vBool.Valid:
				config.BoolSlice = append(config.BoolSlice, vBool.Bool)
				dataType = models_handler_storage.BoolType
			}
		} else {
			switch {
			case vString.Valid:
				config.String = vString.String
				dataType = models_handler_storage.StringType
			case vInt.Valid:
				config.Int64 = vInt.Int64
				dataType = models_handler_storage.Int64Type
			case vFloat.Valid:
				config.Float64 = vFloat.Float64
				dataType = models_handler_storage.Float64Type
			case vBool.Valid:
				config.Bool = vBool.Bool
				dataType = models_handler_storage.BoolType
			}
		}
		if !ok {
			config.Id = id
			config.Name = name
			config.IsSlice = isList
			config.DataType = dataType
		} else {
			if !config.IsSlice {
				return nil, errors.New("config type mismatch")
			}
			if dataType != config.DataType {
				return nil, errors.New("data type mismatch")
			}
		}
		globalConfigs[id] = config
	}
	return globalConfigs, nil
}

func (h *Handler) UpdateGlobalConfig(ctx context.Context, config models_handler_storage.GlobalConfig) error {
	tx, err := h.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		err = tx.Rollback()
	}()
	_, err = tx.ExecContext(ctx, "DELETE FROM global_configs WHERE id = ?", config.Id)
	if err != nil {
		return err
	}
	if config.IsSlice {
		colName, values := getListConfigValsAndCol(config.ConfigValue)
		stmt := fmt.Sprintf("INSERT INTO global_configs (id, name, ord, is_list, %s) VALUES (?, ?, ?, ?, ?)", colName)
		for i, value := range values {
			_, err = tx.ExecContext(ctx, stmt, config.Id, config.Name, i, true, value)
			if err != nil {
				return err
			}
		}
	} else {
		colName, value := getConfigValAndCol(config.ConfigValue)
		_, err = tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO global_configs (id, name, ord, is_list, %s) VALUES (?, ?, ?, ?, ?)", colName), config.Id, config.Name, 0, false, value)
		if err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (h *Handler) DeleteGlobalConfig(ctx context.Context, id string) error {
	return h.DeleteGlobalConfigs(ctx, []string{id})
}

func (h *Handler) DeleteGlobalConfigs(ctx context.Context, ids []string) error {
	ids = helper_slices.RemoveDuplicates(ids)
	_, err := h.sqlDB.ExecContext(
		ctx,
		"DELETE FROM global_configs WHERE id IN ("+genQuestionMarks(len(ids))+")",
		helper_slices.ToAny(ids)...,
	)
	if err != nil {
		return err
	}
	return nil
}

func genGlobalConfigsFilter(ids []string) (string, []any) {
	var fc []string
	var val []any
	if len(ids) > 0 {
		ids = helper_slices.RemoveDuplicates(ids)
		fc = append(fc, "id IN ("+genQuestionMarks(len(ids))+")")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}

func getConfigValAndCol(config models_handler_storage.ConfigValue) (colName string, value any) {
	switch config.DataType {
	case models_handler_storage.StringType:
		colName = "v_string"
		value = config.String
	case models_handler_storage.Int64Type:
		colName = "v_int"
		value = config.Int64
	case models_handler_storage.Float64Type:
		colName = "v_float"
		value = config.Float64
	case models_handler_storage.BoolType:
		colName = "v_bool"
		value = config.Bool
	}
	return
}

func getListConfigValsAndCol(config models_handler_storage.ConfigValue) (colName string, values []any) {
	switch config.DataType {
	case models_handler_storage.StringType:
		colName = "v_string"
		values = helper_slices.ToAny(config.StringSlice)
	case models_handler_storage.Int64Type:
		colName = "v_int"
		values = helper_slices.ToAny(config.Int64Slice)
	case models_handler_storage.Float64Type:
		colName = "v_float"
		values = helper_slices.ToAny(config.Float64Slice)
	case models_handler_storage.BoolType:
		colName = "v_bool"
		values = helper_slices.ToAny(config.BoolSlice)
	}
	return
}
