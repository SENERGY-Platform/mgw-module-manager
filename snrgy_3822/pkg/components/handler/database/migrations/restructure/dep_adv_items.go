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

func migrateDepAdvItemsTab(ctx context.Context, db *sql.DB) error {
	tableName := "dep_adv_items"
	ok, err := columnExists(ctx, db, tableName, "index")
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
	ok, err = columnExists(ctx, db, tableName, "adv_id")
	if err != nil {
		return err
	}
	if ok {
		logger.Info("renaming column", attrColumn, "adv_id", attrNewName, "dep_adv_id", attrTable, tableName)
		err = changeColumn(ctx, db, tableName, "adv_id", "dep_adv_id", "char(36)", "NOT NULL", "FIRST")
		if err != nil {
			return err
		}
	}
	ok, err = columnExists(ctx, db, tableName, "key")
	if err != nil {
		return err
	}
	if ok {
		logger.Info("renaming column", attrColumn, "key", attrNewName, "item_key", attrTable, tableName)
		err = changeColumn(ctx, db, tableName, "`key`", "item_key", "varchar(256)", "NOT NULL", "AFTER dep_adv_id")
		if err != nil {
			return err
		}
	}
	ok, err = columnExists(ctx, db, tableName, "value")
	if err != nil {
		return err
	}
	if ok {
		logger.Info("renaming column", attrColumn, "value", attrNewName, "item_value", attrTable, tableName)
		err = changeColumn(ctx, db, tableName, "value", "item_value", "varchar(512)", "NULL", "AFTER item_key")
		if err != nil {
			return err
		}
	}

	ok, err = indexExists(ctx, db, tableName, "uk_dep_adv_id_item_key")
	if err != nil {
		return err
	}
	if !ok {
		logger.Info("adding unique index", attrIndex, "uk_dep_adv_id_item_key", attrTable, tableName)
		err = addUniqueIndex(ctx, db, tableName, "uk_dep_adv_id_item_key", "dep_adv_id", "item_key")
		if err != nil {
			return err
		}
	}
	ok, err = indexExists(ctx, db, tableName, "i_dep_adv_id")
	if err != nil {
		return err
	}
	if !ok {
		logger.Info("adding index", attrIndex, "i_dep_adv_id", attrTable, tableName)
		err = addIndex(ctx, db, tableName, "i_dep_adv_id", "dep_adv_id")
		if err != nil {
			return err
		}
	}
	currentIndexKeys, err := indexKeyNames(ctx, db, tableName)
	if err != nil {
		return err
	}
	newIndexKeys := []string{"uk_dep_adv_id_item_key", "i_dep_adv_id"}
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
	return nil
}
