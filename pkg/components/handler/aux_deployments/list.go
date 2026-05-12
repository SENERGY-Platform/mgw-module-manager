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
	"maps"
	"slices"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) GetDeployment(
	ctx context.Context,
	deploymentId string,
	auxDeploymentId string,
) (lib_models.AuxiliaryDeployment, error) {
	mu := h.mutexes.Get(deploymentId)
	mu.RLock()
	defer mu.RUnlock()
	auxDeployments, err := h.GetDeployments(ctx, deploymentId, lib_models.AuxiliaryDeploymentsFilterWithState{
		AuxiliaryDeploymentsFilter: lib_models.AuxiliaryDeploymentsFilter{
			Ids: []string{auxDeploymentId},
		},
	})
	if err != nil {
		return lib_models.AuxiliaryDeployment{}, err
	}
	if len(auxDeployments) == 0 {
		return lib_models.AuxiliaryDeployment{}, lib_errors.New[lib_errors.ErrNotFound]("auxiliary deployment not found")
	}
	return auxDeployments[auxDeploymentId], nil
}

func (h *Handler) GetDeployments(
	ctx context.Context,
	deploymentId string,
	filter lib_models.AuxiliaryDeploymentsFilterWithState,
) (map[string]lib_models.AuxiliaryDeployment, error) {
	mu := h.mutexes.Get(deploymentId)
	mu.RLock()
	defer mu.RUnlock()
	dbAuxDeployments, err := h.databaseHandler.ReadAuxiliaryDeployments(ctx, deploymentId, filter.AuxiliaryDeploymentsFilter)
	if err != nil {
		logger.Error(
			"get auxiliary deployments, read from database",
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Filter, filter,
			slog_keys.Error, err,
		)
		return nil, err
	}
	auxDepIds := slices.Collect(maps.Keys(dbAuxDeployments))
	dbAuxDepLabels, err := h.databaseHandler.ReadAuxiliaryDeploymentsLabels(ctx, auxDepIds)
	if err != nil {
		logger.Error(
			"get auxiliary deployments, read labels from database",
			slog_keys.DeploymentId, deploymentId,
			slog_keys.AuxDeploymentIds, auxDepIds,
			slog_keys.Error, err,
		)
		return nil, err
	}
	dbAuxDepConfigs, err := h.databaseHandler.ReadAuxiliaryDeploymentsConfigs(ctx, auxDepIds)
	if err != nil {
		logger.Error(
			"get auxiliary deployments, read configs from database",
			slog_keys.DeploymentId, deploymentId,
			slog_keys.AuxDeploymentIds, auxDepIds,
			slog_keys.Error, err,
		)
		return nil, err
	}
	dbAuxDepVolumeMounts, err := h.databaseHandler.ReadAuxiliaryDeploymentsVolumeMounts(ctx, auxDepIds)
	if err != nil {
		logger.Error(
			"get auxiliary deployments, read volume mounts from database",
			slog_keys.DeploymentId, deploymentId,
			slog_keys.AuxDeploymentIds, auxDepIds,
			slog_keys.Error, err,
		)
		return nil, err
	}
	cewContainers, err := h.getCewContainers(ctx, dbAuxDeployments)
	if err != nil {
		logger.Error(
			"get auxiliary deployments, get containers",
			slog_keys.DeploymentId, deploymentId,
			slog_keys.AuxDeploymentIds, auxDepIds,
			slog_keys.Error, err,
		)
		return nil, err
	}
	return getAuxiliaryDeployments(dbAuxDeployments, dbAuxDepLabels, dbAuxDepConfigs, dbAuxDepVolumeMounts, cewContainers), nil
}

