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

	lib_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/lib/models/aux_deployments"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/lib/models/results"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/aux_deployments"
	models_constants "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) GetVolumes(
	ctx context.Context,
	deploymentId string,
	filterReferences []string,
) (map[string]aux_deployments.AuxiliaryDeploymentVolume, error) {
	mu := h.mutexes.Get(deploymentId)
	mu.RLock()
	defer mu.RUnlock()
	volumes, err := h.databaseHandler.ReadAuxiliaryDeploymentVolumes(ctx, deploymentId, filterReferences)
	if err != nil {
		return nil, err
	}
	return volumes, nil
}

func (h *Handler) GetVolumesWithMounts(
	ctx context.Context,
	deploymentId string,
	filterReferences []string,
) (map[string]aux_deployments.AuxiliaryDeploymentVolumeWithMounts, error) {
	mu := h.mutexes.Get(deploymentId)
	mu.RLock()
	defer mu.RUnlock()
	volumes, err := h.databaseHandler.ReadAuxiliaryDeploymentVolumesWithMounts(ctx, deploymentId, filterReferences)
	if err != nil {
		return nil, err
	}
	return volumes, nil
}

func (h *Handler) DeleteVolumes(
	ctx context.Context,
	deploymentId string,
	filterReferences []string,
	allowAll bool,
) ([]lib_aux_deployments.VolumeResult, error) {
	if !allowAll && len(filterReferences) == 0 {
		return nil, nil
	}
	mu := h.mutexes.Get(deploymentId)
	mu.Lock()
	defer mu.Unlock()
	volumes, err := h.databaseHandler.ReadAuxiliaryDeploymentVolumes(ctx, deploymentId, filterReferences)
	if err != nil {
		return nil, err
	}
	return h.deleteVolumes(ctx, deploymentId, volumes)
}

func (h *Handler) DeleteUnusedVolumes(
	ctx context.Context,
	deploymentId string,
	excludeReferences []string,
) ([]lib_aux_deployments.VolumeResult, error) {
	mu := h.mutexes.Get(deploymentId)
	mu.Lock()
	defer mu.Unlock()
	volumesWithMounts, err := h.databaseHandler.ReadAuxiliaryDeploymentVolumesWithMounts(ctx, deploymentId, nil)
	if err != nil {
		return nil, err
	}
	for _, reference := range excludeReferences {
		delete(volumesWithMounts, reference)
	}
	volumes := make(map[string]aux_deployments.AuxiliaryDeploymentVolume)
	for reference, volume := range volumesWithMounts {
		if len(volume.MountedBy) == 0 {
			volumes[reference] = volume.AuxiliaryDeploymentVolume
		}
	}
	return h.deleteVolumes(ctx, deploymentId, volumes)
}

func (h *Handler) deleteVolumes(
	ctx context.Context,
	deploymentId string,
	volumes map[string]aux_deployments.AuxiliaryDeploymentVolume,
) ([]lib_aux_deployments.VolumeResult, error) {
	var deleted []string
	var results []lib_aux_deployments.VolumeResult
	for reference, volume := range volumes {
		result := lib_aux_deployments.VolumeResult{Reference: reference}
		err := h.containerEngineWrapperClient.RemoveVolume(ctx, volume.Name, false)
		if err != nil {
			result.ErrorResult = models_error.NewErrorResult(err.Error())
		} else {
			deleted = append(deleted, reference)
		}
		results = append(results, result)
	}
	err := h.databaseHandler.DeleteAuxiliaryDeploymentVolumes(ctx, deploymentId, deleted)
	if err != nil {
		return results, err
	}
	return results, nil
}

func (h *Handler) ensureContainerVolumes(
	ctx context.Context,
	volumes map[string]aux_deployments.AuxiliaryDeploymentVolume,
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

func (h *Handler) createContainerVolume(ctx context.Context, volume aux_deployments.AuxiliaryDeploymentVolume) error {
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
	deploymentVolumes map[string]aux_deployments.AuxiliaryDeploymentVolume,
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
	auxiliaryDeploymentVolumes map[string]aux_deployments.AuxiliaryDeploymentVolume,
) map[string]aux_deployments.AuxiliaryDeploymentVolume {
	volumes := make(map[string]aux_deployments.AuxiliaryDeploymentVolume)
	for _, reference := range serviceInputVolumes {
		_, ok := auxiliaryDeploymentVolumes[reference]
		if ok {
			continue
		}
		volume := aux_deployments.AuxiliaryDeploymentVolume{
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
	auxiliaryDeploymentVolumes map[string]aux_deployments.AuxiliaryDeploymentVolume,
) []aux_deployments.AuxiliaryDeploymentVolumeMount {
	var volumeMounts []aux_deployments.AuxiliaryDeploymentVolumeMount
	for mountPath, reference := range serviceInputVolumes {
		volume := auxiliaryDeploymentVolumes[reference]
		volumeMounts = append(volumeMounts, aux_deployments.AuxiliaryDeploymentVolumeMount{
			VolumeId:              volume.Id,
			VolumeName:            volume.Name,
			Reference:             volume.Reference,
			AuxiliaryDeploymentId: auxDeploymentId,
			MountPath:             mountPath,
		})
	}
	return volumeMounts
}
