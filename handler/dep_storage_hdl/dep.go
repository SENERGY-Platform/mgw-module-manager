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
	"github.com/SENERGY-Platform/mgw-module-manager/handler/dep_storage_hdl/dep_util"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"strings"
	"time"
)

func (h *Handler) ListDep(ctx context.Context, filter model.DepFilter, dependencyInfo, assets, containers bool) (map[string]model.Deployment, error) {
	q := "SELECT `id`, `mod_id`, `mod_ver`, `name`, `dir`, `enabled`, `indirect`, `created`, `updated` FROM `deployments`"
	fc, val := genDepFilter(filter)
	if fc != "" {
		q += fc
	}
	rows, err := h.db.QueryContext(ctx, q, val...)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	defer rows.Close()
	deployments := make(map[string]model.Deployment)
	for rows.Next() {
		var deployment model.Deployment
		var depModule model.DepModule
		var ct, ut []uint8
		if err = rows.Scan(&deployment.ID, &depModule.ID, &depModule.Version, &deployment.Name, &deployment.Dir, &deployment.Enabled, &deployment.Indirect, &ct, &ut); err != nil {
			return nil, model.NewInternalError(err)
		}
		deployment.Module = depModule
		if deployment.Created, err = time.Parse(tLayout, string(ct)); err != nil {
			return nil, model.NewInternalError(err)
		}
		if deployment.Updated, err = time.Parse(tLayout, string(ut)); err != nil {
			return nil, model.NewInternalError(err)
		}
		if dependencyInfo {
			if deployment.RequiredDep, err = dep_util.SelectRequiredDep(ctx, h.db, deployment.ID); err != nil {
				return nil, model.NewInternalError(err)
			}
			if deployment.DepRequiring, err = dep_util.SelectDepRequiring(ctx, h.db, deployment.ID); err != nil {
				return nil, model.NewInternalError(err)
			}
		}
		if assets {
			if deployment.DepAssets, err = readDepAssets(ctx, h.db, deployment.ID); err != nil {
				return nil, model.NewInternalError(err)
			}
		}
		if containers {
			if deployment.Containers, err = dep_util.SelectDepContainers(ctx, h.db, deployment.ID); err != nil {
				return nil, model.NewInternalError(err)
			}
		}
		deployments[deployment.ID] = deployment
	}
	if err = rows.Err(); err != nil {
		return nil, model.NewInternalError(err)
	}
	return deployments, nil
}

func (h *Handler) ReadDep(ctx context.Context, dID string, dependencyInfo, assets, containers bool) (model.Deployment, error) {
	row := h.db.QueryRowContext(ctx, "SELECT `id`, `mod_id`, `mod_ver`, `name`, `dir`, `enabled`, `indirect`, `created`, `updated` FROM `deployments` WHERE `id` = ?", dID)
	var deployment model.Deployment
	var depModule model.DepModule
	var ct, ut []uint8
	err := row.Scan(&deployment.ID, &depModule.ID, &depModule.Version, &deployment.Name, &deployment.Dir, &deployment.Enabled, &deployment.Indirect, &ct, &ut)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Deployment{}, model.NewNotFoundError(err)
		}
		return model.Deployment{}, model.NewInternalError(err)
	}
	deployment.Module = depModule
	if deployment.Created, err = time.Parse(tLayout, string(ct)); err != nil {
		return model.Deployment{}, model.NewInternalError(err)
	}
	if deployment.Updated, err = time.Parse(tLayout, string(ut)); err != nil {
		return model.Deployment{}, model.NewInternalError(err)
	}
	if dependencyInfo {
		if deployment.RequiredDep, err = dep_util.SelectRequiredDep(ctx, h.db, deployment.ID); err != nil {
			return model.Deployment{}, model.NewInternalError(err)
		}
		if deployment.DepRequiring, err = dep_util.SelectDepRequiring(ctx, h.db, deployment.ID); err != nil {
			return model.Deployment{}, model.NewInternalError(err)
		}
	}
	if assets {
		if deployment.DepAssets, err = readDepAssets(ctx, h.db, deployment.ID); err != nil {
			return model.Deployment{}, model.NewInternalError(err)
		}
	}
	if containers {
		if deployment.Containers, err = dep_util.SelectDepContainers(ctx, h.db, deployment.ID); err != nil {
			return model.Deployment{}, model.NewInternalError(err)
		}
	}
	return deployment, nil
}

func (h *Handler) ReadDepTree(ctx context.Context, dID string, assets, containers bool) (map[string]model.Deployment, error) {
	rootDep, err := h.ReadDep(ctx, dID, true, assets, containers)
	if err != nil {
		return nil, err
	}
	tree := map[string]model.Deployment{rootDep.ID: rootDep}
	if err = h.appendDepTree(ctx, rootDep, tree, assets, containers); err != nil {
		return nil, err
	}
	return tree, nil
}

