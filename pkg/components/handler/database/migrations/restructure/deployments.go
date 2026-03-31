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
)

func migrateDeploymentsTab(ctx context.Context, db *sql.DB) error {
	tableName := "deployments"
	ok, err := tableExists(ctx, db, tableName)
	if err != nil {
		return err
	}
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
	ok, err = columnExists(ctx, db, tableName, "name")
	if err != nil {
		return err
	}
	if ok {
		logger.Info("dropping column", attrColumn, "name", attrTable, tableName)
		err = dropColumn(ctx, db, tableName, "`name`")
		if err != nil {
			return err
		}
	}
	ok, err = columnExists(ctx, db, tableName, "indirect")
	if err != nil {
		return err
	}
	if ok {
		logger.Info("dropping column", attrColumn, "indirect", attrTable, tableName)
		err = dropColumn(ctx, db, tableName, "`indirect`")
		if err != nil {
			return err
		}
	}
	ok, err = columnExists(ctx, db, tableName, "`order`")
	if err != nil {
		return err
	}
	if ok {
		logger.Info("dropping column", attrColumn, "order", attrTable, tableName)
		err = dropColumn(ctx, db, tableName, "`order`")
		if err != nil {
			return err
		}
	}
	ok, err = columnExists(ctx, db, tableName, "mod_source")
	if err != nil {
		return err
	}
	if !ok {
		logger.Info("adding column", attrColumn, "mod_source", attrTable, tableName)
		err = addColumn(ctx, db, tableName, "mod_source", "VARCHAR(512)", "NOT NULL", "AFTER mod_id")
		if err != nil {
			return err
		}
	}
	ok, err = columnExists(ctx, db, tableName, "mod_channel")
	if err != nil {
		return err
	}
	if !ok {
		logger.Info("adding column", attrColumn, "mod_channel", attrTable, tableName)
		err = addColumn(ctx, db, tableName, "mod_channel", "VARCHAR(256)", "NOT NULL", "AFTER mod_source")
		if err != nil {
			return err
		}
	}
	ok, err = columnExists(ctx, db, tableName, "files_dir")
	if err != nil {
		return err
	}
	if !ok {
		logger.Info("adding column", attrColumn, "files_dir", attrTable, tableName)
		err = addColumn(ctx, db, tableName, "files_dir", "VARCHAR(256)", "NOT NULL", "AFTER dir")
		if err != nil {
			return err
		}
	}
	ok, err = indexExists(ctx, db, tableName, "PRIMARY")
	if err != nil {
		return err
	}
	if !ok {
		logger.Info("adding primary key", attrColumn, "id", attrTable, tableName)
		err = addPrimaryKey(ctx, db, tableName, "id")
		if err != nil {
			return err
		}
	}
	ok, err = indexExists(ctx, db, tableName, "uk_mod_id")
	if err != nil {
		return err
	}
	if !ok {
		logger.Info("adding unique index", attrIndex, "uk_mod_id", attrTable, tableName)
		err = addUniqueIndex(ctx, db, tableName, "uk_mod_id", "mod_id")
		if err != nil {
			return err
		}
	}
	currentIndexKeys, err := indexKeyNames(ctx, db, tableName)
	if err != nil {
		return err
	}
	for _, key := range currentIndexKeys {
		if key == "PRIMARY" || key == "uk_mod_id" {
			continue
		}
		logger.Info("dropping index", attrIndex, key, attrTable, tableName)
		err = dropIndex(ctx, db, tableName, key)
		if err != nil {
			return err
		}
	}
	return nil
}
