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

package deployments

import (
	"context"
	"fmt"
	"maps"

	helper_containers "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/containers"
	helper_errors "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/errors"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) ensureContainerVolumes(ctx context.Context,
	volumes map[string]pkg_models.DeploymentVolume,
	deploymentId string,
) error {
	existingVolumes, err := h.getContainerVolumes(ctx, deploymentId)
	if err != nil {
		return err
	}
	var errs []error
	for name := range existingVolumes {
		_, ok := volumes[name]
		if !ok {
			err = helper_containers.RemoveVolume(ctx, h.containerEngineWrapperClient, name)
			if err != nil {
				errs = append(errs, fmt.Errorf("'%s' %w", name, err))
			}
		}
	}
	for _, volume := range volumes {
		_, ok := existingVolumes[volume.Name]
		if !ok {
			err = h.createContainerVolume(ctx, volume)
			if err != nil {
				errs = append(errs, fmt.Errorf("'%s' %w", volume.Reference, err))
			}
		}
	}
	if len(errs) > 0 {
		return helper_errors.Join(errs...)
	}
	return nil
}

func (h *Handler) createContainerVolume(ctx context.Context, volume pkg_models.DeploymentVolume) error {
	_, err := h.containerEngineWrapperClient.CreateVolume(ctx, external_models.CewVolume{
		Name: volume.Name,
		Labels: map[string]string{
			constants.LabelCoreId:          helper_naming.CoreId,
			constants.LabelManagerId:       helper_naming.ManagerId,
			constants.LabelVolumeType:      constants.DeploymentAbbreviation,
			constants.LabelDeploymentId:    volume.DeploymentId,
			constants.LabelVolumeReference: volume.Reference,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) removeContainerVolumes(
	ctx context.Context,
	deploymentVolumes map[string]pkg_models.DeploymentVolume,
) error {
	var errs []error
	for _, volume := range deploymentVolumes {
		err := helper_containers.RemoveVolume(ctx, h.containerEngineWrapperClient, volume.Name)
		if err != nil {
			errs = append(errs, fmt.Errorf("'%s' %w", volume.Name, err))
		}
	}
	if len(errs) > 0 {
		return helper_errors.Join(errs...)
	}
	return nil
}

func (h *Handler) getContainerVolumes(ctx context.Context, deploymentId string) (map[string]external_models.CewVolume, error) {
	volumes, err := h.containerEngineWrapperClient.GetVolumes(ctx, external_models.CewVolumesFilter{
		Labels: map[string]string{
			constants.LabelCoreId:       helper_naming.CoreId,
			constants.LabelManagerId:    helper_naming.ManagerId,
			constants.LabelVolumeType:   constants.DeploymentAbbreviation,
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
