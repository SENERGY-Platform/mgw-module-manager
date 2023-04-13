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

package dep_storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"strconv"
	"strings"
	"time"
)

func insertDeployment(ctx context.Context, ef func(context.Context, string, ...any) (sql.Result, error), qwf func(context.Context, string, ...any) *sql.Row, modId string, depName string, timestamp time.Time) (string, error) {
	res, err := ef(ctx, "INSERT INTO `deployments` (`id`, `module_id`, `name`, `created`, `updated`) VALUES (UUID(), ?, ?, ?, ?)", modId, depName, timestamp, timestamp)
	if err != nil {
		return "", err
	}
	i, err := res.LastInsertId()
	if err != nil {
		return "", err
	}
	row := qwf(ctx, "SELECT `id` FROM `deployments` WHERE `index` = ?", i)
	var id string
	if err = row.Scan(&id); err != nil {
		return "", err
	}
	if id == "" {
		return "", errors.New("generating id failed")
	}
	return id, nil
}

func insertHostResources(ctx context.Context, pf func(context.Context, string) (*sql.Stmt, error), depId string, m map[string]string) error {
	return insertResources(ctx, pf, "INSERT INTO `host_resources` (`dep_id`, `ref`, `res_id`) VALUES (?, ?, ?)", depId, m)
}

func insertSecrets(ctx context.Context, pf func(context.Context, string) (*sql.Stmt, error), depId string, m map[string]string) error {
	return insertResources(ctx, pf, "INSERT INTO `secrets` (`dep_id`, `ref`, `sec_id`) VALUES (?, ?, ?)", depId, m)
}

func insertResources(ctx context.Context, pf func(context.Context, string) (*sql.Stmt, error), query string, depId string, m map[string]string) error {
	stmt, err := pf(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for ref, id := range m {
		if _, err = stmt.ExecContext(ctx, depId, ref, id); err != nil {
			return err
		}
	}
	return nil
}

func insertConfigs(ctx context.Context, pf func(context.Context, string) (*sql.Stmt, error), id string, configs map[string]model.DepConfig) (err error) {
	stmtMap := make(map[string]*sql.Stmt)
	for ref, dC := range configs {
		var stmt *sql.Stmt
		key := dC.DataType + strconv.FormatBool(dC.IsSlice)
		if stmt = stmtMap[key]; stmt == nil {
			stmt, err = pf(ctx, genCfgInsertQuery(dC.DataType, dC.IsSlice))
			if err != nil {
				return
			}
			defer stmt.Close()
			stmtMap[key] = stmt
		}
		if dC.IsSlice {
			switch dC.DataType {
			case module.StringType:
				err = execCfgSlStmt[string](ctx, stmt, id, ref, dC.Value)
			case module.BoolType:
				err = execCfgSlStmt[bool](ctx, stmt, id, ref, dC.Value)
			case module.Int64Type:
				err = execCfgSlStmt[int64](ctx, stmt, id, ref, dC.Value)
			case module.Float64Type:
				err = execCfgSlStmt[float64](ctx, stmt, id, ref, dC.Value)
			default:
				err = fmt.Errorf("unknown data type '%s'", dC.DataType)
			}
			if err != nil {
				return
			}
		} else {
			if _, err = stmt.ExecContext(ctx, id, ref, dC.Value); err != nil {
				return
			}
		}
	}
	return
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

func insertInstance(ctx context.Context, ef func(context.Context, string, ...any) (sql.Result, error), qwf func(context.Context, string, ...any) *sql.Row, depId string, modPath string, timestamp time.Time) (string, error) {
	res, err := ef(ctx, "INSERT INTO `instances` (`id`, `dep_id`, `mod_path`, `created`, `updated`) VALUES (UUID(), ?, ?, ?, ?)", depId, modPath, timestamp, timestamp)
	if err != nil {
		return "", err
	}
	i, err := res.LastInsertId()
	if err != nil {
		return "", err
	}
	row := qwf(ctx, "SELECT `id` FROM `instances` WHERE `index` = ?", i)
	var id string
	if err = row.Scan(&id); err != nil {
		return "", err
	}
	if id == "" {
		return "", errors.New("generating id failed")
	}
	return id, nil
}

func insertContainers(ctx context.Context, pf func(context.Context, string) (*sql.Stmt, error), instId string, m map[string]string) error {
	stmt, err := pf(ctx, "INSERT INTO `containers` (`i_id`, `s_ref`, `c_id`) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for ref, id := range m {
		if _, err = stmt.ExecContext(ctx, instId, ref, id); err != nil {
			return err
		}
	}
	return nil
}
