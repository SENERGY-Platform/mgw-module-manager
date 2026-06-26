/*
 * Copyright 2026 InfAI (CC SES)
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

package database

import (
	"context"
	"database/sql"
	"encoding/json"

	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

func (h *Handler) UpdateAuxiliaryDeployment(
	ctx context.Context,
	auxiliaryDeployment pkg_models.AuxiliaryDeployment,
	labels map[string]string,
	configs map[string]string,
	volumeMounts []pkg_models.AuxiliaryDeploymentVolumeMount,
) error {
	tx, err := h.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var runConfigCmd string
	if len(auxiliaryDeployment.RunConfig.Command) > 0 {
		b, err := json.Marshal(auxiliaryDeployment.RunConfig.Command)
		if err != nil {
			return err
		}
		runConfigCmd = string(b)
	}
	_, err = tx.ExecContext(
		ctx,
		"UPDATE aux_deployments SET image = ?, updated = ?, ref = ?, name = ?, ctr_name = ?, ctr_alias = ?, recreate = ?, command = ?, pseudo_tty = ? WHERE dep_id = ? AND id = ?",
		auxiliaryDeployment.Image,
		auxiliaryDeployment.Updated,
		auxiliaryDeployment.Reference,
		auxiliaryDeployment.Name,
		auxiliaryDeployment.Container.Name,
		auxiliaryDeployment.Container.Alias,
		auxiliaryDeployment.Recreate,
		runConfigCmd,
		auxiliaryDeployment.RunConfig.PseudoTTY,
		auxiliaryDeployment.DeploymentId,
		auxiliaryDeployment.Id,
	)
	if err != nil {
		return err
	}
	err = h.deleteAuxiliaryDeploymentAssets(ctx, tx, auxiliaryDeployment.Id)
	if err != nil {
		return err
	}
	err = h.createAuxiliaryDeploymentAssets(ctx, tx, auxiliaryDeployment.Id, labels, configs, volumeMounts)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) UpdateAuxiliaryDeploymentContainerName(ctx context.Context, auxDeploymentId, name string) error {
	_, err := h.sqlDB.ExecContext(
		ctx,
		"UPDATE aux_deployments SET ctr_name = ? WHERE id = ?",
		name,
		auxDeploymentId,
	)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) UpdateAuxiliaryDeploymentsEnabledState(ctx context.Context, auxDeploymentIds []string, state bool) error {
	var db sqlDatabase = h.sqlDB
	var tx *sql.Tx
	var err error
	if len(auxDeploymentIds) > 0 {
		tx, err = h.sqlDB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		db = tx
	}
	for _, id := range auxDeploymentIds {
		_, err = db.ExecContext(
			ctx,
			"UPDATE aux_deployments SET enabled = ? WHERE id = ?",
			state,
			id,
		)
		if err != nil {
			return err
		}
	}
	if tx != nil {
		err = tx.Commit()
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) deleteAuxiliaryDeploymentAssets(ctx context.Context, tx *sql.Tx, auxDeploymentId string) error {
	_, err := tx.ExecContext(
		ctx,
		"DELETE FROM aux_dep_labels WHERE aux_dep_id = ?;",
		auxDeploymentId,
	)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		"DELETE FROM aux_dep_configs WHERE aux_dep_id = ?;",
		auxDeploymentId,
	)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		"DELETE FROM aux_dep_volume_mounts WHERE aux_dep_id = ?;",
		auxDeploymentId,
	)
	if err != nil {
		return err
	}
	return nil
}
