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

package handler_database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) queryConfigs(ctx context.Context, ids []string, t1, t2 string, filterIdCol string, t1Cols ...string) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error
	if len(ids) > 0 {
		ids = helper_slices.RemoveDuplicates(ids)
		rows, err = h.sqlDB.QueryContext(
			ctx,
			genSelectConfigsFilterIdsStmt(t1, t2, len(ids), filterIdCol, t1Cols...)+";",
			helper_slices.ToAny(ids)...,
		)
	} else {
		rows, err = h.sqlDB.QueryContext(ctx, genSelectConfigsStmt(t1, t2, t1Cols...)+";")
	}
	if err != nil {
		return nil, err
	}
	return rows, nil
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

func genSelectConfigsFilterIdsStmt(t1, t2 string, lenIds int, filterIdCol string, t1Cols ...string) string {
	return "SELECT * FROM (" + genSelectConfigsStmt(t1, t2, t1Cols...) + fmt.Sprintf(") AS SUB WHERE SUB.%s IN (", filterIdCol) + genQuestionMarks(lenIds) + ");"
}

func createConfigValues(ctx context.Context, tx *sql.Tx, tableName string, config models_handler_storage.Config) error {
	if config.IsSlice {
		colName, values := getListConfigValsAndCol(config)
		stmt := fmt.Sprintf("INSERT INTO %s (c_id, %s, ord) VALUES (?, ?, ?)", tableName, colName)
		for i, value := range values {
			_, err := tx.ExecContext(ctx, stmt, config.Id, value, i)
			if err != nil {
				return err
			}
		}
	} else {
		colName, value := getConfigValAndCol(config)
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
