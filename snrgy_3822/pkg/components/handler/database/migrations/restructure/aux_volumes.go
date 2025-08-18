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
	"slices"
)

func migrateAuxVolumesTab(ctx context.Context, db *sql.DB) error {
	tableName := "aux_volumes"
	ok, err := tableExists(ctx, db, tableName)
	if !ok {
		return nil
	}
	ok, err = columnExists(ctx, db, tableName, "index")
	if err != nil {
		return err
	}
	if ok {
		logger.Info("dropping column", attrColumn, "index", attrTable, tableName)
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
		logger.Info("renaming column", attrColumn, "aux_id", attrNewName, "aux_dep_id", attrTable, tableName)
		err = changeColumn(ctx, db, tableName, "aux_id", "aux_dep_id", "char(36)", "NOT NULL", "FIRST")
		if err != nil {
			return err
		}
	}
	ok, err = indexExists(ctx, db, tableName, "uk_aux_dep_id_name")
	if err != nil {
		return err
	}
	if !ok {
		logger.Info("adding unique index", attrIndex, "uk_aux_dep_id_name", attrTable, tableName)
		err = addUniqueIndex(ctx, db, tableName, "uk_aux_dep_id_name", "aux_dep_id", "name")
		if err != nil {
			return err
		}
	}
	ok, err = indexExists(ctx, db, tableName, "i_aux_dep_id")
	if err != nil {
		return err
	}
	if !ok {
		logger.Info("adding index", attrIndex, "i_aux_dep_id", attrTable, tableName)
		err = addIndex(ctx, db, tableName, "i_aux_dep_id", "aux_dep_id")
		if err != nil {
			return err
		}
	}
	currentIndexKeys, err := indexKeyNames(ctx, db, tableName)
	if err != nil {
		return err
	}
	newIndexKeys := []string{"uk_aux_dep_id_name", "i_aux_dep_id"}
	for _, key := range currentIndexKeys {
		if key == "PRIMARY" {
			continue
		}
		if !slices.Contains(newIndexKeys, key) {
			logger.Info("dropping index", attrIndex, key, attrTable, tableName)
			err = dropIndex(ctx, db, tableName, key)
			if err != nil {
				return err
			}
		}
	}
	logger.Info("renaming table", attrTable, tableName, attrNewName, "aux_dep_volumes")
	return renameTable(ctx, db, tableName, "aux_dep_volumes")
}
