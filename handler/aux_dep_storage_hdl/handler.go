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

package aux_dep_storage_hdl

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"strings"
	"time"
)

const tLayout = "2006-01-02 15:04:05.000000"

type Handler struct {
	db *sql.DB
}

func New(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) BeginTransaction(ctx context.Context) (driver.Tx, error) {
	tx, e := h.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, model.NewInternalError(e)
	}
	return tx, nil
}

func (h *Handler) List(ctx context.Context, filter model.AuxDepFilter) ([]model.AuxDeployment, error) {
	q := "SELECT `id`, `dep_id`, `image`, `ctr_id`, `created`, `updated`, `type`, `name` FROM `aux_deployments`"
	fc, val := genListFilter(filter)
	if fc != "" {
		q += fc
	}
	q += " ORDER BY `created`"
	rows, err := h.db.QueryContext(ctx, q, val...)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	defer rows.Close()
	var auxDeployments []model.AuxDeployment
	for rows.Next() {
		var auxDep model.AuxDeployment
		var ct, ut []uint8
		if err = rows.Scan(&auxDep.ID, &auxDep.DepID, &auxDep.Image, &auxDep.CtrID, &ct, &ut, &auxDep.Type, &auxDep.Name); err != nil {
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
		labels, err := h.selectLabels(ctx, auxDep.ID)
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		configs, err := h.selectConfigs(ctx, auxDep.ID)
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		auxDep.Created = tc
		auxDep.Updated = tu
		auxDep.Labels = labels
		auxDep.Configs = configs
	}
	if err = rows.Err(); err != nil {
		return nil, model.NewInternalError(err)
	}
	return auxDeployments, nil
}

func (h *Handler) Create(ctx context.Context, itf driver.Tx, auxDep model.AuxDeployment) (string, error) {
	tx := itf.(*sql.Tx)
	res, err := tx.ExecContext(ctx, "INSERT INTO `aux_deployments` (`id`, `dep_id`, `image`, `ctr_id`, `created`, `updated`, `type`, `name`) VALUES (UUID(), ?, ?, ?, ?, ?, ?, ?, )", auxDep.DepID, auxDep.Image, auxDep.CtrID, auxDep.Created, auxDep.Updated, auxDep.Type, auxDep.Name)
	if err != nil {
		return "", model.NewInternalError(err)
	}
	i, err := res.LastInsertId()
	if err != nil {
		return "", model.NewInternalError(err)
	}
	row := tx.QueryRowContext(ctx, "SELECT `id` FROM `aux_deployments` WHERE `index` = ?", i)
	var id string
	if err = row.Scan(&id); err != nil {
		return "", model.NewInternalError(err)
	}
	if id == "" {
		return "", model.NewInternalError(errors.New("generating id failed"))
	}
	if len(auxDep.Labels) > 0 {
		if err = h.insertLabels(ctx, tx, id, auxDep.Labels); err != nil {
			return "", model.NewInternalError(err)
		}
	}
	if len(auxDep.Configs) > 0 {
		if err = h.insertConfigs(ctx, tx, id, auxDep.Configs); err != nil {
			return "", model.NewInternalError(err)
		}
	}
	return id, nil
}

func (h *Handler) Read(ctx context.Context, id string) (model.AuxDeployment, error) {
	row := h.db.QueryRowContext(ctx, "SELECT `id`, `dep_id`, `image`, `ctr_id`, `created`, `updated`, `type`, `name` FROM `aux_deployments` WHERE `id` = ?", id)
	var auxDep model.AuxDeployment
	var ct, ut []uint8
	err := row.Scan(&auxDep.ID, &auxDep.DepID, &auxDep.Image, &auxDep.CtrID, &ct, &ut, &auxDep.Type, &auxDep.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.AuxDeployment{}, model.NewNotFoundError(err)
		}
		return model.AuxDeployment{}, model.NewInternalError(err)
	}
	tc, err := time.Parse(tLayout, string(ct))
	if err != nil {
		return model.AuxDeployment{}, model.NewInternalError(err)
	}
	tu, err := time.Parse(tLayout, string(ut))
	if err != nil {
		return model.AuxDeployment{}, model.NewInternalError(err)
	}
	labels, err := h.selectLabels(ctx, auxDep.ID)
	if err != nil {
		return model.AuxDeployment{}, model.NewInternalError(err)
	}
	configs, err := h.selectConfigs(ctx, auxDep.ID)
	if err != nil {
		return model.AuxDeployment{}, model.NewInternalError(err)
	}
	auxDep.Created = tc
	auxDep.Updated = tu
	auxDep.Labels = labels
	auxDep.Configs = configs
	return auxDep, nil
}

