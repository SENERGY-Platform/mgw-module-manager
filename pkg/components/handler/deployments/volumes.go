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
	"errors"
	"maps"
	"strings"

	helper_containers "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/containers"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

func (h *Handler) ensureContainerVolumes(ctx context.Context,
	volumes map[string]pkg_models.DeploymentVolume,
	deploymentId string,
) error {
	existingVolumes, err := h.getContainerVolumes(ctx, deploymentId)
	if err != nil {
		return err
	}
	var errs []string
	for name := range existingVolumes {
		_, ok := volumes[name]
		if !ok {
			err = helper_containers.RemoveVolume(ctx, h.containerEngineWrapperClient, name)
			if err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
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

func (h *Handler) createContainerVolume(ctx context.Context, volume pkg_models.DeploymentVolume) error {
	_, err := h.containerEngineWrapperClient.CreateVolume(ctx, pkg_models.Volume{
		Name: volume.Name,
		Labels: map[string]string{
			pkg_models.LabelCoreId:          helper_naming.CoreId,
			pkg_models.LabelManagerId:       helper_naming.ManagerId,
			pkg_models.LabelVolumeType:      pkg_models.DeploymentAbbreviation,
			pkg_models.LabelDeploymentId:    volume.DeploymentId,
			pkg_models.LabelVolumeReference: volume.Reference,
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
	var errs []string
	for _, volume := range deploymentVolumes {
		err := helper_containers.RemoveVolume(ctx, h.containerEngineWrapperClient, volume.Name)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (h *Handler) getContainerVolumes(ctx context.Context, deploymentId string) (map[string]pkg_models.Volume, error) {
	volumes, err := h.containerEngineWrapperClient.GetVolumes(ctx, pkg_models.VolumesFilter{
		Labels: map[string]string{
			pkg_models.LabelCoreId:       helper_naming.CoreId,
			pkg_models.LabelManagerId:    helper_naming.ManagerId,
			pkg_models.LabelVolumeType:   pkg_models.DeploymentAbbreviation,
			pkg_models.LabelDeploymentId: deploymentId,
		},
	})
	if err != nil {
		return nil, err
	}
	volumesMap := maps.Collect(helper_slices.AllFunc(volumes, func(item pkg_models.Volume) string {
		return item.Name
	}))
	return volumesMap, nil
}
