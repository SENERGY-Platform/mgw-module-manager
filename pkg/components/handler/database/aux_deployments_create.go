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

package handler_database

import (
	"context"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

func (h *Handler) CreateAuxiliaryDeployment(
	ctx context.Context,
	auxiliaryDeployment models_handler_database.AuxiliaryDeployment,
	labels map[string]string,
	configs map[string]string,
	volumeMounts []models_handler_database.AuxiliaryDeploymentVolumeMount,
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
		auxiliaryDeployment.RunConfig.Command,
		auxiliaryDeployment.RunConfig.PseudoTTY,
	)
	if err != nil {
		return err
	}
	for name, value := range labels {
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO aux_dep_labels (aux_dep_id, name, value) VALUES (?, ?, ?)",
			auxiliaryDeployment.Id,
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
			auxiliaryDeployment.Id,
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
			auxiliaryDeployment.Id,
			mount.MountPath,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (h *Handler) CreateAuxiliaryDeploymentVolumes(
	ctx context.Context,
	deploymentId string,
	volumes []models_handler_database.AuxiliaryDeploymentVolume,
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