func (h *Handler) Update(ctx context.Context, itf driver.Tx, auxDep model.AuxDeployment) error {
	tx := itf.(*sql.Tx)
	res, err := tx.ExecContext(ctx, "UPDATE `aux_deployments` SET `image` = ?, `ctr_id` = ?, `updated` = ?, `name` = ? WHERE `id` = ?", auxDep.Image, auxDep.CtrID, auxDep.Updated, auxDep.Name, auxDep.ID)
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
	_, err = tx.ExecContext(ctx, "DELETE FROM `aux_labels` WHERE `id` = ?", auxDep.ID)
	if err != nil {
		return model.NewInternalError(err)
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM `aux_configs` WHERE `id` = ?", auxDep.ID)
	if err != nil {
		return model.NewInternalError(err)
	}
	if len(auxDep.Labels) > 0 {
		if err = h.insertLabels(ctx, tx, auxDep.ID, auxDep.Labels); err != nil {
			return model.NewInternalError(err)
		}
	}
	if len(auxDep.Configs) > 0 {
		if err = h.insertConfigs(ctx, tx, auxDep.ID, auxDep.Configs); err != nil {
			return model.NewInternalError(err)
		}
	}
	return nil
}

func (h *Handler) Delete(ctx context.Context, id string) error {
	res, err := h.db.ExecContext(ctx, "DELETE FROM `aux_deployments` WHERE `id` = ?", id)
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

func (h *Handler) selectLabels(ctx context.Context, id string) (map[string]string, error) {
	return selectStrMap(ctx, h.db.QueryContext, "SELECT `name`, `value` FROM `aux_labels` WHERE `aux_id` = ?", id)
}

func (h *Handler) selectConfigs(ctx context.Context, id string) (map[string]string, error) {
	return selectStrMap(ctx, h.db.QueryContext, "SELECT `ref`, `value` FROM `aux_configs` WHERE `aux_id` = ?", id)
}

func (h *Handler) insertLabels(ctx context.Context, tx *sql.Tx, id string, m map[string]string) error {
	return insertStrMap(ctx, tx, "INSERT INTO `aux_labels` (`aux_id`, `name`, `value`) VALUES (?, ?, ?)", id, m)
}

func (h *Handler) insertConfigs(ctx context.Context, tx *sql.Tx, id string, m map[string]string) error {
	return insertStrMap(ctx, tx, "INSERT INTO `aux_configs` (`aux_id`, `ref`, `value`) VALUES (?, ?, ?)", id, m)
}

func genListFilter(filter model.AuxDepFilter) (string, []any) {
	var str string
	var val []any
	tc := 0
	if len(filter.Labels) > 0 {
		for n, v := range filter.Labels {
			var fl []string
			fl = append(fl, "`name` = ?")
			val = append(val, n)
			fl = append(fl, "`value` = ?")
			val = append(val, v)
			if tc == 0 {
				str = fmt.Sprintf("SELECT t%d.* FROM (SELECT `aux_id` FROM `aux_labels` WHERE %s) t%d", tc, strings.Join(fl, " AND "), tc)
			} else {
				str += fmt.Sprintf(" INNER JOIN (SELECT `aux_id` FROM `aux_labels` WHERE %s) t%d ON t%d.aux_id = t%d.aux_id", strings.Join(fl, " AND "), tc, tc-1, tc)
			}
			tc += 1
		}
		str = " `id` IN (" + str + ")"
	}
	if filter.Image != "" {
		if str != "" {
			str += " AND"
		}
		str += " `image` = ?"
		val = append(val, filter.Image)
	}
	if len(val) > 0 {
		return " WHERE" + str, val
	}
	return "", nil
}

func selectStrMap(ctx context.Context, qf func(ctx context.Context, query string, args ...any) (*sql.Rows, error), query, id string) (map[string]string, error) {
	rows, err := qf(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]string)
	for rows.Next() {
		var key string
		var val string
		if err = rows.Scan(&key, &val); err != nil {
			return nil, model.NewInternalError(err)
		}
		m[key] = val
	}
	if err = rows.Err(); err != nil {
		return nil, model.NewInternalError(err)
	}
	return m, nil
}

func insertStrMap(ctx context.Context, tx *sql.Tx, query, id string, m map[string]string) error {
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return model.NewInternalError(err)
	}
	for key, val := range m {
		if _, err = stmt.ExecContext(ctx, id, key, val); err != nil {
			return model.NewInternalError(err)
		}
	}
	return nil
}
