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
	"fmt"
	"strings"

	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) CreateGlobalConfig(ctx context.Context, config models_handler_storage.GlobalConfig) (err error) {
	tx, err := h.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer func() {
		err = tx.Rollback()
	}()
	_, err = tx.ExecContext(
		ctx,
		"INSERT INTO global_configs (id, name, data_type, is_list) VALUES (?, ?, ?, ?)",
		config.Id,
		config.Name,
		config.DataType,
		config.IsSlice,
	)
	err = createConfigValues(ctx, tx, "global_config_values", config)
	if err != nil {
		return
	}
	err = tx.Commit()
	return
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
	var rows *sql.Rows
	var err error
	if len(ids) > 0 {
		ids = helper_slices.RemoveDuplicates(ids)
		rows, err = h.sqlDB.QueryContext(
			ctx,
			"SELECT * FROM ("+genSelectConfigsStmt("global_configs", "global_config_values", "name")+") AS SUB WHERE SUB.id IN ("+genQuestionMarks(len(ids))+");",
			helper_slices.ToAny(ids)...,
		)
	} else {
		rows, err = h.sqlDB.QueryContext(ctx, genSelectConfigsStmt("global_configs", "global_config_values", "name")+";")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	globalConfigs := make(map[string]models_handler_storage.GlobalConfig)
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
			config.Name = name
			config.IsSlice = isList
			config.DataType = dataType
		}
		if isList {
			switch dataType {
			case models_handler_storage.StringType:
				config.StringSlice = append(config.StringSlice, vString.String)
			case models_handler_storage.Int64Type:
				config.Int64Slice = append(config.Int64Slice, vInt.Int64)
			case models_handler_storage.Float64Type:
				config.Float64Slice = append(config.Float64Slice, vFloat.Float64)
			case models_handler_storage.BoolType:
				config.BoolSlice = append(config.BoolSlice, vBool.Bool)
			}
		} else {
			switch dataType {
			case models_handler_storage.StringType:
				config.String = vString.String
			case models_handler_storage.Int64Type:
				config.Int64 = vInt.Int64
			case models_handler_storage.Float64Type:
				config.Float64 = vFloat.Float64
			case models_handler_storage.BoolType:
				config.Bool = vBool.Bool
			}
		}
		globalConfigs[id] = config
	}
	return globalConfigs, nil
}

func (h *Handler) UpdateGlobalConfig(ctx context.Context, config models_handler_storage.GlobalConfig) (err error) {
	tx, err := h.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer func() {
		err = tx.Rollback()
	}()
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
	err = createConfigValues(ctx, tx, "global_config_values", config)
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

const selectConfigsStmt = `SELECT _t1_.id, _t1_.data_type, _t1_.is_list, _t2_.v_string, _t2_.v_int, _t2_.v_float, _t2_.v_bool, _t2_.ord%s
FROM _t1_
LEFT JOIN _t2_
ON _t1_.id = _t2_.c_id ORDER BY is_list, _t1_.id, ord`

func genSelectConfigsStmt(t1, t2 string, t1Cols ...string) string {
	stmt := strings.ReplaceAll(strings.ReplaceAll(selectConfigsStmt, "_t1_", t1), "_t2_", t2)
	if len(t1Cols) > 0 {
		var cols []string
		for _, col := range t1Cols {
			cols = append(cols, t1+"."+col)
		}
		return fmt.Sprintf(stmt, ", "+strings.Join(cols, ", "))
	}
	return fmt.Sprintf(stmt, "")
}

func createConfigValues(ctx context.Context, tx *sql.Tx, tableName string, config models_handler_storage.GlobalConfig) error {
	if config.IsSlice {
		colName, values := getListConfigValsAndCol(config.Config)
		stmt := fmt.Sprintf("INSERT INTO %s (c_id, %s, ord) VALUES (?, ?, ?)", tableName, colName)
		for i, value := range values {
			_, err := tx.ExecContext(ctx, stmt, config.Id, value, i)
			if err != nil {
				return err
			}
		}
	} else {
		colName, value := getConfigValAndCol(config.Config)
		_, err := tx.ExecContext(
			ctx,
			fmt.Sprintf("INSERT INTO %s (c_id, %s, ord) VALUES (?, ?, ?)", tableName, colName),
			config.Id,
			value,
			0,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func getConfigValAndCol(config models_handler_storage.Config) (colName string, value any) {
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

func getListConfigValsAndCol(config models_handler_storage.Config) (colName string, values []any) {
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
