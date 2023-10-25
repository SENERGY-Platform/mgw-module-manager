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
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"strings"
	"time"
)

func (h *Handler) ListAuxDep(ctx context.Context, dID string, filter model.AuxDepFilter) ([]model.AuxDeployment, error) {
	q := "SELECT `id`, `dep_id`, `image`, `created`, `updated`, `ref`, `name` FROM `aux_deployments`"
	fc, val := genAuxDepFilter(dID, filter)
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
		var auxDepBase model.AuxDepBase
		var ct, ut []uint8
		if err = rows.Scan(&auxDep.ID, &auxDepBase.DepID, &auxDepBase.Image, &ct, &ut, &auxDepBase.Ref, &auxDepBase.Name); err != nil {
			return nil, model.NewInternalError(err)
		}
		auxDepBase.Created, err = time.Parse(tLayout, string(ct))
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		auxDepBase.Updated, err = time.Parse(tLayout, string(ut))
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		auxDepBase.Labels, err = h.selectAuxDepLabels(ctx, auxDep.ID)
		if err != nil {
			return nil, err
		}
		auxDepBase.Configs, err = h.selectAuxDepConfigs(ctx, auxDep.ID)
		if err != nil {
			return nil, err
		}
		auxDep.Container, err = h.selectAuxDepContainer(ctx, auxDep.ID)
		if err != nil {
			return nil, err
		}
		auxDep.AuxDepBase = auxDepBase
		auxDeployments = append(auxDeployments, auxDep)
	}
	if err = rows.Err(); err != nil {
		return nil, model.NewInternalError(err)
	}
	return auxDeployments, nil
}

func (h *Handler) CreateAuxDep(ctx context.Context, itf driver.Tx, auxDep model.AuxDepBase) (string, error) {
	tx := itf.(*sql.Tx)
	res, err := tx.ExecContext(ctx, "INSERT INTO `aux_deployments` (`id`, `dep_id`, `image`, `created`, `updated`, `ref`, `name`) VALUES (UUID(), ?, ?, ?, ?, ?, ?, ?, )", auxDep.DepID, auxDep.Image, auxDep.Created, auxDep.Updated, auxDep.Ref, auxDep.Name)
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
		if err = h.insertAuxDepLabels(ctx, tx, id, auxDep.Labels); err != nil {
			return "", model.NewInternalError(err)
		}
	}
	if len(auxDep.Configs) > 0 {
		if err = h.insertAuxDepConfigs(ctx, tx, id, auxDep.Configs); err != nil {
			return "", model.NewInternalError(err)
		}
	}
	return id, nil
}

func (h *Handler) CreateAuxDepCtr(ctx context.Context, itf driver.Tx, aID string, ctr model.AuxDepContainer) error {
	tx := itf.(*sql.Tx)
	_, err := tx.ExecContext(ctx, "INSERT INTO `aux_containers` (`aux_id`, `ctr_id`, `alias`) VALUES (?, ?, ?)", aID, ctr.ID, ctr.Alias)
	if err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) ReadAuxDep(ctx context.Context, aID string) (model.AuxDeployment, error) {
	row := h.db.QueryRowContext(ctx, "SELECT `id`, `dep_id`, `image`, `created`, `updated`, `ref`, `name` FROM `aux_deployments` WHERE `id` = ?", aID)
	var auxDep model.AuxDeployment
	var auxDepBase model.AuxDepBase
	var ct, ut []uint8
	err := row.Scan(&auxDep.ID, &auxDepBase.DepID, &auxDepBase.Image, &ct, &ut, &auxDepBase.Ref, &auxDepBase.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.AuxDeployment{}, model.NewNotFoundError(err)
		}
		return model.AuxDeployment{}, model.NewInternalError(err)
	}
	auxDepBase.Created, err = time.Parse(tLayout, string(ct))
	if err != nil {
		return model.AuxDeployment{}, model.NewInternalError(err)
	}
	auxDepBase.Updated, err = time.Parse(tLayout, string(ut))
	if err != nil {
		return model.AuxDeployment{}, model.NewInternalError(err)
	}
	auxDepBase.Labels, err = h.selectAuxDepLabels(ctx, auxDep.ID)
	if err != nil {
		return model.AuxDeployment{}, err
	}
	auxDepBase.Configs, err = h.selectAuxDepConfigs(ctx, auxDep.ID)
	if err != nil {
		return model.AuxDeployment{}, err
	}
	auxDep.Container, err = h.selectAuxDepContainer(ctx, auxDep.ID)
	if err != nil {
		return model.AuxDeployment{}, err
	}
	auxDep.AuxDepBase = auxDepBase
	return auxDep, nil
}

