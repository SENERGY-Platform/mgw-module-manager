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

package dep_storage

import (
	"context"
	"database/sql"
	"module-manager/model"
	"time"
)

const tLayout = "2006-01-02 15:04:05.000000"

type StorageHandler struct {
	db      *sql.DB
	ctx     context.Context
	timeout time.Duration
}

func NewStorageHandler(db *sql.DB, ctx context.Context, timeout time.Duration) *StorageHandler {
	return &StorageHandler{
		db:      db,
		ctx:     ctx,
		timeout: timeout,
	}
}

func (h *StorageHandler) List() ([]model.DepMeta, error) {
	ctx, cf := context.WithTimeout(h.ctx, h.timeout)
	defer cf()
	rows, err := h.db.QueryContext(ctx, "SELECT `id`, `module_id`, `name`, `created`, `updated` FROM `deployments` ORDER BY `name`")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dms []model.DepMeta
	for rows.Next() {
		var dm model.DepMeta
		var ct, ut []uint8
		if err = rows.Scan(&dm.ID, &dm.ModuleID, &dm.Name, &ct, &ut); err != nil {
			return nil, err
		}
		tc, err := time.Parse(tLayout, string(ct))
		if err != nil {
			return nil, err
		}
		tu, err := time.Parse(tLayout, string(ut))
		if err != nil {
			return nil, err
		}
		dm.Created = tc
		dm.Updated = tu
		dms = append(dms, dm)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return dms, nil
}

func (h *StorageHandler) Create(dep *model.Deployment) (string, error) {
	ctx, cf := context.WithTimeout(h.ctx, h.timeout)
	defer cf()
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	id, err := insertDeployment(ctx, tx.ExecContext, tx.QueryRowContext, dep.ModuleID, dep.Name, dep.Created)
	if err != nil {
		return "", err
	}
	if len(dep.HostResources) > 0 {
		err = insertHostResources(ctx, tx.PrepareContext, id, dep.HostResources)
		if err != nil {
			return "", err
		}
	}
	if len(dep.Secrets) > 0 {
		err = insertSecrets(ctx, tx.PrepareContext, id, dep.Secrets)
		if err != nil {
			return "", err
		}
	}
	if len(dep.Configs) > 0 {
		err = insertConfigs(ctx, tx.PrepareContext, id, dep.Configs)
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	return id, nil
}

func (h *StorageHandler) Read(id string) (*model.Deployment, error) {
	ctx, cf := context.WithTimeout(h.ctx, h.timeout)
	defer cf()
	depMeta, err := selectDeployment(ctx, h.db.QueryRowContext, id)
	if err != nil {
		return nil, err
	}
	depMeta.ID = id
	hostRes, err := selectHostResources(ctx, h.db.QueryContext, id)
	if err != nil {
		return nil, err
	}
	secrets, err := selectSecrets(ctx, h.db.QueryContext, id)
	if err != nil {
		return nil, err
	}
	configs := make(map[string]model.DepConfig)
	err = selectConfigs(ctx, h.db.QueryContext, id, configs)
	if err != nil {
		return nil, err
	}
	err = selectListConfigs(ctx, h.db.QueryContext, id, configs)
	if err != nil {
		return nil, err
	}
	dep := model.Deployment{
		DepMeta:       depMeta,
		HostResources: hostRes,
		Secrets:       secrets,
		Configs:       configs,
	}
	return &dep, nil
}

func (h *StorageHandler) Update(dep *model.Deployment) error {
	panic("not implemented")
}

func (h *StorageHandler) Delete(id string) error {
	panic("not implemented")
}
