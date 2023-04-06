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
	"errors"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"time"
)

const tLayout = "2006-01-02 15:04:05.000000"

type StorageHandler struct {
	db *sql.DB
}

func NewStorageHandler(db *sql.DB) *StorageHandler {
	return &StorageHandler{db: db}
}

func (h *StorageHandler) ListDep(ctx context.Context) ([]model.DepMeta, error) {
	rows, err := h.db.QueryContext(ctx, "SELECT `id`, `module_id`, `name`, `created`, `updated` FROM `deployments` ORDER BY `name`")
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	defer rows.Close()
	var dms []model.DepMeta
	for rows.Next() {
		var dm model.DepMeta
		var ct, ut []uint8
		if err = rows.Scan(&dm.ID, &dm.ModuleID, &dm.Name, &ct, &ut); err != nil {
			return nil, model.NewInternalError(err)
		}
		tc, err := time.Parse(tLayout, string(ct))
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		tu, err := time.Parse(tLayout, string(ut))
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		dm.Created = tc
		dm.Updated = tu
		dms = append(dms, dm)
	}
	if err = rows.Err(); err != nil {
		return nil, model.NewInternalError(err)
	}
	return dms, nil
}

func (h *StorageHandler) CreateDep(ctx context.Context, dep *model.Deployment) (handler.Transaction, string, error) {
	tx, e := h.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, "", model.NewInternalError(e)
	}
	var err error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	var id string
	id, err = insertDeployment(ctx, tx.ExecContext, tx.QueryRowContext, dep.ModuleID, dep.Name, dep.Created)
	if err != nil {
		return nil, "", model.NewInternalError(err)
	}
	if len(dep.HostResources) > 0 {
		err = insertHostResources(ctx, tx.PrepareContext, id, dep.HostResources)
		if err != nil {
			return nil, "", model.NewInternalError(err)
		}
	}
	if len(dep.Secrets) > 0 {
		err = insertSecrets(ctx, tx.PrepareContext, id, dep.Secrets)
		if err != nil {
			return nil, "", model.NewInternalError(err)
		}
	}
	if len(dep.Configs) > 0 {
		err = insertConfigs(ctx, tx.PrepareContext, id, dep.Configs)
		if err != nil {
			return nil, "", model.NewInternalError(err)
		}
	}
	return tx, id, nil
}

func (h *StorageHandler) ReadDep(ctx context.Context, id string) (*model.Deployment, error) {
	depMeta, err := selectDeployment(ctx, h.db.QueryRowContext, id)
	if err != nil {
		return nil, err
	}
	depMeta.ID = id
	hostRes, err := selectHostResources(ctx, h.db.QueryContext, id)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	secrets, err := selectSecrets(ctx, h.db.QueryContext, id)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	configs := make(map[string]model.DepConfig)
	err = selectConfigs(ctx, h.db.QueryContext, id, configs)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	err = selectListConfigs(ctx, h.db.QueryContext, id, configs)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	dep := model.Deployment{
		DepMeta:       depMeta,
		HostResources: hostRes,
		Secrets:       secrets,
		Configs:       configs,
	}
	return &dep, nil
}

func (h *StorageHandler) UpdateDep(ctx context.Context, dep *model.Deployment) (handler.Transaction, error) {
	tx, e := h.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, model.NewInternalError(e)
	}
	var err error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	res, err := tx.ExecContext(ctx, "UPDATE `deployments` SET `name` = ?, `updated` = ? WHERE `id` = ?", dep.Name, dep.Updated, dep.ID)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	if n < 1 {
		return nil, model.NewNotFoundError(errors.New("no rows affected"))
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM `host_resources` WHERE `dep_id` = ?", dep.ID)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM `secrets` WHERE `dep_id` = ?", dep.ID)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM `configs` WHERE `dep_id` = ?", dep.ID)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM `list_configs` WHERE `dep_id` = ?", dep.ID)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	if len(dep.HostResources) > 0 {
		err = insertHostResources(ctx, tx.PrepareContext, dep.ID, dep.HostResources)
		if err != nil {
			return nil, model.NewInternalError(err)
		}
	}
	if len(dep.Secrets) > 0 {
		err = insertSecrets(ctx, tx.PrepareContext, dep.ID, dep.Secrets)
		if err != nil {
			return nil, model.NewInternalError(err)
		}
	}
	if len(dep.Configs) > 0 {
		err = insertConfigs(ctx, tx.PrepareContext, dep.ID, dep.Configs)
		if err != nil {
			return nil, model.NewInternalError(err)
		}
	}
	return tx, nil
}

func (h *StorageHandler) DeleteDep(ctx context.Context, id string) error {
	res, err := h.db.ExecContext(ctx, "DELETE FROM `deployments` WHERE `id` = ?", id)
	if err != nil {
		return model.NewInternalError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return model.NewInternalError(err)
	}
	if n < 1 {
		return model.NewNotFoundError(errors.New("no rows affected"))
	}
	return nil
}

func (h *StorageHandler) ListInst(ctx context.Context) ([]model.DepInstanceMeta, error) {
	rows, err := h.db.QueryContext(ctx, "SELECT `id`, `dep_id`, `created`, `updated` FROM `instances` ORDER BY `created`")
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	defer rows.Close()
	var dims []model.DepInstanceMeta
	for rows.Next() {
		var dim model.DepInstanceMeta
		var ct, ut []uint8
		if err = rows.Scan(&dim.ID, &dim.DepID, &ct, &ut); err != nil {
			return nil, model.NewInternalError(err)
		}
		tc, err := time.Parse(tLayout, string(ct))
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		tu, err := time.Parse(tLayout, string(ut))
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		dim.Created = tc
		dim.Updated = tu
		dims = append(dims, dim)
	}
	if err = rows.Err(); err != nil {
		return nil, model.NewInternalError(err)
	}
	return dims, nil
}

func (h *StorageHandler) CreateInst(ctx context.Context, inst *model.DepInstance) (handler.Transaction, string, error) {
	panic("not implemented")
}

func (h *StorageHandler) ReadInst(ctx context.Context, id string) (*model.DepInstance, error) {
	panic("not implemented")
}

func (h *StorageHandler) UpdateInst(ctx context.Context, inst *model.DepInstance) (handler.Transaction, error) {
	panic("not implemented")
}

func (h *StorageHandler) DeleteInst(ctx context.Context, id string) error {
	panic("not implemented")
}
