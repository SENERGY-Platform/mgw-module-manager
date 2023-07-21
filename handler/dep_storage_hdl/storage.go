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
	"bufio"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

const tLayout = "2006-01-02 15:04:05.000000"

type Handler struct {
	db *sql.DB
}

func New(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) Init(ctx context.Context, schemaPath string) error {
	file, err := os.Open(schemaPath)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	var stmts []string
	for {
		stmt, err := reader.ReadString(';')
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		stmts = append(stmts, strings.TrimSuffix(stmt, ";"))
	}
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, stmt := range stmts {
		_, err = tx.ExecContext(ctx, stmt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (h *Handler) BeginTransaction(ctx context.Context) (driver.Tx, error) {
	tx, e := h.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, model.NewInternalError(e)
	}
	return tx, nil
}

func (h *Handler) ListDep(ctx context.Context, filter model.DepFilter) ([]model.DepBase, error) {
	q := "SELECT `id`, `mod_id`, `name`, `dir`, `enabled`, `indirect`, `created`, `updated` FROM `deployments`"
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
	var dms []model.DepBase
	for rows.Next() {
		var dm model.DepBase
		var ct, ut []uint8
		if err = rows.Scan(&dm.ID, &dm.ModuleID, &dm.Name, &dm.Dir, &dm.Enabled, &dm.Indirect, &ct, &ut); err != nil {
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

func (h *Handler) CreateDep(ctx context.Context, itf driver.Tx, depMeta model.DepBase) (string, error) {
	tx := itf.(*sql.Tx)
	res, err := tx.ExecContext(ctx, "INSERT INTO `deployments` (`id`, `mod_id`, `name`, `dir`, `enabled`, `indirect`, `created`, `updated`) VALUES (UUID(), ?, ?, ?, ?, ?, ?, ?)", depMeta.ModuleID, depMeta.Name, depMeta.Dir, depMeta.Enabled, depMeta.Indirect, depMeta.Created, depMeta.Updated)
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

func (h *Handler) ReadDep(ctx context.Context, id string) (model.Deployment, error) {
	depMeta, err := selectDeployment(ctx, h.db.QueryRowContext, id)
	if err != nil {
		return model.Deployment{}, err
	}
	hostRes, err := selectHostResources(ctx, h.db.QueryContext, id)
	if err != nil {
		return model.Deployment{}, model.NewInternalError(err)
	}
	secrets, err := selectSecrets(ctx, h.db.QueryContext, id)
	if err != nil {
		return model.Deployment{}, model.NewInternalError(err)
	}
	configs := make(map[string]model.DepConfig)
	err = selectConfigs(ctx, h.db.QueryContext, id, configs)
	if err != nil {
		return model.Deployment{}, model.NewInternalError(err)
	}
	err = selectListConfigs(ctx, h.db.QueryContext, id, configs)
	if err != nil {
		return model.Deployment{}, model.NewInternalError(err)
	}
	reqDep, err := selectRequiredDep(ctx, h.db.QueryContext, id)
	if err != nil {
		return model.Deployment{}, model.NewInternalError(err)
	}
	depReq, err := selectDepRequiring(ctx, h.db.QueryContext, id)
	if err != nil {
		return model.Deployment{}, model.NewInternalError(err)
	}
	return model.Deployment{
		DepBase: depMeta,
		DepAssets: model.DepAssets{
			HostResources: hostRes,
			Secrets:       secrets,
			Configs:       configs,
			RequiredDep:   reqDep,
			DepRequiring:  depReq,
		},
	}, nil
}

func (h *Handler) UpdateDep(ctx context.Context, depBase model.DepBase) error {
	res, err := h.db.ExecContext(ctx, "UPDATE `deployments` SET `name` = ?, `dir` = ?, `enabled` = ?, `indirect` = ?, `updated` = ? WHERE `id` = ?", depBase.Name, depBase.Dir, depBase.Enabled, depBase.Indirect, depBase.Updated, depBase.ID)
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

func (h *Handler) ListInst(ctx context.Context, filter model.DepInstFilter) ([]model.Instance, error) {
	q := "SELECT `id`, `dep_id`, `created` FROM `instances`"
	fc, val := genListInstFilter(filter)
	if fc != "" {
		q += fc
	}
	rows, err := h.db.QueryContext(ctx, q+" ORDER BY `created`", val...)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	defer rows.Close()
	var dims []model.Instance
	for rows.Next() {
		var dim model.Instance
		var ct []uint8
		if err = rows.Scan(&dim.ID, &dim.DepID, &ct); err != nil {
			return nil, model.NewInternalError(err)
		}
		tc, err := time.Parse(tLayout, string(ct))
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		dim.Created = tc
		dims = append(dims, dim)
	}
	if err = rows.Err(); err != nil {
		return nil, model.NewInternalError(err)
	}
	return dims, nil
}

func (h *Handler) ListInstCtr(ctx context.Context, iID string, filter model.CtrFilter) ([]model.Container, error) {
	q := "SELECT `srv_ref`, `order`, `ctr_id` FROM `inst_containers` WHERE `inst_id` = ? ORDER BY `order` "
	switch filter.SortOrder {
	case model.Ascending:
		q += "ASC"
	case model.Descending:
		q += "DESC"
	default:
		return nil, model.NewInvalidInputError(errors.New("invalid sort direction"))
	}
	rows, err := h.db.QueryContext(ctx, q, iID)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	defer rows.Close()
	var containers []model.Container
	for rows.Next() {
		var ctr model.Container
		if err = rows.Scan(&ctr.Ref, &ctr.Order, &ctr.ID); err != nil {
			return nil, model.NewInternalError(err)
		}
		containers = append(containers, ctr)
	}
	if err = rows.Err(); err != nil {
		return nil, model.NewInternalError(err)
	}
	return containers, nil
}

func (h *Handler) CreateInst(ctx context.Context, itf driver.Tx, dID string, timestamp time.Time) (string, error) {
	tx := itf.(*sql.Tx)
	res, err := tx.ExecContext(ctx, "INSERT INTO `instances` (`id`, `dep_id`, `created`) VALUES (UUID(), ?, ?)", dID, timestamp)
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

func (h *Handler) ReadInst(ctx context.Context, id string) (model.Instance, error) {
	row := h.db.QueryRowContext(ctx, "SELECT `id`, `dep_id`, `created` FROM `instances` WHERE `id` = ?", id)
	var dim model.Instance
	var ct []uint8
	err := row.Scan(&dim.ID, &dim.DepID, &ct)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Instance{}, model.NewNotFoundError(err)
		}
		return model.Instance{}, model.NewInternalError(err)
	}
	tc, err := time.Parse(tLayout, string(ct))
	if err != nil {
		return model.Instance{}, model.NewInternalError(err)
	}
	dim.Created = tc
	return dim, nil
}

func (h *Handler) DeleteInst(ctx context.Context, id string) error {
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

func (h *Handler) CreateInstCtr(ctx context.Context, itf driver.Tx, iID, cID, sRef string, order uint) error {
	tx := itf.(*sql.Tx)
	res, err := tx.ExecContext(ctx, "INSERT INTO `inst_containers` (`inst_id`, `srv_ref`, `order`, `ctr_id`) VALUES (?, ?, ?, ?)", iID, sRef, order, cID)
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
				err = execCfgSlStmt[string](ctx, stmt, dID, ref, depConfig)
			case module.BoolType:
				err = execCfgSlStmt[bool](ctx, stmt, dID, ref, depConfig)
			case module.Int64Type:
				err = execCfgSlStmt[int64](ctx, stmt, dID, ref, depConfig)
			case module.Float64Type:
				err = execCfgSlStmt[float64](ctx, stmt, dID, ref, depConfig)
			default:
				err = fmt.Errorf("unknown data type '%s'", depConfig)
			}
			if err != nil {
				return model.NewInternalError(err)
			}
		} else {
			if _, err := stmt.ExecContext(ctx, dID, ref, depConfig); err != nil {
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
