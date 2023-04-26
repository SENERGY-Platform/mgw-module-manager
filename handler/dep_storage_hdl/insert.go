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

package dep_storage_hdl

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"strings"
)

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
