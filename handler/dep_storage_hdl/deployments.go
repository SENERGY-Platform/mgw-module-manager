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
	"database/sql/driver"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) ListDep(ctx context.Context, filter model.DepFilter) ([]model.DepBase, error) {
	q := "SELECT `id`, `mod_id`, `mod_ver`, `name`, `dir`, `enabled`, `indirect`, `created`, `updated` FROM `deployments`"
	fc, val := genListDepFilter(filter)
	if fc != "" {
		q += fc
	}
	q += " ORDER BY `name`"
	rows, err := h.db.QueryContext(ctx, q, val...)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	defer rows.Close()
	var deployments []model.DepBase
	for rows.Next() {
		var depBase model.DepBase
		var depModule model.DepModule
		var ct, ut []uint8
		if err = rows.Scan(&depBase.ID, &depModule.ID, &depModule.Version, &depBase.Name, &depBase.Dir, &depBase.Autostart, &depBase.Started, &depBase.Indirect, &ct, &ut); err != nil {
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
		depBase.Module = depModule
		depBase.Created = tc
		depBase.Updated = tu
		deployments = append(deployments, depBase)
	}
	if err = rows.Err(); err != nil {
		return nil, model.NewInternalError(err)
	}
	return deployments, nil
}

func (h *Handler) CreateDep(ctx context.Context, itf driver.Tx, depBase model.DepBase) (string, error) {
	tx := itf.(*sql.Tx)
	res, err := tx.ExecContext(ctx, "INSERT INTO `deployments` (`id`, `mod_id`, `mod_ver`, `name`, `dir`, `enabled`, `indirect`, `created`, `updated`) VALUES (UUID(), ?, ?, ?, ?, ?, ?, ?, ?)", depBase.Module.ID, depBase.Module.Version, depBase.Name, depBase.Dir, depBase.Enabled, depBase.Indirect, depBase.Created, depBase.Updated)
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

func (h *Handler) CreateDepAssets(ctx context.Context, itf driver.Tx, dID string, depAssets model.DepAssets) error {
	tx := itf.(*sql.Tx)
	if err := h.createDepHostRes(ctx, tx, dID, depAssets.HostResources); err != nil {
		return err
	}
	if err := h.createDepSecrets(ctx, tx, dID, depAssets.Secrets); err != nil {
		return err
	}
	if err := h.createDepConfigs(ctx, tx, dID, depAssets.Configs); err != nil {
		return err
	}
	if err := h.createDepReq(ctx, tx, dID, depAssets.RequiredDep); err != nil {
		return err
	}
	return nil
}

func (h *Handler) DeleteDepAssets(ctx context.Context, itf driver.Tx, dID string) error {
	tx := itf.(*sql.Tx)
	if err := h.deleteDepHostRes(ctx, tx, dID); err != nil {
		return err
	}
	if err := h.deleteDepSecrets(ctx, tx, dID); err != nil {
		return err
	}
	if err := h.deleteDepConfigs(ctx, tx, dID); err != nil {
		return err
	}
	if err := h.deleteDepReq(ctx, tx, dID); err != nil {
		return err
	}
	return nil
}

func (h *Handler) ReadDep(ctx context.Context, id string, assets bool) (model.Deployment, error) {
	var dep model.Deployment
	var err error
	dep.DepBase, err = selectDeployment(ctx, h.db.QueryRowContext, id)
	if err != nil {
		return model.Deployment{}, err
	}
	if assets {
		dep.DepAssets, err = h.readDepAssets(ctx, id)
		if err != nil {
			return model.Deployment{}, err
		}
	}
	return dep, nil
}

func (h *Handler) readDepAssets(ctx context.Context, id string) (model.DepAssets, error) {
	hostRes, err := selectHostResources(ctx, h.db.QueryContext, id)
	if err != nil {
		return model.DepAssets{}, model.NewInternalError(err)
	}
	secrets, err := selectSecrets(ctx, h.db.QueryContext, id)
	if err != nil {
		return model.DepAssets{}, model.NewInternalError(err)
	}
	configs := make(map[string]model.DepConfig)
	err = selectConfigs(ctx, h.db.QueryContext, id, configs)
	if err != nil {
		return model.DepAssets{}, model.NewInternalError(err)
	}
	err = selectListConfigs(ctx, h.db.QueryContext, id, configs)
	if err != nil {
		return model.DepAssets{}, model.NewInternalError(err)
	}
	reqDep, err := selectRequiredDep(ctx, h.db.QueryContext, id)
	if err != nil {
		return model.DepAssets{}, model.NewInternalError(err)
	}
	depReq, err := selectDepRequiring(ctx, h.db.QueryContext, id)
	if err != nil {
		return model.DepAssets{}, model.NewInternalError(err)
	}
	return model.DepAssets{
		HostResources: hostRes,
		Secrets:       secrets,
		Configs:       configs,
		RequiredDep:   reqDep,
		DepRequiring:  depReq,
	}, nil
}

func (h *Handler) UpdateDep(ctx context.Context, itf driver.Tx, depBase model.DepBase) error {
	execContext := h.db.ExecContext
	if itf != nil {
		tx := itf.(*sql.Tx)
		execContext = tx.ExecContext
	}
	res, err := execContext(ctx, "UPDATE `deployments` SET `mod_ver` = ?, `name` = ?, `dir` = ?, `enabled` = ?, `indirect` = ?, `updated` = ? WHERE `id` = ?", depBase.Module.Version, depBase.Name, depBase.Dir, depBase.Enabled, depBase.Indirect, depBase.Updated, depBase.ID)
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

func (h *Handler) DeleteDep(ctx context.Context, id string) error {
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

func (h *Handler) createDepConfigs(ctx context.Context, tx *sql.Tx, dID string, depConfigs map[string]model.DepConfig) error {
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
				return model.NewInternalError(err)
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
				return model.NewInternalError(err)
			}
		} else {
			if _, err := stmt.ExecContext(ctx, dID, ref, depConfig.Value); err != nil {
				return model.NewInternalError(err)
			}
		}
	}
	return nil
}

func (h *Handler) createDepHostRes(ctx context.Context, tx *sql.Tx, dID string, hostResources map[string]string) error {
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO `host_resources` (`dep_id`, `ref`, `res_id`) VALUES (?, ?, ?)")
	if err != nil {
		return model.NewInternalError(err)
	}
	defer stmt.Close()
	for ref, id := range hostResources {
		if _, err = stmt.ExecContext(ctx, dID, ref, id); err != nil {
			return model.NewInternalError(err)
		}
	}
	return nil
}

func (h *Handler) createDepSecrets(ctx context.Context, tx *sql.Tx, dID string, secrets map[string]model.DepSecret) error {
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO `secrets` (`dep_id`, `ref`, `sec_id`, `item`, `as_mount`, `as_env`) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return model.NewInternalError(err)
	}
	defer stmt.Close()
	for ref, secret := range secrets {
		for _, variant := range secret.Variants {
			if _, err = stmt.ExecContext(ctx, dID, ref, secret.ID, variant.Item, variant.AsMount, variant.AsEnv); err != nil {
				return model.NewInternalError(err)
			}
		}
	}
	return nil
}

