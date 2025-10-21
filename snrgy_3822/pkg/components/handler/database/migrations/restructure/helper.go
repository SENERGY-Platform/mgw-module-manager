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
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/go-sql-driver/mysql"
)

const showIndexFromStmt = "SHOW INDEX FROM %s;"

func indexKeyNames(ctx context.Context, db *sql.DB, tableName string) ([]string, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(showIndexFromStmt, tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keyNames []string
	for rows.Next() {
		var tmp any
		var name string
		err = rows.Scan(&tmp, &tmp, &name, &tmp, &tmp, &tmp, &tmp, &tmp, &tmp, &tmp, &tmp, &tmp, &tmp, &tmp, &tmp)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(keyNames, name) {
			keyNames = append(keyNames, name)
		}
	}
	return keyNames, nil
}

const showColumnsFromStmt = "SHOW COLUMNS FROM %s WHERE Field = ?;"

func columnExists(ctx context.Context, db *sql.DB, tableName, columnName string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(showColumnsFromStmt, tableName), columnName)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	return rows.Next(), nil
}

const showIndexFromIndexStmt = "SHOW INDEX FROM %s WHERE Key_name = ?;"

func indexExists(ctx context.Context, db *sql.DB, tableName, keyName string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(showIndexFromIndexStmt, tableName), keyName)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	return rows.Next(), nil
}

const showTablesLikeStmt = "SHOW TABLES LIKE '%s';"

func tableExists(ctx context.Context, db *sql.DB, tableName string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(showTablesLikeStmt, tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	return rows.Next(), nil
}

const alterTableDropColumnStmt = "ALTER TABLE %s DROP %s;"

func dropColumn(ctx context.Context, db *sql.DB, tableName, columnName string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(alterTableDropColumnStmt, tableName, columnName))
	if err != nil {
		return err
	}
	return nil
}

const alterTableChangeColumnNameStmt = "ALTER TABLE %s CHANGE %s %s %s COLLATE 'utf8mb4_0900_ai_ci' %s %s;"

func changeColumn(ctx context.Context, db *sql.DB, tableName, columnName, columnNameNew, columnType, columnConstraint, columnPosition string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(alterTableChangeColumnNameStmt, tableName, columnName, columnNameNew, columnType, columnConstraint, columnPosition))
	if err != nil {
		return err
	}
	return nil
}

const alterTableAddColumnStmt = "ALTER TABLE %s ADD %s %s COLLATE 'utf8mb4_0900_ai_ci' %s %s;"

func addColumn(ctx context.Context, db *sql.DB, tableName, columnName, columnType, columnConstraint, columnPosition string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(alterTableAddColumnStmt, tableName, columnName, columnType, columnConstraint, columnPosition))
	if err != nil {
		return err
	}
	return nil
}

const alterTableAddIndexStmt = "ALTER TABLE %s ADD INDEX %s (%s);"

func addIndex(ctx context.Context, db *sql.DB, tableName, keyName string, columnNames ...string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(alterTableAddIndexStmt, tableName, keyName, strings.Join(columnNames, ", ")))
	if err != nil {
		return err
	}
	return nil
}

const alterTableAddUniqueIndexStmt = "ALTER TABLE %s ADD UNIQUE %s (%s);"

func addUniqueIndex(ctx context.Context, db *sql.DB, tableName, keyName string, columnNames ...string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(alterTableAddUniqueIndexStmt, tableName, keyName, strings.Join(columnNames, ", ")))
	if err != nil {
		return err
	}
	return nil
}

const alterTableAddPrimaryKeyStmt = "ALTER TABLE %s ADD PRIMARY KEY (%s);"

func addPrimaryKey(ctx context.Context, db *sql.DB, tableName string, columnNames ...string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(alterTableAddPrimaryKeyStmt, tableName, strings.Join(columnNames, ", ")))
	if err != nil {
		return err
	}
	return nil
}

const alterTableRenameStmt = "ALTER TABLE %s RENAME TO %s;"

func renameTable(ctx context.Context, db *sql.DB, tableName, tableNameNew string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(alterTableRenameStmt, tableName, tableNameNew))
	if err != nil {
		return err
	}
	return nil
}

const alterTableAddFK = "ALTER TABLE %s ADD FOREIGN KEY (%s) REFERENCES %s (%s) ON DELETE %s ON UPDATE %s;"

func addForeignKey(ctx context.Context, db *sql.DB, tableName, sourceColumn, targetTable, targetColumn, onDelete, onUpdate string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(alterTableAddFK, tableName, sourceColumn, targetTable, targetColumn, onDelete, onUpdate))
	if err != nil {
		return err
	}
	return nil
}

const alterTableDropFK = "ALTER TABLE %s DROP FOREIGN KEY %s;"

func dropForeignKey(ctx context.Context, db *sql.DB, tableName, keyName string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(alterTableDropFK, tableName, keyName))
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1091 {
			return nil
		}
		return err
	}
	return nil
}

const alterTableDropIndex = "ALTER TABLE %s DROP INDEX %s;"

func dropIndex(ctx context.Context, db *sql.DB, tableName, indexName string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(alterTableDropIndex, tableName, indexName))
	if err != nil {
		return err
	}
	return nil
}

const dropTableStmt = "DROP TABLE %s;"

func dropTable(ctx context.Context, db *sql.DB, tableName string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(dropTableStmt, tableName))
	if err != nil {
		return err
	}
	return nil
}
