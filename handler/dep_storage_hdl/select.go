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
	"errors"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"time"
)

func selectDeployment(ctx context.Context, qwf func(context.Context, string, ...any) *sql.Row, depID string) (model.DepBase, error) {
	row := qwf(ctx, "SELECT `id`, `mod_id`, `name`, `dir`, `enabled`, `indirect`, `created`, `updated` FROM `deployments` WHERE `id` = ?", depID)
	var dm model.DepBase
	var ct, ut []uint8
	err := row.Scan(&dm.ID, &dm.ModuleID, &dm.Name, &dm.Dir, &dm.Enabled, &dm.Indirect, &ct, &ut)
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
	dm.Created = tc
	dm.Updated = tu
	return dm, nil
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
