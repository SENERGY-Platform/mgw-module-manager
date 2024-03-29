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
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

func SelectHostResources(ctx context.Context, db *sql.DB, dID string) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, "SELECT `ref`, `res_id` FROM `host_resources` WHERE `dep_id` = ?", dID)
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

func SelectSecrets(ctx context.Context, db *sql.DB, dID string) (map[string]lib_model.DepSecret, error) {
	rows, err := db.QueryContext(ctx, "SELECT `ref`, `sec_id`, `item`, `as_mount`, `as_env` FROM `secrets` WHERE `dep_id` = ?", dID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]lib_model.DepSecret)
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
		ds.Variants = append(ds.Variants, lib_model.DepSecretVariant{
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

func SelectConfigs(ctx context.Context, db *sql.DB, dID string, configs map[string]lib_model.DepConfig) error {
	cfgRows, err := db.QueryContext(ctx, "SELECT `ref`, `v_string`, `v_int`, `v_float`, `v_bool` FROM `configs` WHERE `dep_id` = ?", dID)
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
		dc := lib_model.DepConfig{}
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
		configs[ref] = dc
	}
	if err = cfgRows.Err(); err != nil {
		return err
	}
	return nil
}

func SelectListConfigs(ctx context.Context, db *sql.DB, dID string, configs map[string]lib_model.DepConfig) error {
	lstCfgRows, err := db.QueryContext(ctx, "SELECT `ref`, `ord`, `v_string`, `v_int`, `v_float`, `v_bool` FROM `list_configs` WHERE `dep_id` = ? ORDER BY `ref`, `ord`", dID)
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
		dc, ok := configs[ref]
		if !ok {
			dc = lib_model.DepConfig{IsSlice: true}
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
		configs[ref] = dc
	}
	if err = lstCfgRows.Err(); err != nil {
		return err
	}
	return nil
}

func SelectRequiredDep(ctx context.Context, db *sql.DB, dID string) ([]string, error) {
	return selectReq(ctx, db, "SELECT `req_id` FROM `dependencies` WHERE `dep_id` = ?", dID)
}

func SelectDepRequiring(ctx context.Context, db *sql.DB, dID string) ([]string, error) {
	return selectReq(ctx, db, "SELECT `dep_id` FROM `dependencies` WHERE `req_id` = ?", dID)
}

func SelectRequiredMod(ctx context.Context, db *sql.DB, mID string) ([]string, error) {
	return selectReq(ctx, db, "SELECT `req_id` FROM `mod_dependencies` WHERE `mod_id` = ?", mID)
}

func SelectModRequiring(ctx context.Context, db *sql.DB, mID string) ([]string, error) {
	return selectReq(ctx, db, "SELECT `mod_id` FROM `mod_dependencies` WHERE `req_id` = ?", mID)
}

func selectReq(ctx context.Context, db *sql.DB, query, ID string) ([]string, error) {
	rows, err := db.QueryContext(ctx, query, ID)
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

func SelectDepContainers(ctx context.Context, db *sql.DB, dID string) (map[string]lib_model.DepContainer, error) {
	rows, err := db.QueryContext(ctx, "SELECT `ctr_id`, `srv_ref`, `alias`, `order` FROM `containers` WHERE `dep_id` = ?", dID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	depContainers := make(map[string]lib_model.DepContainer)
	for rows.Next() {
		var depContainer lib_model.DepContainer
		if err = rows.Scan(&depContainer.ID, &depContainer.SrvRef, &depContainer.Alias, &depContainer.Order); err != nil {
			return nil, err
		}
		depContainers[depContainer.SrvRef] = depContainer
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return depContainers, nil
}
