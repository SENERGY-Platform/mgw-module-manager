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

package dep_util

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"strconv"
	"strings"
)

func InsertDepHostRes(ctx context.Context, tx *sql.Tx, dID string, hostResources map[string]string) error {
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO `host_resources` (`dep_id`, `ref`, `res_id`) VALUES (?, ?, ?)")
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	defer stmt.Close()
	for ref, id := range hostResources {
		if _, err = stmt.ExecContext(ctx, dID, ref, id); err != nil {
			return lib_model.NewInternalError(err)
		}
	}
	return nil
}

func InsertDepSecrets(ctx context.Context, tx *sql.Tx, dID string, secrets map[string]lib_model.DepSecret) error {
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO `secrets` (`dep_id`, `ref`, `sec_id`, `item`, `as_mount`, `as_env`) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	defer stmt.Close()
	for ref, secret := range secrets {
		for _, variant := range secret.Variants {
			if _, err = stmt.ExecContext(ctx, dID, ref, secret.ID, variant.Item, variant.AsMount, variant.AsEnv); err != nil {
				return lib_model.NewInternalError(err)
			}
		}
	}
	return nil
}

func InsertDepConfigs(ctx context.Context, tx *sql.Tx, dID string, depConfigs map[string]lib_model.DepConfig) error {
	stmtMap := make(map[string]*sql.Stmt)
	defer func() {
		for _, stmt := range stmtMap {
			stmt.Close()
		}
	}()
	for ref, depConfig := range depConfigs {
		key := depConfig.DataType + strconv.FormatBool(depConfig.IsSlice)
		stmt, ok := stmtMap[key]
		if !ok {
			var err error
			stmt, err = tx.PrepareContext(ctx, genCfgInsertQuery(depConfig.DataType, depConfig.IsSlice))
			if err != nil {
				return lib_model.NewInternalError(err)
			}
			stmtMap[key] = stmt
		}
		if depConfig.IsSlice {
			var err error
			switch depConfig.DataType {
			case module.StringType:
				err = execCfgSlStmt[string](ctx, stmt, dID, ref, depConfig.Value)
			case module.BoolType:
				err = execCfgSlStmt[bool](ctx, stmt, dID, ref, depConfig.Value)
			case module.Int64Type:
				err = execCfgSlStmt[int64](ctx, stmt, dID, ref, depConfig.Value)
			case module.Float64Type:
				err = execCfgSlStmt[float64](ctx, stmt, dID, ref, depConfig.Value)
			default:
				err = fmt.Errorf("unknown data type '%s'", depConfig.Value)
			}
			if err != nil {
				return lib_model.NewInternalError(err)
			}
		} else {
			if _, err := stmt.ExecContext(ctx, dID, ref, depConfig.Value); err != nil {
				return lib_model.NewInternalError(err)
			}
		}
	}
	return nil
}

func genCfgInsertQuery(dataType module.DataType, isSlice bool) string {
	table := "configs"
	cols := []string{"`dep_id`", "`ref`", fmt.Sprintf("`v_%s`", dataType)}
	if isSlice {
		table = "list_" + table
		cols = append(cols, "`ord`")
	}
	return fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s?)", table, strings.Join(cols, ", "), strings.Repeat("?, ", len(cols)-1))
}

func execCfgSlStmt[T any](ctx context.Context, stmt *sql.Stmt, depId string, ref string, val any) error {
	vSl, ok := val.([]T)
	if !ok {
		return fmt.Errorf("invalid data type '%T'", val)
	}
	for i, v := range vSl {
		if _, err := stmt.ExecContext(ctx, depId, ref, v, i); err != nil {
			return err
		}
	}
	return nil
}
