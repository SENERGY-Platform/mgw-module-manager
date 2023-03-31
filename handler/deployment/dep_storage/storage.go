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
	panic("not implemented")
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
	panic("not implemented")
}

func (h *StorageHandler) Update(dep *model.Deployment) error {
	panic("not implemented")
}

func (h *StorageHandler) Delete(id string) error {
	panic("not implemented")
}
