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

package storage_hdl

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/storage_hdl/dep_util"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

func (h *Handler) CreateDepAssets(ctx context.Context, txItf driver.Tx, dID string, depAssets lib_model.DepAssets) error {
	var tx *sql.Tx
	if txItf != nil {
		tx = txItf.(*sql.Tx)
	} else {
		var e error
		if tx, e = h.db.BeginTx(ctx, nil); e != nil {
			return lib_model.NewInternalError(e)
		}
		defer tx.Rollback()
	}
	if err := dep_util.InsertDepHostRes(ctx, tx, dID, depAssets.HostResources); err != nil {
		return err
	}
	if err := dep_util.InsertDepSecrets(ctx, tx, dID, depAssets.Secrets); err != nil {
		return err
	}
	if err := dep_util.InsertDepConfigs(ctx, tx, dID, depAssets.Configs); err != nil {
		return err
	}
	if txItf == nil {
		if err := tx.Commit(); err != nil {
			return lib_model.NewInternalError(err)
		}
	}
	return nil
}

func (h *Handler) DeleteDepAssets(ctx context.Context, txItf driver.Tx, dID string) error {
	var tx *sql.Tx
	if txItf != nil {
		tx = txItf.(*sql.Tx)
	} else {
		var e error
		if tx, e = h.db.BeginTx(ctx, nil); e != nil {
			return lib_model.NewInternalError(e)
		}
		defer tx.Rollback()
	}
	if err := dep_util.DeleteDepHostRes(ctx, tx, dID); err != nil {
		return err
	}
	if err := dep_util.DeleteDepSecrets(ctx, tx, dID); err != nil {
		return err
	}
	if err := dep_util.DeleteDepConfigs(ctx, tx, dID); err != nil {
		return err
	}
	if txItf == nil {
		if err := tx.Commit(); err != nil {
			return lib_model.NewInternalError(err)
		}
	}
	return nil
}