func (h *Handler) UpdateAuxDep(ctx context.Context, itf driver.Tx, aID string, auxDep model.AuxDepBase) error {
	tx := itf.(*sql.Tx)
	res, err := tx.ExecContext(ctx, "UPDATE `aux_deployments` SET `image` = ?, `updated` = ?, `name` = ? WHERE `id` = ?", auxDep.Image, auxDep.Updated, auxDep.Name, auxDep.DepID, aID)
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
	_, err = tx.ExecContext(ctx, "DELETE FROM `aux_labels` WHERE `id` = ?", aID)
	if err != nil {
		return model.NewInternalError(err)
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM `aux_configs` WHERE `id` = ?", aID)
	if err != nil {
		return model.NewInternalError(err)
	}
	if len(auxDep.Labels) > 0 {
		if err = h.insertAuxDepLabels(ctx, tx, aID, auxDep.Labels); err != nil {
			return model.NewInternalError(err)
		}
	}
	if len(auxDep.Configs) > 0 {
		if err = h.insertAuxDepConfigs(ctx, tx, aID, auxDep.Configs); err != nil {
			return model.NewInternalError(err)
		}
	}
	return nil
}

func (h *Handler) DeleteAuxDep(ctx context.Context, aID string) error {
	res, err := h.db.ExecContext(ctx, "DELETE FROM `aux_deployments` WHERE `id` = ?", aID)
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

func (h *Handler) DeleteAuxDepCtr(ctx context.Context, itf driver.Tx, aID string) error {
	tx := itf.(*sql.Tx)
	_, err := tx.ExecContext(ctx, "DELETE FROM `aux_containers` WHERE `id` = ?", aID)
	if err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) selectAuxDepLabels(ctx context.Context, id string) (map[string]string, error) {
	return selectStrMap(ctx, h.db.QueryContext, "SELECT `name`, `value` FROM `aux_labels` WHERE `aux_id` = ?", id)
}

func (h *Handler) selectAuxDepConfigs(ctx context.Context, id string) (map[string]string, error) {
	return selectStrMap(ctx, h.db.QueryContext, "SELECT `ref`, `value` FROM `aux_configs` WHERE `aux_id` = ?", id)
}

func (h *Handler) selectAuxDepContainer(ctx context.Context, id string) (model.AuxDepContainer, error) {
	row := h.db.QueryRowContext(ctx, "SELECT `ctr_id`, `alias` FROM `aux_containers` WHERE `aux_id` = ?", id)
	var auxDepCtr model.AuxDepContainer
	err := row.Scan(&auxDepCtr.ID, &auxDepCtr.Alias)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.AuxDepContainer{}, model.NewNotFoundError(err)
		}
		return model.AuxDepContainer{}, model.NewInternalError(err)
	}
	return auxDepCtr, nil
}

func (h *Handler) insertAuxDepLabels(ctx context.Context, tx *sql.Tx, id string, m map[string]string) error {
	return insertStrMap(ctx, tx, "INSERT INTO `aux_labels` (`aux_id`, `name`, `value`) VALUES (?, ?, ?)", id, m)
}

func (h *Handler) insertAuxDepConfigs(ctx context.Context, tx *sql.Tx, id string, m map[string]string) error {
	return insertStrMap(ctx, tx, "INSERT INTO `aux_configs` (`aux_id`, `ref`, `value`) VALUES (?, ?, ?)", id, m)
}

func genAuxDepFilter(dID string, filter model.AuxDepFilter) (string, []any) {
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
	if str != "" {
		str += " AND"
	}
	str += " `dep_id` = ?"
	val = append(val, dID)
	if filter.Image != "" {
		str += " AND `image` = ?"
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
