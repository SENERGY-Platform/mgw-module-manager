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

type migration []func(ctx context.Context, db *sql.DB) error

var Migration = migration{
	migrateAuxConfigsTab,
	migrateAuxContainersTab,
	migrateAuxLabelsTab,
	migrateAuxVolumesTab,
	migrateAuxDeployments,
	migrateDepAdvItemsTab,
	migrateDepAdvertisementsTab,
	migrateConfigsTab,
	migrateContainersTab,
	migrateHostResourcesTab,
	migrateListConfigsTab,
	migrateSecretsTab,
	migrateDeploymentsTab,
	migrateModulesTab,
	dropTables,
}

func (m migration) Required(_ context.Context, _ *sql.DB) (bool, error) {
	return true, nil
}

func (m migration) Run(ctx context.Context, db *sql.DB) error {
	for _, f := range m {
		err := f(ctx, db)
		if err != nil {
			return err
		}
	}
	return nil
}

func dropTables(ctx context.Context, db *sql.DB) error {
	ok, err := tableExists(ctx, db, "dependencies")
	if err != nil {
		return err
	}
	if ok {
		logger.Info("dropping table", attrTable, "dependencies")
		err = dropTable(ctx, db, "dependencies")
		if err != nil {
			return err
		}
	}
	ok, err = tableExists(ctx, db, "mod_dependencies")
	if err != nil {
		return err
	}
	if ok {
		logger.Info("dropping table", attrTable, "mod_dependencies")
		err = dropTable(ctx, db, "mod_dependencies")
		if err != nil {
			return err
		}
	}
	return nil
}
