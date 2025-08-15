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

package restructure

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
)

type Migration struct{}

func (m *Migration) Required(_ context.Context, _ *sql.DB) (bool, error) {
	return true, nil
}

func (m *Migration) Run(ctx context.Context, db *sql.DB) error {
	err := migrateAuxConfigsTab(ctx, db)
	if err != nil {
		return err
	}
	return nil
}

func migrateAuxConfigsTab(ctx context.Context, db *sql.DB) error {
	tableName := "aux_configs"
	ok, err := tableExists(ctx, db, tableName)
	if !ok {
		return nil
	}
	fmt.Printf("migrating table '%s' ...\n", tableName)
	ok, err = columnExists(ctx, db, tableName, "index")
	if err != nil {
		return err
	}
	if ok {
		fmt.Println("dropping column 'index'")
		err = dropColumn(ctx, db, tableName, "`index`")
		if err != nil {
			return err
		}
	}
	ok, err = columnExists(ctx, db, tableName, "aux_id")
	if err != nil {
		return err
	}
	if ok {
		fmt.Println("renaming column 'aux_id' -> 'aux_dep_id'")
		err = changeColumn(ctx, db, tableName, "aux_id", "aux_dep_id", "char(36)", "NOT NULL", "FIRST")
		if err != nil {
			return err
		}
	}
	err = dropForeignKey(ctx, db, "aux_configs", "aux_configs_ibfk_1")
	if err != nil {
		return err
	}
	currentIndexKeys, err := indexKeyNames(ctx, db, "aux_configs")
	if err != nil {
		return err
	}
	newIndexKeys := []string{"uk_aux_dep_id_ref", "i_aux_dep_id"}
	for _, key := range currentIndexKeys {
		if key == "PRIMARY" {
			continue
		}
		if !slices.Contains(newIndexKeys, key) {
			err = dropIndex(ctx, db, "aux_configs", key)
			if err != nil {
				return err
			}
		}
	}
	ok, err = indexExists(ctx, db, tableName, "uk_aux_dep_id_ref")
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("adding unique index for columns 'aux_dep_id', 'ref'")
		err = addUniqueIndex(ctx, db, tableName, "uk_aux_dep_id_ref", "aux_dep_id", "ref")
		if err != nil {
			return err
		}
	}
	ok, err = indexExists(ctx, db, tableName, "i_aux_dep_id")
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("adding index for column 'aux_dep_id'")
		err = addIndex(ctx, db, tableName, "i_aux_dep_id", "aux_dep_id")
		if err != nil {
			return err
		}
	}
	err = addForeignKey(ctx, db, tableName, "aux_dep_id", "aux_deployments", "id", "CASCADE", "RESTRICT")
	if err != nil {
		return err
	}
	fmt.Println("renaming table to 'aux_dep_configs'")
	return renameTable(ctx, db, tableName, "aux_dep_configs")
}
