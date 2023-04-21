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
	"database/sql/driver"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"strconv"
	"strings"
	"time"
)

const tLayout = "2006-01-02 15:04:05.000000"

type StorageHandler struct {
	db *sql.DB
}

func NewStorageHandler(db *sql.DB) *StorageHandler {
	return &StorageHandler{db: db}
}

func (h *StorageHandler) BeginTransaction(ctx context.Context) (driver.Tx, error) {
	tx, e := h.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, model.NewInternalError(e)
	}
	return tx, nil
}

func genListDepFilter(filter model.DepFilter) (string, []any) {
	var fc []string
	var val []any
	if filter.Name != "" {
		fc = append(fc, "`name` = ?")
		val = append(val, filter.Name)
	}
	if filter.ModuleID != "" {
		fc = append(fc, "`module_id` = ?")
		val = append(val, filter.ModuleID)
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}

func listDep(ctx context.Context, qf func(ctx context.Context, query string, args ...any) (*sql.Rows, error), query string, args ...any) ([]model.DepMeta, error) {
	rows, err := qf(ctx, query, args...)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	defer rows.Close()
	var dms []model.DepMeta
	for rows.Next() {
		var dm model.DepMeta
		var ct, ut []uint8
		if err = rows.Scan(&dm.ID, &dm.ModuleID, &dm.Name, &dm.Indirect, &ct, &ut); err != nil {
			return nil, model.NewInternalError(err)
		}
		tc, err := time.Parse(tLayout, string(ct))
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		tu, err := time.Parse(tLayout, string(ut))
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		dm.Created = tc
		dm.Updated = tu
		dms = append(dms, dm)
	}
	if err = rows.Err(); err != nil {
		return nil, model.NewInternalError(err)
	}
	return dms, nil
}

func (h *StorageHandler) ListDep(ctx context.Context, filter model.DepFilter) ([]model.DepMeta, error) {
	q := "SELECT `id`, `module_id`, `name`, `indirect`, `created`, `updated` FROM `deployments`"
	fc, val := genListDepFilter(filter)
	if fc != "" {
		q += fc
	}
	q += " ORDER BY `name`"
	return listDep(ctx, h.db.QueryContext, q, val...)
}

func (h *StorageHandler) ListRequiredDep(ctx context.Context, dID string) ([]model.DepMeta, error) {
	return listDep(ctx, h.db.QueryContext, "SELECT `id`, `module_id`, `name`, `indirect`, `created`, `updated` FROM `deployments` WHERE `id` IN (SELECT `req_id` FROM `dependencies` WHERE `dep_id` = ?)", dID)
}

func (h *StorageHandler) ListDepRequiring(ctx context.Context, dID string) ([]model.DepMeta, error) {
	return listDep(ctx, h.db.QueryContext, "SELECT `id`, `module_id`, `name`, `indirect`, `created`, `updated` FROM `deployments` WHERE `id` IN (SELECT `dep_id` FROM `dependencies` WHERE `req_id` = ?)", dID)
}

func (h *StorageHandler) CreateDep(ctx context.Context, itf driver.Tx, mID, name string, indirect bool, timestamp time.Time) (string, error) {
	tx := itf.(*sql.Tx)
	res, err := tx.ExecContext(ctx, "INSERT INTO `deployments` (`id`, `module_id`, `name`, `indirect`, `created`, `updated`) VALUES (UUID(), ?, ?, ?, ?, ?)", mID, name, indirect, timestamp, timestamp)
	if err != nil {
		return "", model.NewInternalError(err)
	}
	i, err := res.LastInsertId()
	if err != nil {
		return "", model.NewInternalError(err)
	}
	row := tx.QueryRowContext(ctx, "SELECT `id` FROM `deployments` WHERE `index` = ?", i)
	var id string
	if err = row.Scan(&id); err != nil {
		return "", model.NewInternalError(err)
	}
	if id == "" {
		return "", model.NewInternalError(errors.New("generating id failed"))
	}
	return id, nil
}

func (h *StorageHandler) CreateDepConfigs(ctx context.Context, itf driver.Tx, mConfigs module.Configs, dConfigs map[string]any, dID string) error {
	tx := itf.(*sql.Tx)
	stmtMap := make(map[string]*sql.Stmt)
	defer func() {
		for _, stmt := range stmtMap {
			stmt.Close()
		}
	}()
	for ref, val := range dConfigs {
		mConfig, ok := mConfigs[ref]
		if !ok {
			return model.NewInternalError(fmt.Errorf("config '%s' not defined", ref))
		}
		var stmt *sql.Stmt
		key := mConfig.DataType + strconv.FormatBool(mConfig.IsSlice)
		if stmt = stmtMap[key]; stmt == nil {
			stmt, err := tx.PrepareContext(ctx, genCfgInsertQuery(mConfig.DataType, mConfig.IsSlice))
			if err != nil {
				return model.NewInternalError(err)
			}
			stmtMap[key] = stmt
		}
		if mConfig.IsSlice {
			var err error
			switch mConfig.DataType {
			case module.StringType:
				err = execCfgSlStmt[string](ctx, stmt, dID, ref, val)
			case module.BoolType:
				err = execCfgSlStmt[bool](ctx, stmt, dID, ref, val)
			case module.Int64Type:
				err = execCfgSlStmt[int64](ctx, stmt, dID, ref, val)
			case module.Float64Type:
				err = execCfgSlStmt[float64](ctx, stmt, dID, ref, val)
			default:
				err = fmt.Errorf("unknown data type '%s'", val)
			}
			if err != nil {
				return model.NewInternalError(err)
			}
		} else {
			if _, err := stmt.ExecContext(ctx, dID, ref, val); err != nil {
				return model.NewInternalError(err)
			}
		}
	}
	return nil
}

func (h *StorageHandler) CreateDepHostRes(ctx context.Context, itf driver.Tx, hostResources map[string]string, dID string) error {
	tx := itf.(*sql.Tx)
	err := insertResources(ctx, tx.PrepareContext, "INSERT INTO `host_resources` (`dep_id`, `ref`, `res_id`) VALUES (?, ?, ?)", dID, hostResources)
	if err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func (h *StorageHandler) CreateDepSecrets(ctx context.Context, itf driver.Tx, secrets map[string]string, dID string) error {
	tx := itf.(*sql.Tx)
	err := insertResources(ctx, tx.PrepareContext, "INSERT INTO `secrets` (`dep_id`, `ref`, `sec_id`) VALUES (?, ?, ?)", dID, secrets)
	if err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func (h *StorageHandler) CreateDepReq(ctx context.Context, itf driver.Tx, depReq []string, dID string) error {
	tx := itf.(*sql.Tx)
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO `dependencies` (`dep_id`, `req_id`) VALUES (?, ?)")
	if err != nil {
		return model.NewInternalError(err)
	}
	defer stmt.Close()
	for _, id := range depReq {
		if _, err = stmt.ExecContext(ctx, dID, id); err != nil {
			return model.NewInternalError(err)
		}
	}
	return nil
}

func (h *StorageHandler) ReadDep(ctx context.Context, id string) (*model.Deployment, error) {
	depMeta, err := selectDeployment(ctx, h.db.QueryRowContext, id)
	if err != nil {
		return nil, err
	}
	depMeta.ID = id
	hostRes, err := selectHostResources(ctx, h.db.QueryContext, id)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	secrets, err := selectSecrets(ctx, h.db.QueryContext, id)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	configs := make(map[string]model.DepConfig)
	err = selectConfigs(ctx, h.db.QueryContext, id, configs)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	err = selectListConfigs(ctx, h.db.QueryContext, id, configs)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	dep := model.Deployment{
		DepMeta:       depMeta,
		HostResources: hostRes,
		Secrets:       secrets,
		Configs:       configs,
	}
	return &dep, nil
}

func (h *StorageHandler) UpdateDep(ctx context.Context, itf driver.Tx, dID, name string, indirect bool, timestamp time.Time) error {
	tx := itf.(*sql.Tx)
	res, err := tx.ExecContext(ctx, "UPDATE `deployments` SET `name` = ?, `indirect` = ?, `updated` = ? WHERE `id` = ?", name, indirect, timestamp, dID)
	if err != nil {
		return model.NewInternalError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return model.NewInternalError(err)
	}
	if n < 1 {
		return model.NewNotFoundError(errors.New("no rows affected"))
	}
	return nil
}

func (h *StorageHandler) DeleteDep(ctx context.Context, id string) error {
	res, err := h.db.ExecContext(ctx, "DELETE FROM `deployments` WHERE `id` = ?", id)
	if err != nil {
		return model.NewInternalError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return model.NewInternalError(err)
	}
	if n < 1 {
		return model.NewNotFoundError(errors.New("no rows affected"))
	}
	return nil
}

func (h *StorageHandler) DeleteDepConfigs(ctx context.Context, itf driver.Tx, dID string) error {
	tx := itf.(*sql.Tx)
	_, err := tx.ExecContext(ctx, "DELETE FROM `configs` WHERE `dep_id` = ?", dID)
	if err != nil {
		return model.NewInternalError(err)
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM `list_configs` WHERE `dep_id` = ?", dID)
	if err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func (h *StorageHandler) DeleteDepHostRes(ctx context.Context, itf driver.Tx, dID string) error {
	tx := itf.(*sql.Tx)
	_, err := tx.ExecContext(ctx, "DELETE FROM `host_resources` WHERE `dep_id` = ?", dID)
	if err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func (h *StorageHandler) DeleteDepSecrets(ctx context.Context, itf driver.Tx, dID string) error {
	tx := itf.(*sql.Tx)
	_, err := tx.ExecContext(ctx, "DELETE FROM `secrets` WHERE `dep_id` = ?", dID)
	if err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func genListInstFilter(filter model.DepInstFilter) (string, []any) {
	var fc []string
	var val []any
	if filter.DepID != "" {
		fc = append(fc, "`dep_id` = ?")
		val = append(val, filter.DepID)
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}

func (h *StorageHandler) ListInst(ctx context.Context, filter model.DepInstFilter) ([]model.DepInstanceMeta, error) {
	q := "SELECT `id`, `dep_id`, `created`, `updated` FROM `instances`"
	fc, val := genListInstFilter(filter)
	if fc != "" {
		q += fc
	}
	rows, err := h.db.QueryContext(ctx, q+" ORDER BY `created`", val...)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	defer rows.Close()
	var dims []model.DepInstanceMeta
	for rows.Next() {
		var dim model.DepInstanceMeta
		var ct, ut []uint8
		if err = rows.Scan(&dim.ID, &dim.DepID, &ct, &ut); err != nil {
			return nil, model.NewInternalError(err)
		}
		tc, err := time.Parse(tLayout, string(ct))
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		tu, err := time.Parse(tLayout, string(ut))
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		dim.Created = tc
		dim.Updated = tu
		dims = append(dims, dim)
	}
	if err = rows.Err(); err != nil {
		return nil, model.NewInternalError(err)
	}
	return dims, nil
}

func (h *StorageHandler) CreateInst(ctx context.Context, itf driver.Tx, dID string, timestamp time.Time) (string, error) {
	tx := itf.(*sql.Tx)
	res, err := tx.ExecContext(ctx, "INSERT INTO `instances` (`id`, `dep_id`, `created`, `updated`) VALUES (UUID(), ?, ?, ?)", dID, timestamp, timestamp)
	if err != nil {
		return "", model.NewInternalError(err)
	}
	i, err := res.LastInsertId()
	if err != nil {
		return "", model.NewInternalError(err)
	}
	row := tx.QueryRowContext(ctx, "SELECT `id` FROM `instances` WHERE `index` = ?", i)
	var id string
	if err = row.Scan(&id); err != nil {
		return "", model.NewInternalError(err)
	}
	if id == "" {
		return "", model.NewInternalError(errors.New("generating id failed"))
	}
	return id, nil
}

func (h *StorageHandler) ReadInst(ctx context.Context, id string) (*model.DepInstance, error) {
	instMeta, err := selectInstance(ctx, h.db.QueryRowContext, id)
	if err != nil {
		return nil, err
	}
	instMeta.ID = id
	containers, err := selectContainers(ctx, h.db.QueryContext, id)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	inst := model.DepInstance{
		DepInstanceMeta: instMeta,
		Containers:      containers,
	}
	return &inst, nil
}

func (h *StorageHandler) UpdateInst(ctx context.Context, itf driver.Tx, id string, timestamp time.Time) error {
	tx := itf.(*sql.Tx)
	res, err := tx.ExecContext(ctx, "UPDATE `instances` SET `updated` = ? WHERE `id` = ?", timestamp, id)
	if err != nil {
		return model.NewInternalError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return model.NewInternalError(err)
	}
	if n < 1 {
		return model.NewNotFoundError(errors.New("no rows affected"))
	}
	return nil
}

func (h *StorageHandler) DeleteInst(ctx context.Context, id string) error {
	res, err := h.db.ExecContext(ctx, "DELETE FROM `instances` WHERE `id` = ?", id)
	if err != nil {
		return model.NewInternalError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return model.NewInternalError(err)
	}
	if n < 1 {
		return model.NewNotFoundError(errors.New("no rows affected"))
	}
	return nil
}

func (h *StorageHandler) CreateInstCtr(ctx context.Context, itf driver.Tx, iID, cID, sRef string) error {
	tx := itf.(*sql.Tx)
	res, err := tx.ExecContext(ctx, "INSERT INTO `containers` (`i_id`, `s_ref`, `c_id`) VALUES (?, ?, ?)", iID, sRef, cID)
	if err != nil {
		return model.NewInternalError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return model.NewInternalError(err)
	}
	if n < 1 {
		return model.NewNotFoundError(errors.New("no rows affected"))
	}
	return nil
}

func (h *StorageHandler) DeleteInstCtr(ctx context.Context, cID string) error {
	panic("not implemented")
}
