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
	"maps"
	"slices"

	lib_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/lib/models/aux_deployments"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/aux_deployments"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) GetDeployment(
	ctx context.Context,
	deploymentId string,
	auxDeploymentId string,
) (lib_aux_deployments.AuxiliaryDeployment, error) {
	mu := h.mutexes.Get(deploymentId)
	mu.RLock()
	defer mu.RUnlock()
	auxDeployments, err := h.GetDeployments(ctx, deploymentId, lib_aux_deployments.AuxiliaryDeploymentsFilterWithState{
		AuxiliaryDeploymentsFilter: lib_aux_deployments.AuxiliaryDeploymentsFilter{
			Ids: []string{auxDeploymentId},
		},
	})
	if err != nil {
		return lib_aux_deployments.AuxiliaryDeployment{}, err
	}
	if len(auxDeployments) == 0 {
		return lib_aux_deployments.AuxiliaryDeployment{}, models_error.NotFoundErr
	}
	return auxDeployments[auxDeploymentId], nil
}

func (h *Handler) GetDeployments(
	ctx context.Context,
	deploymentId string,
	filter lib_aux_deployments.AuxiliaryDeploymentsFilterWithState,
) (map[string]lib_aux_deployments.AuxiliaryDeployment, error) {
	mu := h.mutexes.Get(deploymentId)
	mu.RLock()
	defer mu.RUnlock()
	dbAuxDeployments, err := h.databaseHandler.ReadAuxiliaryDeployments(ctx, deploymentId, filter.AuxiliaryDeploymentsFilter)
	if err != nil {
		return nil, err
	}
	auxDepIds := slices.Collect(maps.Keys(dbAuxDeployments))
	dbAuxDepLabels, err := h.databaseHandler.ReadAuxiliaryDeploymentsLabels(ctx, auxDepIds)
	if err != nil {
		return nil, err
	}
	dbAuxDepConfigs, err := h.databaseHandler.ReadAuxiliaryDeploymentsConfigs(ctx, auxDepIds)
	if err != nil {
		return nil, err
	}
	dbAuxDepVolumeMounts, err := h.databaseHandler.ReadAuxiliaryDeploymentsVolumeMounts(ctx, auxDepIds)
	if err != nil {
		return nil, err
	}
	cewContainers, err := h.getCewContainers(ctx, dbAuxDeployments)
	if err != nil {
		return nil, err
	}
	return getAuxiliaryDeployments(dbAuxDeployments, dbAuxDepLabels, dbAuxDepConfigs, dbAuxDepVolumeMounts, cewContainers), nil
}

func (h *Handler) GetReducedDeployments(
	ctx context.Context,
	deploymentId string,
	filter lib_aux_deployments.AuxiliaryDeploymentsFilterWithState,
) (map[string]lib_aux_deployments.AuxiliaryDeploymentReduced, error) {
	mu := h.mutexes.Get(deploymentId)
	mu.RLock()
	defer mu.RUnlock()
	dbAuxDeployments, err := h.databaseHandler.ReadAuxiliaryDeployments(ctx, deploymentId, filter.AuxiliaryDeploymentsFilter)
	if err != nil {
		return nil, err
	}
	cewContainers, err := h.getCewContainers(ctx, dbAuxDeployments)
	if err != nil {
		return nil, err
	}
	return getReducedAuxiliaryDeployments(dbAuxDeployments, cewContainers), nil
}

func (h *Handler) getCewContainers(
	ctx context.Context,
	auxDeployments map[string]aux_deployments.AuxiliaryDeployment,
) (map[string]models_external.Container, error) {
	cewContainers, err := h.containerEngineWrapperClient.GetContainers(ctx, models_external.ContainersFilter{
		Names: helper_slices.CollectFunc(maps.Values(auxDeployments), func(item aux_deployments.AuxiliaryDeployment) string {
			return item.Container.Name
		}),
	})
	if err != nil {
		return nil, err
	}
	cewContainersMap := maps.Collect(helper_slices.AllFunc(cewContainers, func(item models_external.Container) string {
		return item.Name
	}))
	return cewContainersMap, nil
}

func getAuxiliaryDeployments(
	dbAuxDeployments map[string]aux_deployments.AuxiliaryDeployment,
	dbAuxDepLabels map[string]map[string]string,
	dbAuxDepConfigs map[string]map[string]string,
	dbAuxDepVolumeMounts map[string][]aux_deployments.AuxiliaryDeploymentVolumeMount,
	cewContainers map[string]models_external.Container,
) map[string]lib_aux_deployments.AuxiliaryDeployment {
	auxDeployments := make(map[string]lib_aux_deployments.AuxiliaryDeployment)
	for id, dbAuxDep := range dbAuxDeployments {
		cewContainer := cewContainers[dbAuxDep.Container.Name]
		auxDeployments[id] = lib_aux_deployments.AuxiliaryDeployment{
			AuxiliaryDeploymentBase: newAuxiliaryDeploymentBase(dbAuxDep),
			Labels:                  dbAuxDepLabels[id],
			Configs:                 dbAuxDepConfigs[id],
			Volumes:                 getVolumes(dbAuxDepVolumeMounts[id]),
			Container:               getContainer(dbAuxDep.Container, cewContainer),
		}
	}
	return auxDeployments
}

func getReducedAuxiliaryDeployments(
	dbAuxDeployments map[string]aux_deployments.AuxiliaryDeployment,
	cewContainers map[string]models_external.Container,
) map[string]lib_aux_deployments.AuxiliaryDeploymentReduced {
	auxDeployments := make(map[string]lib_aux_deployments.AuxiliaryDeploymentReduced)
	for id, dbAuxDep := range dbAuxDeployments {
		cewContainer := cewContainers[dbAuxDep.Container.Name]
		auxDeployments[id] = lib_aux_deployments.AuxiliaryDeploymentReduced{
			AuxiliaryDeploymentBase: newAuxiliaryDeploymentBase(dbAuxDep),
			Container:               getContainer(dbAuxDep.Container, cewContainer),
		}
	}
	return auxDeployments
}

func getVolumes(mounts []aux_deployments.AuxiliaryDeploymentVolumeMount) []lib_aux_deployments.Volume {
	var volumes []lib_aux_deployments.Volume
	for _, mount := range mounts {
		volumes = append(volumes, lib_aux_deployments.Volume{
			Reference: mount.Reference,
			MountPath: mount.MountPath,
		})
	}
	return volumes
}

func getContainer(dbContainer aux_deployments.AuxiliaryDeploymentContainer, cewContainer models_external.Container) lib_aux_deployments.Container {
	ctrInfo := lib_aux_deployments.Container{
		Name:    dbContainer.Name,
		Alias:   dbContainer.Alias,
		ImageId: cewContainer.ImageID,
		State:   cewContainer.State,
	}
	if cewContainer.Health != nil {
		ctrInfo.Health = *cewContainer.Health
	}
	return ctrInfo
}

func newAuxiliaryDeploymentBase(dbAuxDep aux_deployments.AuxiliaryDeployment) lib_aux_deployments.AuxiliaryDeploymentBase {
	return lib_aux_deployments.AuxiliaryDeploymentBase{
		Id:           dbAuxDep.Id,
		DeploymentId: dbAuxDep.DeploymentId,
		Reference:    dbAuxDep.Reference,
		Name:         dbAuxDep.Name,
		Image:        dbAuxDep.Image,
		Created:      dbAuxDep.Created,
		Updated:      dbAuxDep.Updated,
		Enabled:      dbAuxDep.Enabled,
		Recreate:     dbAuxDep.Recreate,
		RunConfig:    dbAuxDep.RunConfig,
	}
}
