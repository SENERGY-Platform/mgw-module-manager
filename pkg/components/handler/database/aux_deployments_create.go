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
	"strings"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

func (h *Handler) CreateAuxiliaryDeployment(
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
	_, err = tx.ExecContext(
		ctx,
		"INSERT INTO aux_deployments (id, dep_id, image, created, updated, ref, name, enabled, ctr_name, ctr_alias, command, pseudo_tty) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		auxiliaryDeployment.Id,
		auxiliaryDeployment.DeploymentId,
		auxiliaryDeployment.Image,
		auxiliaryDeployment.Created,
		auxiliaryDeployment.Updated,
		auxiliaryDeployment.Reference,
		auxiliaryDeployment.Name,
		auxiliaryDeployment.Enabled,
		auxiliaryDeployment.Container.Name,
		auxiliaryDeployment.Container.Alias,
		strings.Join(auxiliaryDeployment.RunConfig.Command, ","),
		auxiliaryDeployment.RunConfig.PseudoTTY,
	)
	if err != nil {
		return err
	}
	err = h.createAuxiliaryDeploymentAssets(ctx, tx, auxiliaryDeployment.Id, labels, configs, volumeMounts)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (h *Handler) CreateAuxiliaryDeploymentVolumes(
	ctx context.Context,
	deploymentId string,
	volumes []lib_models.AuxiliaryDeploymentVolume,
) error {
	tx, err := h.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, volume := range volumes {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO aux_dep_volumes (id, dep_id, ref, name) VALUES (?, ?, ?, ?)",
			volume.Id,
			deploymentId,
			volume.Reference,
			volume.Name,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (h *Handler) createAuxiliaryDeploymentAssets(
	ctx context.Context,
	tx *sql.Tx,
	auxDeploymentId string,
	labels map[string]string,
	configs map[string]string,
	volumeMounts []pkg_models.AuxiliaryDeploymentVolumeMount,
) error {
	var err error
	for name, value := range labels {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO aux_dep_labels (aux_dep_id, name, value) VALUES (?, ?, ?)",
			auxDeploymentId,
			name,
			value,
		)
		if err != nil {
			return err
		}
	}
	for varName, value := range configs {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO aux_dep_configs (aux_dep_id, name, value) VALUES (?, ?, ?)",
			auxDeploymentId,
			varName,
			value,
		)
		if err != nil {
			return err
		}
	}
	for _, mount := range volumeMounts {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO aux_dep_volume_mounts (vol_id, aux_dep_id, mnt_path) VALUES (?, ?, ?)",
			mount.VolumeId,
			auxDeploymentId,
			mount.MountPath,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
