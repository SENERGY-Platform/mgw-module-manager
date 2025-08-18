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

type Migration struct{}

func (m *Migration) Required(_ context.Context, _ *sql.DB) (bool, error) {
	return true, nil
}

func (m *Migration) Run(ctx context.Context, db *sql.DB) error {
	err := migrateAuxConfigsTab(ctx, db)
	if err != nil {
		return err
	}
	err = migrateAuxContainersTab(ctx, db)
	if err != nil {
		return err
	}
	err = migrateAuxLabelsTab(ctx, db)
	if err != nil {
		return err
	}
	err = migrateAuxVolumesTab(ctx, db)
	if err != nil {
		return err
	}
	err = migrateAuxDeployments(ctx, db)
	if err != nil {
		return err
	}
	err = migrateDepAdvItemsTab(ctx, db)
	if err != nil {
		return err
	}
	err = migrateDepAdvertisementsTab(ctx, db)
	if err != nil {
		return err
	}
	err = migrateConfigsTab(ctx, db)
	if err != nil {
		return err
	}
	err = migrateContainersTab(ctx, db)
	if err != nil {
		return err
	}
	return nil
}
