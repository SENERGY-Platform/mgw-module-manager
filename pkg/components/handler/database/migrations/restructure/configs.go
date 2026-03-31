/*
 * Copyright 2025 InfAI (CC SES)
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

package migration_db_restructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/config"
)

func createDepConfigTables(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS dep_configs
(
id        VARCHAR(256) NOT NULL,
dep_id    CHAR(36)     NOT NULL,
ref       VARCHAR(128) NOT NULL,
data_type SMALLINT     NOT NULL,
is_list   BOOLEAN      NOT NULL,
PRIMARY KEY (id),
UNIQUE KEY uk_dep_id_ref (dep_id, ref),
INDEX i_dep_id (dep_id),
FOREIGN KEY (dep_id) REFERENCES deployments (id) ON DELETE CASCADE ON UPDATE RESTRICT
);`,
	)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS dep_config_values
(
c_id     VARCHAR(256) NOT NULL,
v_string VARCHAR(512),
v_int    BIGINT,
v_float  DOUBLE,
v_bool   BOOLEAN,
ord      SMALLINT     NOT NULL,
UNIQUE KEY uk_c_id_ord (c_id, ord),
INDEX i_c_id (c_id),
FOREIGN KEY (c_id) REFERENCES dep_configs (id) ON DELETE CASCADE ON UPDATE RESTRICT
);`,
	)
	if err != nil {
		return err
	}
	return nil
}

func migrateConfigsTab(ctx context.Context, db *sql.DB) error {
	tableName := "configs"
	ok, err := tableExists(ctx, db, tableName)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	configs, err := queryConfigs(ctx, db)
	if err != nil {
		return err
	}
	if len(configs) == 0 {
		return nil
	}
	logger.Info("transforming data from table", attrTable, tableName)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, c := range configs {
		cId := c.DepId + "_" + c.Ref
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO dep_configs (id, dep_id, ref, data_type, is_list) VALUES (?, ?, ?, ?, ?)",
			cId,
			c.DepId,
			c.Ref,
			c.DT,
			false,
		)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(
			ctx,
			fmt.Sprintf("INSERT INTO dep_config_values (c_id, %s, ord) VALUES (?, ?, ?)", getColName(c.DT)),
			cId,
			c.Val,
			0,
		)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	logger.Info("dropping table", attrTable, tableName)
	return dropTable(ctx, db, tableName)
}

func migrateListConfigsTab(ctx context.Context, db *sql.DB) error {
	tableName := "list_configs"
	ok, err := tableExists(ctx, db, tableName)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	configs, err := queryListConfigs(ctx, db)
	if err != nil {
		return err
	}
	if len(configs) == 0 {
		return nil
	}
	logger.Info("transforming data from table", attrTable, tableName)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, c := range configs {
		cId := c.DepId + "_" + c.Ref
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO dep_configs (id, dep_id, ref, data_type, is_list) VALUES (?, ?, ?, ?, ?)",
			cId,
			c.DepId,
			c.Ref,
			c.DT,
			false,
		)
		if err != nil {
			return err
		}
		stmt := fmt.Sprintf("INSERT INTO dep_config_values (c_id, %s, ord) VALUES (?, ?, ?)", getColName(c.DT))
		for i, val := range c.Vals {
			_, err = tx.ExecContext(ctx, stmt, cId, val, i)
			if err != nil {
				return err
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	logger.Info("dropping table", attrTable, tableName)
	return dropTable(ctx, db, tableName)
}

type config struct {
	DepId string
	Ref   string
	DT    int
	Val   any
}

type listConfig struct {
	DepId string
	Ref   string
	DT    int
	Vals  []any
}

func queryConfigs(ctx context.Context, db *sql.DB) ([]config, error) {
	rows, err := db.QueryContext(ctx, "SELECT dep_id, ref, v_string, v_int, v_float, v_bool FROM configs")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var configs []config
	for rows.Next() {
		var depId string
		var ref string
		var vString sql.NullString
		var vInt sql.NullInt64
		var vFloat sql.NullFloat64
		var vBool sql.NullBool
		if err = rows.Scan(&depId, &ref, &vString, &vInt, &vFloat, &vBool); err != nil {
			return nil, err
		}
		c := config{
			DepId: depId,
			Ref:   ref,
		}
		if vString.Valid {
			c.Val = vString.String
			c.DT = models_config.StringType
		} else if vInt.Valid {
			c.Val = vInt.Int64
			c.DT = models_config.Int64Type
		} else if vFloat.Valid {
			c.Val = vFloat.Float64
			c.DT = models_config.Float64Type
		} else if vBool.Valid {
			c.Val = vBool.Bool
			c.DT = models_config.BoolType
		}
		configs = append(configs, c)
	}
	return configs, nil
}

func queryListConfigs(ctx context.Context, db *sql.DB) ([]listConfig, error) {
	rows, err := db.QueryContext(ctx, "SELECT dep_id, ref, ord, v_string, v_int, v_float, v_bool FROM list_configs ORDER BY dep_id, ref, ord")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	listConfigsMap := make(map[string]map[string]listConfig)
	for rows.Next() {
		var depId string
		var ref string
		var ord int
		var vString sql.NullString
		var vInt sql.NullInt64
		var vFloat sql.NullFloat64
		var vBool sql.NullBool
		if err = rows.Scan(&depId, &ref, &ord, &vString, &vInt, &vFloat, &vBool); err != nil {
			return nil, err
		}
		configsMap, ok := listConfigsMap[depId]
		if !ok {
			configsMap = make(map[string]listConfig)
			listConfigsMap[depId] = configsMap
		}
		lc, ok := configsMap[ref]
		if !ok {
			lc.DepId = depId
			lc.Ref = ref
		}
		var dt int
		if vString.Valid {
			lc.Vals = append(lc.Vals, vString.String)
			dt = models_config.StringType
		} else if vInt.Valid {
			lc.Vals = append(lc.Vals, vInt.Int64)
			dt = models_config.Int64Type
		} else if vFloat.Valid {
			lc.Vals = append(lc.Vals, vFloat.Float64)
			dt = models_config.Float64Type
		} else if vBool.Valid {
			lc.Vals = append(lc.Vals, vBool.Bool)
			dt = models_config.BoolType
		}
		if !ok {
			lc.DT = dt
		} else {
			if dt != lc.DT {
				return nil, errors.New("mismatched datatype")
			}
		}
		configsMap[ref] = lc
	}
	var listConfigs []listConfig
	for _, configsMap := range listConfigsMap {
		listConfigs = append(listConfigs, slices.Collect(maps.Values(configsMap))...)
	}
	return listConfigs, nil
}

func getColName(dt int) string {
	switch dt {
	case models_config.StringType:
		return "v_string"
	case models_config.Int64Type:
		return "v_int"
	case models_config.Float64Type:
		return "v_float"
	case models_config.BoolType:
		return "v_bool"
	}
	return ""
}