func (h *Handler) createDepReq(ctx context.Context, tx *sql.Tx, dID string, depReq []string) error {
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

func (h *Handler) deleteDepReq(ctx context.Context, tx *sql.Tx, dID string) error {
	_, err := tx.ExecContext(ctx, "DELETE FROM `dependencies` WHERE `dep_id` = ?", dID)
	if err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) deleteDepConfigs(ctx context.Context, tx *sql.Tx, dID string) error {
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

func (h *Handler) deleteDepHostRes(ctx context.Context, tx *sql.Tx, dID string) error {
	_, err := tx.ExecContext(ctx, "DELETE FROM `host_resources` WHERE `dep_id` = ?", dID)
	if err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) deleteDepSecrets(ctx context.Context, tx *sql.Tx, dID string) error {
	_, err := tx.ExecContext(ctx, "DELETE FROM `secrets` WHERE `dep_id` = ?", dID)
	if err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func selectDeployment(ctx context.Context, qwf func(context.Context, string, ...any) *sql.Row, depID string) (model.DepBase, error) {
	row := qwf(ctx, "SELECT `id`, `mod_id`, `mod_ver`, `name`, `dir`, `enabled`, `indirect`, `created`, `updated` FROM `deployments` WHERE `id` = ?", depID)
	var depBase model.DepBase
	var depModule model.DepModule
	var ct, ut []uint8
	err := row.Scan(&depBase.ID, &depModule.ID, &depModule.Version, &depBase.Name, &depBase.Dir, &depBase.Autostart, &depBase.Started, &depBase.Indirect, &ct, &ut)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.DepBase{}, model.NewNotFoundError(err)
		}
		return model.DepBase{}, model.NewInternalError(err)
	}
	tc, err := time.Parse(tLayout, string(ct))
	if err != nil {
		return model.DepBase{}, model.NewInternalError(err)
	}
	tu, err := time.Parse(tLayout, string(ut))
	if err != nil {
		return model.DepBase{}, model.NewInternalError(err)
	}
	depBase.Module = depModule
	depBase.Created = tc
	depBase.Updated = tu
	return depBase, nil
}

func selectHostResources(ctx context.Context, qf func(ctx context.Context, query string, args ...any) (*sql.Rows, error), depID string) (map[string]string, error) {
	rows, err := qf(ctx, "SELECT `ref`, `res_id` FROM `host_resources` WHERE `dep_id` = ?", depID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]string)
	for rows.Next() {
		var ref, rID string
		if err = rows.Scan(&ref, &rID); err != nil {
			return nil, err
		}
		m[ref] = rID
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return m, nil
}

func selectSecrets(ctx context.Context, qf func(ctx context.Context, query string, args ...any) (*sql.Rows, error), depID string) (map[string]model.DepSecret, error) {
	rows, err := qf(ctx, "SELECT `ref`, `sec_id`, `item`, `as_mount`, `as_env` FROM `secrets` WHERE `dep_id` = ?", depID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]model.DepSecret)
	for rows.Next() {
		var ref, sID string
		var item *string
		var asMount, asEnv bool
		if err = rows.Scan(&ref, &sID, &item, &asMount, &asEnv); err != nil {
			return nil, err
		}
		ds, ok := m[ref]
		if !ok {
			ds.ID = sID
		}
		ds.Variants = append(ds.Variants, model.DepSecretVariant{
			Item:    item,
			AsMount: asMount,
			AsEnv:   asEnv,
		})
		m[ref] = ds
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return m, nil
}

func selectConfigs(ctx context.Context, qf func(ctx context.Context, query string, args ...any) (*sql.Rows, error), depID string, m map[string]model.DepConfig) error {
	cfgRows, err := qf(ctx, "SELECT `ref`, `v_string`, `v_int`, `v_float`, `v_bool` FROM `configs` WHERE `dep_id` = ?", depID)
	if err != nil {
		return err
	}
	defer cfgRows.Close()
	for cfgRows.Next() {
		var ref string
		var vString sql.NullString
		var vInt sql.NullInt64
		var vFloat sql.NullFloat64
		var vBool sql.NullBool
		if err = cfgRows.Scan(&ref, &vString, &vInt, &vFloat, &vBool); err != nil {
			return err
		}
		dc := model.DepConfig{}
		if vString.Valid {
			dc.Value = vString.String
			dc.DataType = module.StringType
		} else if vInt.Valid {
			dc.Value = vInt.Int64
			dc.DataType = module.Int64Type
		} else if vFloat.Valid {
			dc.Value = vFloat.Float64
			dc.DataType = module.Float64Type
		} else if vBool.Valid {
			dc.Value = vBool.Bool
			dc.DataType = module.BoolType
		}
		m[ref] = dc
	}
	if err = cfgRows.Err(); err != nil {
		return err
	}
	return nil
}

func selectListConfigs(ctx context.Context, qf func(ctx context.Context, query string, args ...any) (*sql.Rows, error), depID string, m map[string]model.DepConfig) error {
	lstCfgRows, err := qf(ctx, "SELECT `ref`, `ord`, `v_string`, `v_int`, `v_float`, `v_bool` FROM `list_configs` WHERE `dep_id` = ? ORDER BY `ref`, `ord`", depID)
	if err != nil {
		return err
	}
	defer lstCfgRows.Close()
	for lstCfgRows.Next() {
		var ref string
		var ord int
		var vString sql.NullString
		var vInt sql.NullInt64
		var vFloat sql.NullFloat64
		var vBool sql.NullBool
		if err = lstCfgRows.Scan(&ref, &ord, &vString, &vInt, &vFloat, &vBool); err != nil {
			return err
		}
		dc, ok := m[ref]
		if !ok {
			dc = model.DepConfig{IsSlice: true}
			if vString.Valid {
				dc.Value = []string{}
				dc.DataType = module.StringType
			} else if vInt.Valid {
				dc.Value = []int64{}
				dc.DataType = module.Int64Type
			} else if vFloat.Valid {
				dc.Value = []float64{}
				dc.DataType = module.Float64Type
			} else if vBool.Valid {
				dc.Value = []bool{}
				dc.DataType = module.BoolType
			}
		}
		switch dc.DataType {
		case module.StringType:
			dc.Value = append(dc.Value.([]string), vString.String)
		case module.Int64Type:
			dc.Value = append(dc.Value.([]int64), vInt.Int64)
		case module.Float64Type:
			dc.Value = append(dc.Value.([]float64), vFloat.Float64)
		case module.BoolType:
			dc.Value = append(dc.Value.([]bool), vBool.Bool)
		}
		m[ref] = dc
	}
	if err = lstCfgRows.Err(); err != nil {
		return err
	}
	return nil
}

func selectRequiredDep(ctx context.Context, qf func(ctx context.Context, query string, args ...any) (*sql.Rows, error), dID string) ([]string, error) {
	return selectReq(ctx, qf, "SELECT `req_id` FROM `dependencies` WHERE `dep_id` = ?", dID)
}

func selectDepRequiring(ctx context.Context, qf func(ctx context.Context, query string, args ...any) (*sql.Rows, error), dID string) ([]string, error) {
	return selectReq(ctx, qf, "SELECT `dep_id` FROM `dependencies` WHERE `req_id` = ?", dID)
}

func selectReq(ctx context.Context, qf func(ctx context.Context, query string, args ...any) (*sql.Rows, error), query, dID string) ([]string, error) {
	rows, err := qf(ctx, query, dID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var IDs []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		IDs = append(IDs, id)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return IDs, nil
}

func genListDepFilter(filter model.DepFilter) (string, []any) {
	var fc []string
	var val []any
	if filter.Name != "" {
		fc = append(fc, "`name` = ?")
		val = append(val, filter.Name)
	}
	if filter.ModuleID != "" {
		fc = append(fc, "`mod_id` = ?")
		val = append(val, filter.ModuleID)
	}
	if filter.Indirect {
		fc = append(fc, "`indirect` = ?")
		val = append(val, filter.Indirect)
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
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