func (h *Handler) AppendDepTree(ctx context.Context, tree map[string]model.Deployment, assets, containers bool) error {
	for _, dep := range tree {
		if err := h.appendDepTree(ctx, dep, tree, assets, containers); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) CreateDep(ctx context.Context, txItf driver.Tx, depBase model.DepBase) (string, error) {
	execContext := h.db.ExecContext
	queryRowContext := h.db.QueryRowContext
	if txItf != nil {
		tx := txItf.(*sql.Tx)
		execContext = tx.ExecContext
		queryRowContext = tx.QueryRowContext
	}
	res, err := execContext(ctx, "INSERT INTO `deployments` (`id`, `mod_id`, `mod_ver`, `name`, `dir`, `enabled`, `indirect`, `created`, `updated`) VALUES (UUID(), ?, ?, ?, ?, ?, ?, ?, ?)", depBase.Module.ID, depBase.Module.Version, depBase.Name, depBase.Dir, depBase.Enabled, depBase.Indirect, depBase.Created, depBase.Updated)
	if err != nil {
		return "", model.NewInternalError(err)
	}
	i, err := res.LastInsertId()
	if err != nil {
		return "", model.NewInternalError(err)
	}
	row := queryRowContext(ctx, "SELECT `id` FROM `deployments` WHERE `index` = ?", i)
	var id string
	if err = row.Scan(&id); err != nil {
		return "", model.NewInternalError(err)
	}
	if id == "" {
		return "", model.NewInternalError(errors.New("generating id failed"))
	}
	return id, nil
}

func (h *Handler) UpdateDep(ctx context.Context, txItf driver.Tx, depBase model.DepBase) error {
	execContext := h.db.ExecContext
	if txItf != nil {
		tx := txItf.(*sql.Tx)
		execContext = tx.ExecContext
	}
	res, err := execContext(ctx, "UPDATE `deployments` SET `mod_ver` = ?, `name` = ?, `dir` = ?, `enabled` = ?, `indirect` = ?, `updated` = ? WHERE `id` = ?", depBase.Module.Version, depBase.Name, depBase.Dir, depBase.Enabled, depBase.Indirect, depBase.Updated, depBase.ID)
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

func (h *Handler) DeleteDep(ctx context.Context, txItf driver.Tx, dID string) error {
	execContext := h.db.ExecContext
	if txItf != nil {
		tx := txItf.(*sql.Tx)
		execContext = tx.ExecContext
	}
	res, err := execContext(ctx, "DELETE FROM `deployments` WHERE `id` = ?", dID)
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

func (h *Handler) appendDepTree(ctx context.Context, dep model.Deployment, tree map[string]model.Deployment, assets, containers bool) error {
	for _, dID := range dep.RequiredDep {
		if _, ok := tree[dID]; !ok {
			d, err := h.ReadDep(ctx, dID, true, assets, containers)
			if err != nil {
				return err
			}
			tree[dID] = d
			if err = h.appendDepTree(ctx, d, tree, assets, containers); err != nil {
				return err
			}
		}
	}
	return nil
}

func readDepAssets(ctx context.Context, db *sql.DB, id string) (model.DepAssets, error) {
	hostRes, err := dep_util.SelectHostResources(ctx, db, id)
	if err != nil {
		return model.DepAssets{}, err
	}
	secrets, err := dep_util.SelectSecrets(ctx, db, id)
	if err != nil {
		return model.DepAssets{}, err
	}
	configs := make(map[string]model.DepConfig)
	err = dep_util.SelectConfigs(ctx, db, id, configs)
	if err != nil {
		return model.DepAssets{}, err
	}
	err = dep_util.SelectListConfigs(ctx, db, id, configs)
	if err != nil {
		return model.DepAssets{}, err
	}
	return model.DepAssets{
		HostResources: hostRes,
		Secrets:       secrets,
		Configs:       configs,
	}, nil
}

func genDepFilter(filter model.DepFilter) (string, []any) {
	var fc []string
	var val []any
	if len(filter.IDs) > 0 {
		ids := removeDuplicates(filter.IDs)
		fc = append(fc, "`id` IN ("+strings.Repeat("?, ", len(ids)-1)+"?)")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if filter.ModuleID != "" {
		fc = append(fc, "`mod_id` = ?")
		val = append(val, filter.ModuleID)
	}
	if filter.Name != "" {
		fc = append(fc, "`name` = ?")
		val = append(val, filter.Name)
	}
	if filter.Enabled {
		fc = append(fc, "`enabled` = ?")
		val = append(val, filter.Enabled)
	}
	if filter.Indirect {
		fc = append(fc, "`indirect` = ?")
		val = append(val, filter.Indirect)
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}

func removeDuplicates(sl []string) []string {
	if len(sl) < 2 {
		return sl
	}
	set := make(map[string]struct{})
	var sl2 []string
	for _, s := range sl {
		if _, ok := set[s]; !ok {
			sl2 = append(sl2, s)
		}
	}
	return sl2
}
