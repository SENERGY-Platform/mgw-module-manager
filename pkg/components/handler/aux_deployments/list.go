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

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/aux_deployments"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

func (h *Handler) GetAuxiliaryDeployment(
	ctx context.Context,
	deploymentId string,
	auxDeploymentId string,
) (models_handler_aux_deployments.AuxiliaryDeployment, error) {
	auxDeployments, err := h.GetAuxiliaryDeployments(ctx, deploymentId, models_handler_aux_deployments.AuxiliaryDeploymentsFilter{
		AuxiliaryDeploymentsFilter: models_handler_database.AuxiliaryDeploymentsFilter{
			Ids: []string{auxDeploymentId},
		},
	})
	if err != nil {
		return models_handler_aux_deployments.AuxiliaryDeployment{}, err
	}
	if len(auxDeployments) == 0 {
		return models_handler_aux_deployments.AuxiliaryDeployment{}, models_error.NotFoundErr
	}
	return auxDeployments[auxDeploymentId], nil
}

func (h *Handler) GetAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models_handler_aux_deployments.AuxiliaryDeploymentsFilter,
) (map[string]models_handler_aux_deployments.AuxiliaryDeployment, error) {
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

func (h *Handler) GetReducedAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models_handler_aux_deployments.AuxiliaryDeploymentsFilter,
) (map[string]models_handler_aux_deployments.AuxiliaryDeploymentReduced, error) {
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
	auxDeployments map[string]models_handler_database.AuxiliaryDeployment,
) (map[string]models_external.Container, error) {
	cewContainers, err := h.containerEngineWrapperClient.GetContainers(ctx, models_external.ContainersFilter{
		Names: helper_slices.CollectFunc(maps.Values(auxDeployments), func(item models_handler_database.AuxiliaryDeployment) string {
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
	dbAuxDeployments map[string]models_handler_database.AuxiliaryDeployment,
	dbAuxDepLabels map[string]map[string]string,
	dbAuxDepConfigs map[string]map[string]string,
	dbAuxDepVolumeMounts map[string][]models_handler_database.AuxiliaryDeploymentVolumeMount,
	cewContainers map[string]models_external.Container,
) map[string]models_handler_aux_deployments.AuxiliaryDeployment {
	auxDeployments := make(map[string]models_handler_aux_deployments.AuxiliaryDeployment)
	for id, dbAuxDep := range dbAuxDeployments {
		cewContainer := cewContainers[dbAuxDep.Container.Name]
		auxDeployments[id] = models_handler_aux_deployments.AuxiliaryDeployment{
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
	dbAuxDeployments map[string]models_handler_database.AuxiliaryDeployment,
	cewContainers map[string]models_external.Container,
) map[string]models_handler_aux_deployments.AuxiliaryDeploymentReduced {
	auxDeployments := make(map[string]models_handler_aux_deployments.AuxiliaryDeploymentReduced)
	for id, dbAuxDep := range dbAuxDeployments {
		cewContainer := cewContainers[dbAuxDep.Container.Name]
		auxDeployments[id] = models_handler_aux_deployments.AuxiliaryDeploymentReduced{
			AuxiliaryDeploymentBase: newAuxiliaryDeploymentBase(dbAuxDep),
			Container:               getContainer(dbAuxDep.Container, cewContainer),
		}
	}
	return auxDeployments
}

func getVolumes(mounts []models_handler_database.AuxiliaryDeploymentVolumeMount) []models_handler_aux_deployments.Volume {
	var volumes []models_handler_aux_deployments.Volume
	for _, mount := range mounts {
		volumes = append(volumes, models_handler_aux_deployments.Volume{
			Reference: mount.Reference,
			MountPath: mount.MountPath,
		})
	}
	return volumes
}

func getContainer(dbContainer models_handler_database.AuxiliaryDeploymentContainer, cewContainer models_external.Container) models_handler_aux_deployments.Container {
	ctrInfo := models_handler_aux_deployments.Container{
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

func newAuxiliaryDeploymentBase(dbAuxDep models_handler_database.AuxiliaryDeployment) models_handler_aux_deployments.AuxiliaryDeploymentBase {
	return models_handler_aux_deployments.AuxiliaryDeploymentBase{
		Id:           dbAuxDep.Id,
		DeploymentId: dbAuxDep.DeploymentId,
		Reference:    dbAuxDep.Reference,
		Name:         dbAuxDep.Name,
		Image:        dbAuxDep.Image,
		Created:      dbAuxDep.Created,
		Updated:      dbAuxDep.Updated,
		Enabled:      dbAuxDep.Enabled,
		RunConfig:    dbAuxDep.RunConfig,
	}
}
