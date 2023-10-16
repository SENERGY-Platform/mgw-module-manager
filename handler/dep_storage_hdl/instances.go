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
	"errors"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"strings"
	"time"
)

func (h *Handler) ListInst(ctx context.Context, filter model.DepInstFilter) ([]model.Instance, error) {
	q := "SELECT `id`, `dep_id`, `created` FROM `instances`"
	fc, val := genListInstFilter(filter)
	if fc != "" {
		q += fc
	}
	rows, err := h.db.QueryContext(ctx, q+" ORDER BY `created`", val...)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	defer rows.Close()
	var dims []model.Instance
	for rows.Next() {
		var dim model.Instance
		var ct []uint8
		if err = rows.Scan(&dim.ID, &dim.DepID, &ct); err != nil {
			return nil, model.NewInternalError(err)
		}
		tc, err := time.Parse(tLayout, string(ct))
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		dim.Created = tc
		dims = append(dims, dim)
	}
	if err = rows.Err(); err != nil {
		return nil, model.NewInternalError(err)
	}
	return dims, nil
}

func (h *Handler) ListInstCtr(ctx context.Context, iID string, filter model.CtrFilter) ([]model.Container, error) {
	q := "SELECT `srv_ref`, `order`, `ctr_id` FROM `inst_containers` WHERE `inst_id` = ? ORDER BY `order` "
	switch filter.SortOrder {
	case model.Ascending:
		q += "ASC"
	case model.Descending:
		q += "DESC"
	default:
		return nil, model.NewInvalidInputError(errors.New("invalid sort direction"))
	}
	rows, err := h.db.QueryContext(ctx, q, iID)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	defer rows.Close()
	var containers []model.Container
	for rows.Next() {
		var ctr model.Container
		if err = rows.Scan(&ctr.Ref, &ctr.Order, &ctr.ID); err != nil {
			return nil, model.NewInternalError(err)
		}
		containers = append(containers, ctr)
	}
	if err = rows.Err(); err != nil {
		return nil, model.NewInternalError(err)
	}
	return containers, nil
}

func (h *Handler) CreateInst(ctx context.Context, itf driver.Tx, dID string, timestamp time.Time) (string, error) {
	tx := itf.(*sql.Tx)
	res, err := tx.ExecContext(ctx, "INSERT INTO `instances` (`id`, `dep_id`, `created`) VALUES (UUID(), ?, ?)", dID, timestamp)
	if err != nil {
		return "", model.NewInternalError(err)
	}
	i, err := res.LastInsertId()
	if err != nil {
		return "", model.NewInternalError(err)
	}
	row := tx.QueryRowContext(ctx, "SELECT `id` FROM `instances` WHERE `index` = ?", i)
	var id string
	if err = row.Scan(&id); err != nil {
		return "", model.NewInternalError(err)
	}
	if id == "" {
		return "", model.NewInternalError(errors.New("generating id failed"))
	}
	return id, nil
}

func (h *Handler) ReadInst(ctx context.Context, id string) (model.Instance, error) {
	row := h.db.QueryRowContext(ctx, "SELECT `id`, `dep_id`, `created` FROM `instances` WHERE `id` = ?", id)
	var dim model.Instance
	var ct []uint8
	err := row.Scan(&dim.ID, &dim.DepID, &ct)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Instance{}, model.NewNotFoundError(err)
		}
		return model.Instance{}, model.NewInternalError(err)
	}
	tc, err := time.Parse(tLayout, string(ct))
	if err != nil {
		return model.Instance{}, model.NewInternalError(err)
	}
	dim.Created = tc
	return dim, nil
}

func (h *Handler) DeleteInst(ctx context.Context, id string) error {
	res, err := h.db.ExecContext(ctx, "DELETE FROM `instances` WHERE `id` = ?", id)
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

func (h *Handler) CreateInstCtr(ctx context.Context, itf driver.Tx, iID string, ctr model.Container) error {
	tx := itf.(*sql.Tx)
	res, err := tx.ExecContext(ctx, "INSERT INTO `inst_containers` (`inst_id`, `srv_ref`, `order`, `ctr_id`) VALUES (?, ?, ?, ?)", iID, ctr.Ref, ctr.Order, ctr.ID)
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

func genListInstFilter(filter model.DepInstFilter) (string, []any) {
	var fc []string
	var val []any
	if filter.DepID != "" {
		fc = append(fc, "`dep_id` = ?")
		val = append(val, filter.DepID)
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}
