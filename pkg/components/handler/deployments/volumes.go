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

	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) ensureContainerVolumes(ctx context.Context,
	volumes map[string]models_handler_storage.DeploymentVolume,
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
			err = h.removeContainerVolume(ctx, name)
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

func (h *Handler) createContainerVolume(ctx context.Context, volume models_handler_storage.DeploymentVolume) error {
	_, err := h.cewClient.CreateVolume(ctx, models_external.Volume{
		Name: volume.Name,
		Labels: map[string]string{
			constants.LabelCoreId:          helper_naming.CoreId,
			constants.LabelManagerId:       helper_naming.ManagerId,
			constants.LabelDeploymentId:    volume.DeploymentId,
			constants.LabelVolumeReference: volume.Reference,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) removeContainerVolume(ctx context.Context, name string) error {
	err := h.cewClient.RemoveVolume(ctx, name, false)
	if err != nil {
		var notFoundErr *models_external.CEWNotFoundErr
		if !errors.As(err, &notFoundErr) {
			return err
		}
	}
	return nil
}

func (h *Handler) getContainerVolumes(ctx context.Context, deploymentId string) (map[string]models_external.Volume, error) {
	volumes, err := h.cewClient.GetVolumes(ctx, models_external.VolumesFilter{
		Labels: map[string]string{
			constants.LabelCoreId:       helper_naming.CoreId,
			constants.LabelManagerId:    helper_naming.ManagerId,
			constants.LabelDeploymentId: deploymentId,
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