func (h *Handler) GetReducedDeployments(
	ctx context.Context,
	deploymentId string,
	filter lib_models.AuxiliaryDeploymentsFilterWithState,
) (map[string]lib_models.AuxiliaryDeploymentReduced, error) {
	mu := h.mutexes.Get(deploymentId)
	mu.RLock()
	defer mu.RUnlock()
	dbAuxDeployments, err := h.databaseHandler.ReadAuxiliaryDeployments(ctx, deploymentId, filter.AuxiliaryDeploymentsFilter)
	if err != nil {
		logger.Error(
			"get reduced auxiliary deployments, read from database",
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Filter, filter,
			slog_keys.Error, err,
		)
		return nil, err
	}
	cewContainers, err := h.getCewContainers(ctx, dbAuxDeployments)
	if err != nil {
		logger.Error(
			"get reduced auxiliary deployments, get containers",
			slog_keys.DeploymentId, deploymentId,
			slog_keys.AuxDeploymentIds, slices.Collect(maps.Keys(dbAuxDeployments)),
			slog_keys.Error, err,
		)
		return nil, err
	}
	return getReducedAuxiliaryDeployments(dbAuxDeployments, cewContainers), nil
}

func (h *Handler) getCewContainers(
	ctx context.Context,
	auxDeployments map[string]pkg_models.AuxiliaryDeployment,
) (map[string]external_models.CewContainer, error) {
	cewContainers, err := h.containerEngineWrapperClient.GetContainers(ctx, external_models.CewContainersFilter{
		Names: helper_slices.CollectFunc(maps.Values(auxDeployments), func(item pkg_models.AuxiliaryDeployment) string {
			return item.Container.Name
		}),
	})
	if err != nil {
		return nil, err
	}
	cewContainersMap := maps.Collect(helper_slices.AllFunc(cewContainers, func(item external_models.CewContainer) string {
		return item.Name
	}))
	return cewContainersMap, nil
}

func getAuxiliaryDeployments(
	dbAuxDeployments map[string]pkg_models.AuxiliaryDeployment,
	dbAuxDepLabels map[string]map[string]string,
	dbAuxDepConfigs map[string]map[string]string,
	dbAuxDepVolumeMounts map[string][]pkg_models.AuxiliaryDeploymentVolumeMount,
	cewContainers map[string]external_models.CewContainer,
) map[string]lib_models.AuxiliaryDeployment {
	auxDeployments := make(map[string]lib_models.AuxiliaryDeployment)
	for id, dbAuxDep := range dbAuxDeployments {
		cewContainer := cewContainers[dbAuxDep.Container.Name]
		auxDeployments[id] = lib_models.AuxiliaryDeployment{
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
	dbAuxDeployments map[string]pkg_models.AuxiliaryDeployment,
	cewContainers map[string]external_models.CewContainer,
) map[string]lib_models.AuxiliaryDeploymentReduced {
	auxDeployments := make(map[string]lib_models.AuxiliaryDeploymentReduced)
	for id, dbAuxDep := range dbAuxDeployments {
		cewContainer := cewContainers[dbAuxDep.Container.Name]
		auxDeployments[id] = lib_models.AuxiliaryDeploymentReduced{
			AuxiliaryDeploymentBase: newAuxiliaryDeploymentBase(dbAuxDep),
			Container:               getContainer(dbAuxDep.Container, cewContainer),
		}
	}
	return auxDeployments
}

func getVolumes(mounts []pkg_models.AuxiliaryDeploymentVolumeMount) []lib_models.AuxiliaryDeploymentVolumeMount {
	var volumes []lib_models.AuxiliaryDeploymentVolumeMount
	for _, mount := range mounts {
		volumes = append(volumes, lib_models.AuxiliaryDeploymentVolumeMount{
			Reference: mount.Reference,
			MountPath: mount.MountPath,
		})
	}
	return volumes
}

func getContainer(dbContainer pkg_models.AuxiliaryDeploymentContainer, cewContainer external_models.CewContainer) lib_models.Container {
	ctrInfo := lib_models.Container{
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

func newAuxiliaryDeploymentBase(dbAuxDep pkg_models.AuxiliaryDeployment) lib_models.AuxiliaryDeploymentBase {
	return lib_models.AuxiliaryDeploymentBase{
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
