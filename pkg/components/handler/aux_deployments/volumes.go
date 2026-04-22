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

package handler_aux_deployments

import (
	"context"
	"errors"
	"maps"
	"strings"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

func (h *Handler) GetVolumes(
	ctx context.Context,
	deploymentId string,
) (map[string]models_handler_database.AuxiliaryDeploymentVolume, error) {
	mu := h.mutexes.Get(deploymentId)
	mu.RLock()
	defer mu.RUnlock()
	volumes, err := h.databaseHandler.ReadAuxiliaryDeploymentVolumes(ctx, deploymentId)
	if err != nil {
		return nil, err
	}
	return volumes, nil
}

func (h *Handler) GetVolumesWithMounts(
	ctx context.Context,
	deploymentId string,
) (map[string]models_handler_database.AuxiliaryDeploymentVolumeWithMounts, error) {
	mu := h.mutexes.Get(deploymentId)
	mu.RLock()
	defer mu.RUnlock()
	volumes, err := h.databaseHandler.ReadAuxiliaryDeploymentVolumesWithMounts(ctx, deploymentId)
	if err != nil {
		return nil, err
	}
	return volumes, nil
}

func (h *Handler) ensureContainerVolumes(
	ctx context.Context,
	volumes map[string]models_handler_database.AuxiliaryDeploymentVolume,
	deploymentId string,
) error {
	existingVolumes, err := h.getContainerVolumes(ctx, deploymentId)
	if err != nil {
		return err
	}
	var errs []string
	for _, volume := range volumes {
		_, ok := existingVolumes[volume.Name]
		if !ok {
			err = h.createContainerVolume(ctx, volume)
			if err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func (h *Handler) createContainerVolume(ctx context.Context, volume models_handler_database.AuxiliaryDeploymentVolume) error {
	_, err := h.containerEngineWrapperClient.CreateVolume(ctx, models_external.Volume{
		Name: volume.Name,
		Labels: map[string]string{
			models_constants.LabelCoreId:                helper_naming.CoreId,
			models_constants.LabelManagerId:             helper_naming.ManagerId,
			models_constants.LabelVolumeType:            models_constants.AuxDeploymentAbbreviation,
			models_constants.LabelDeploymentId:          volume.DeploymentId,
			models_constants.LabelVolumeReference:       volume.Reference,
			models_constants.LabelAuxDeploymentVolumeId: volume.Id,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) removeContainerVolumes(
	ctx context.Context,
	deploymentVolumes map[string]models_handler_database.AuxiliaryDeploymentVolume,
) error {
	var errs []string
	for _, volume := range deploymentVolumes {
		err := h.removeContainerVolume(ctx, volume.Name)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (h *Handler) removeContainerVolume(ctx context.Context, name string) error {
	err := h.containerEngineWrapperClient.RemoveVolume(ctx, name, false)
	if err != nil {
		var notFoundErr *models_external.CEWNotFoundErr
		if !errors.As(err, &notFoundErr) {
			return err
		}
	}
	return nil
}

func (h *Handler) getContainerVolumes(ctx context.Context, deploymentId string) (map[string]models_external.Volume, error) {
	volumes, err := h.containerEngineWrapperClient.GetVolumes(ctx, models_external.VolumesFilter{
		Labels: map[string]string{
			models_constants.LabelCoreId:       helper_naming.CoreId,
			models_constants.LabelManagerId:    helper_naming.ManagerId,
			models_constants.LabelVolumeType:   models_constants.AuxDeploymentAbbreviation,
			models_constants.LabelDeploymentId: deploymentId,
		},
	})
	if err != nil {
		return nil, err
	}
	volumesMap := maps.Collect(helper_slices.AllFunc(volumes, func(item models_external.Volume) string {
		return item.Name
	}))
	return volumesMap, nil
}

func getNewVolumes(
	deploymentId string,
	serviceInputVolumes map[string]string, // {mntPath:reference}
	auxiliaryDeploymentVolumes map[string]models_handler_database.AuxiliaryDeploymentVolume,
) map[string]models_handler_database.AuxiliaryDeploymentVolume {
	volumes := make(map[string]models_handler_database.AuxiliaryDeploymentVolume)
	for _, reference := range serviceInputVolumes {
		_, ok := auxiliaryDeploymentVolumes[reference]
		if ok {
			continue
		}
		volume := models_handler_database.AuxiliaryDeploymentVolume{
			Id:           deploymentId + "_" + reference,
			DeploymentId: deploymentId,
			Reference:    reference,
			Name:         helper_naming.NewVolumeName("aux_dep", deploymentId, reference),
		}
		volumes[reference] = volume
	}
	return volumes
}

func getVolumeMounts(
	auxDeploymentId string,
	serviceInputVolumes map[string]string, // {mntPath:reference}
	auxiliaryDeploymentVolumes map[string]models_handler_database.AuxiliaryDeploymentVolume,
) []models_handler_database.AuxiliaryDeploymentVolumeMount {
	var volumeMounts []models_handler_database.AuxiliaryDeploymentVolumeMount
	for mountPath, reference := range serviceInputVolumes {
		volume := auxiliaryDeploymentVolumes[reference]
		volumeMounts = append(volumeMounts, models_handler_database.AuxiliaryDeploymentVolumeMount{
			VolumeId:              volume.Id,
			VolumeName:            volume.Name,
			Reference:             volume.Reference,
			AuxiliaryDeploymentId: auxDeploymentId,
			MountPath:             mountPath,
		})
	}
	return volumeMounts
}
