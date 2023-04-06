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
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"time"
)

func selectDeployment(ctx context.Context, qwf func(context.Context, string, ...any) *sql.Row, depID string) (model.DepMeta, error) {
	row := qwf(ctx, "SELECT `module_id`, `name`, `created`, `updated` FROM `deployments` WHERE `id` = ?", depID)
	var dm model.DepMeta
	var ct, ut []uint8
	err := row.Scan(&dm.ModuleID, &dm.Name, &ct, &ut)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.DepMeta{}, model.NewNotFoundError(err)
		}
		return model.DepMeta{}, model.NewInternalError(err)
	}
	tc, err := time.Parse(tLayout, string(ct))
	if err != nil {
		return model.DepMeta{}, model.NewInternalError(err)
	}
	tu, err := time.Parse(tLayout, string(ut))
	if err != nil {
		return model.DepMeta{}, model.NewInternalError(err)
	}
	dm.Created = tc
	dm.Updated = tu
	return dm, nil
}

func selectHostResources(ctx context.Context, qf func(ctx context.Context, query string, args ...any) (*sql.Rows, error), depID string) (map[string]string, error) {
	return selectResources(ctx, qf, "SELECT `ref`, `res_id` FROM `host_resources` WHERE `dep_id` = ?", depID)
}

func selectSecrets(ctx context.Context, qf func(ctx context.Context, query string, args ...any) (*sql.Rows, error), depID string) (map[string]string, error) {
	return selectResources(ctx, qf, "SELECT `ref`, `sec_id` FROM `secrets` WHERE `dep_id` = ?", depID)
}

func selectResources(ctx context.Context, qf func(ctx context.Context, query string, args ...any) (*sql.Rows, error), query string, depID string) (map[string]string, error) {
	rows, err := qf(ctx, query, depID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]string)
	for rows.Next() {
		var ref, id string
		if err = rows.Scan(&ref, &id); err != nil {
			return nil, err
		}
		m[ref] = id
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

func selectInstance(ctx context.Context, qwf func(context.Context, string, ...any) *sql.Row, instID string) (model.DepInstanceMeta, error) {
	row := qwf(ctx, "SELECT `dep_id`, `mod_path`, `created`, `updated` FROM `instances` WHERE `id` = ?", instID)
	var dim model.DepInstanceMeta
	var ct, ut []uint8
	err := row.Scan(&dim.DepID, &dim.ModPath, &ct, &ut)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.DepInstanceMeta{}, model.NewNotFoundError(err)
		}
		return model.DepInstanceMeta{}, model.NewInternalError(err)
	}
	tc, err := time.Parse(tLayout, string(ct))
	if err != nil {
		return model.DepInstanceMeta{}, model.NewInternalError(err)
	}
	tu, err := time.Parse(tLayout, string(ut))
	if err != nil {
		return model.DepInstanceMeta{}, model.NewInternalError(err)
	}
	dim.Created = tc
	dim.Updated = tu
	return dim, nil
}

func selectContainers(ctx context.Context, qf func(ctx context.Context, query string, args ...any) (*sql.Rows, error), instID string) (map[string]string, error) {
	rows, err := qf(ctx, "SELECT `s_ref`, `c_id` FROM `secrets` WHERE `i_id` = ?", instID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]string)
	for rows.Next() {
		var sRef, cId string
		if err = rows.Scan(&sRef, &cId); err != nil {
			return nil, err
		}
		m[sRef] = cId
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return m, nil
}
