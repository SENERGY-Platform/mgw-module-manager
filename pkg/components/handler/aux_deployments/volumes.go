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

package aux_deployments

import (
	"context"
	"errors"
	"maps"
	"strings"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) GetVolumes(
	ctx context.Context,
	deploymentId string,
	filterReferences []string,
) (map[string]lib_models.AuxiliaryDeploymentVolume, error) {
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
) (map[string]lib_models.AuxiliaryDeploymentVolumeWithMounts, error) {
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
) ([]lib_models.AuxiliaryDeploymentVolumeResult, error) {
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
) ([]lib_models.AuxiliaryDeploymentVolumeResult, error) {
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
	volumes := make(map[string]lib_models.AuxiliaryDeploymentVolume)
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
	volumes map[string]lib_models.AuxiliaryDeploymentVolume,
) ([]lib_models.AuxiliaryDeploymentVolumeResult, error) {
	var deleted []string
	var results []lib_models.AuxiliaryDeploymentVolumeResult
	for reference, volume := range volumes {
		result := lib_models.AuxiliaryDeploymentVolumeResult{Reference: reference}
		err := h.containerEngineWrapperClient.RemoveVolume(ctx, volume.Name, false)
		if err != nil {
			result.ErrorResult = lib_models.NewErrorResult(err.Error())
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
	volumes map[string]lib_models.AuxiliaryDeploymentVolume,
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

func (h *Handler) createContainerVolume(ctx context.Context, volume lib_models.AuxiliaryDeploymentVolume) error {
	_, err := h.containerEngineWrapperClient.CreateVolume(ctx, external_models.CewVolume{
		Name: volume.Name,
		Labels: map[string]string{
			constants.LabelCoreId:                helper_naming.CoreId,
			constants.LabelManagerId:             helper_naming.ManagerId,
			constants.LabelVolumeType:            constants.AuxDeploymentAbbreviation,
			constants.LabelDeploymentId:          volume.DeploymentId,
			constants.LabelVolumeReference:       volume.Reference,
			constants.LabelAuxDeploymentVolumeId: volume.Id,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) removeContainerVolumes(
	ctx context.Context,
	deploymentVolumes map[string]lib_models.AuxiliaryDeploymentVolume,
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
		var notFoundErr *external_models.CewNotFoundErr
		if !errors.As(err, &notFoundErr) {
			return err
		}
	}
	return nil
}

func (h *Handler) getContainerVolumes(ctx context.Context, deploymentId string) (map[string]external_models.CewVolume, error) {
	volumes, err := h.containerEngineWrapperClient.GetVolumes(ctx, external_models.CewVolumesFilter{
		Labels: map[string]string{
			constants.LabelCoreId:       helper_naming.CoreId,
			constants.LabelManagerId:    helper_naming.ManagerId,
			constants.LabelVolumeType:   constants.AuxDeploymentAbbreviation,
			constants.LabelDeploymentId: deploymentId,
		},
	})
	if err != nil {
		return nil, err
	}
	volumesMap := maps.Collect(helper_slices.AllFunc(volumes, func(item external_models.CewVolume) string {
		return item.Name
	}))
	return volumesMap, nil
}

func getNewVolumes(
	deploymentId string,
	serviceInputVolumes map[string]string, // {mntPath:reference}
	auxiliaryDeploymentVolumes map[string]lib_models.AuxiliaryDeploymentVolume,
) map[string]lib_models.AuxiliaryDeploymentVolume {
	volumes := make(map[string]lib_models.AuxiliaryDeploymentVolume)
	for _, reference := range serviceInputVolumes {
		_, ok := auxiliaryDeploymentVolumes[reference]
		if ok {
			continue
		}
		volume := lib_models.AuxiliaryDeploymentVolume{
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
	auxiliaryDeploymentVolumes map[string]lib_models.AuxiliaryDeploymentVolume,
) []pkg_models.AuxiliaryDeploymentVolumeMount {
	var volumeMounts []pkg_models.AuxiliaryDeploymentVolumeMount
	for mountPath, reference := range serviceInputVolumes {
		volume := auxiliaryDeploymentVolumes[reference]
		volumeMounts = append(volumeMounts, pkg_models.AuxiliaryDeploymentVolumeMount{
			VolumeId:              volume.Id,
			VolumeName:            volume.Name,
			Reference:             volume.Reference,
			AuxiliaryDeploymentId: auxDeploymentId,
			MountPath:             mountPath,
		})
	}
	return volumeMounts
}
