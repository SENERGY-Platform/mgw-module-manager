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

package dep_storage_hdl

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

func (h *Handler) CreateDepDependencies(ctx context.Context, txItf driver.Tx, dID string, dIDs []string) error {
	prepareContext := h.db.PrepareContext
	if txItf != nil {
		tx := txItf.(*sql.Tx)
		prepareContext = tx.PrepareContext
	}
	stmt, err := prepareContext(ctx, "INSERT INTO `dependencies` (`dep_id`, `req_id`) VALUES (?, ?)")
	if err != nil {
		return model.NewInternalError(err)
	}
	defer stmt.Close()
	for _, id := range dIDs {
		if _, err = stmt.ExecContext(ctx, dID, id); err != nil {
			return model.NewInternalError(err)
		}
	}
	return nil
}

func (h *Handler) DeleteDepDependencies(ctx context.Context, txItf driver.Tx, dID string) error {
	execContext := h.db.ExecContext
	if txItf != nil {
		tx := txItf.(*sql.Tx)
		execContext = tx.ExecContext
	}
	_, err := execContext(ctx, "DELETE FROM `dependencies` WHERE `dep_id` = ?", dID)
	if err != nil {
		return model.NewInternalError(err)
	}
	return nil
}
