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
)

func migrateAuxContainersTab(ctx context.Context, db *sql.DB) error {
	tableName := "aux_containers"
	ok, err := tableExists(ctx, db, tableName)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	containers, err := queryAuxDepContainers(ctx, db)
	if err != nil {
		return err
	}
	logger.Info("transforming data from table", attrTable, tableName)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, container := range containers {
		_, err = tx.ExecContext(
			ctx,
			"UPDATE aux_deployments SET ctr_name = ?, ctr_alias = ? WHERE id = ?",
			container.Id,
			container.Alias,
			container.AuxId,
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

func queryAuxDepContainers(ctx context.Context, db *sql.DB) ([]auxDepContainer, error) {
	rows, err := db.QueryContext(ctx, "SELECT aux_id, ctr_id, alias FROM aux_containers")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var containers []auxDepContainer
	for rows.Next() {
		var ctr auxDepContainer
		if err = rows.Scan(&ctr.AuxId, &ctr.Id, &ctr.Alias); err != nil {
			return nil, err
		}
		containers = append(containers, ctr)
	}
	return containers, nil
}

type auxDepContainer struct {
	Id    string
	AuxId string
	Alias string
}
