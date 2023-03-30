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

package deployment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"module-manager/model"
	"strconv"
	"strings"
	"time"
)

type StorageHandler struct {
	db      *sql.DB
	ctx     context.Context
	timeout time.Duration
}

func NewStorageHandler(db *sql.DB, ctx context.Context, timeout time.Duration) *StorageHandler {
	return &StorageHandler{
		db:      db,
		ctx:     ctx,
		timeout: timeout,
	}
}

func (h *StorageHandler) List() ([]model.DepMeta, error) {
	panic("not implemented")
}

func (h *StorageHandler) Create(dep *model.Deployment) (string, error) {
	ctx, cf := context.WithTimeout(h.ctx, h.timeout)
	defer cf()
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, "INSERT INTO `deployments` (`id`, `module_id`, `name`, `created`, `updated`) VALUES (UUID(), ?, ?, NOW(), NOW())", dep.ModuleID, dep.Name)
	if err != nil {
		return "", err
	}
	i, err := res.LastInsertId()
	if err != nil {
		return "", err
	}
	row := tx.QueryRowContext(ctx, "SELECT `id` FROM `deployments` WHERE `index` = ?", i)
	var id string
	if err = row.Scan(&id); err != nil {
		return "", err
	}
	if id == "" {
		return "", errors.New("generating id failed")
	}
	if len(dep.HostResources) > 0 {
		stmt, err := tx.PrepareContext(ctx, "INSERT INTO `host_resources` (`dep_id`, `ref`, `res_id`) VALUES (?, ?, ?)")
		if err != nil {
			return "", err
		}
		defer stmt.Close()
		if err := execResStmt(ctx, stmt, id, dep.HostResources); err != nil {
			return "", err
		}
	}
	if len(dep.Secrets) > 0 {
		stmt, err := tx.PrepareContext(ctx, "INSERT INTO `secrets` (`dep_id`, `ref`, `sec_id`) VALUES (?, ?, ?)")
		if err != nil {
			return "", err
		}
		defer stmt.Close()
		if err := execResStmt(ctx, stmt, id, dep.Secrets); err != nil {
			return "", err
		}
	}
	if len(dep.Configs) > 0 {
		stmtMap := make(map[string]*sql.Stmt)
		for ref, dC := range dep.Configs {
			var stmt *sql.Stmt
			key := dC.DataType + strconv.FormatBool(dC.IsSlice)
			if stmt = stmtMap[key]; stmt == nil {
				var err error
				table := []string{"configs", dC.DataType}
				cols := []string{"`dep_id`", "`ref`", "`value`"}
				if dC.IsSlice {
					table = append(table, "list")
					cols = append(cols, "`ord`")
				}
				stmt, err = tx.PrepareContext(ctx, fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s?)", strings.Join(table, "_"), strings.Join(cols, ", "), strings.Repeat("?, ", len(cols)-1)))
				if err != nil {
					return "", err
				}
				defer stmt.Close()
				stmtMap[key] = stmt
			}
			if dC.IsSlice {
				var er error
				switch dC.DataType {
				case module.StringType:
					er = execCfgSlStmt[string](ctx, stmt, id, ref, dC.Value)
				case module.BoolType:
					er = execCfgSlStmt[bool](ctx, stmt, id, ref, dC.Value)
				case module.Int64Type:
					er = execCfgSlStmt[int64](ctx, stmt, id, ref, dC.Value)
				case module.Float64Type:
					er = execCfgSlStmt[float64](ctx, stmt, id, ref, dC.Value)
				default:
					er = fmt.Errorf("unknown data type '%s'", dC.DataType)
				}
				if er != nil {
					return "", er
				}
			} else {
				if err := execStmt(ctx, stmt, id, ref, dC.Value); err != nil {
					return "", err
				}
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return "", err
	}
	return id, nil
}

func (h *StorageHandler) Read(id string) (*model.Deployment, error) {
	panic("not implemented")
}

func (h *StorageHandler) Update(dep *model.Deployment) error {
	panic("not implemented")
}

func (h *StorageHandler) Delete(id string) error {
	panic("not implemented")
}

func execResStmt(ctx context.Context, stmt *sql.Stmt, id string, rMap map[string]string) error {
	for ref, rId := range rMap {
		if err := execStmt(ctx, stmt, id, ref, rId); err != nil {
			return err
		}
	}
	return nil
}

func execCfgSlStmt[T any](ctx context.Context, stmt *sql.Stmt, id string, ref string, val any) error {
	vSl, ok := val.([]T)
	if !ok {
		return fmt.Errorf("invalid data type '%T'", val)
	}
	for i, v := range vSl {
		if err := execStmt(ctx, stmt, id, ref, v, i); err != nil {
			return err
		}
	}
	return nil
}

func execStmt(ctx context.Context, stmt *sql.Stmt, args ...any) error {
	res, err := stmt.ExecContext(ctx, args...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("")
	}
	return nil
}
