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

package handler_deployments

import (
	"context"
	"maps"
	"os"
	"path"
	"slices"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/maps"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployments"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/global_configs"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/modules"
)

func (h *Handler) UpdateDeployments(
	ctx context.Context,
	selectedModules map[string]models_handler_modules.Module,
	userInputs map[string]models_handler_deployments.UserInput,
) ([]models_handler_deployments.Result, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	deployments, err := h.databaseHandler.ReadDeployments(ctx, models_handler_database.DeploymentsFilter{
		ModuleIds: slices.Collect(maps.Keys(selectedModules)),
	})
	if err != nil {
		return nil, err
	}
	deploymentIds := slices.Collect(maps.Keys(deployments))
	deploymentsVolumes, deploymentsContainers, err := h.getDeploymentsVolumesAndContainersFromDB(ctx, deploymentIds)
	if err != nil {
		return nil, err
	}
	cache := cacheCollection{
		HostResources: make(map[string]models_external.HostResource),
		GlobalConfigs: make(map[string]models_handler_global_configs.Config),
		SecretValues:  make(map[string]models_external.SecretValueVariant),
	}
	cache.Deployments, err = initDeploymentsCacheFromModulesAndDeployments(selectedModules, deployments, deploymentsContainers)
	if err != nil {
		return nil, err
	}
	var results []models_handler_deployments.Result
	for moduleId, module := range selectedModules {
		result := models_handler_deployments.Result{ModuleId: moduleId}
		cacheItem, ok := cache.Deployments[moduleId]
		if !ok {
			result.ErrorResult = models_error.NewErrorResult("not installed")
			results = append(results, result)
			continue
		}
		result.Id = cacheItem.DeploymentId
		err = h.updateDeployment(
			ctx,
			module,
			userInputs[moduleId],
			cacheItem.DeploymentId,
			cacheItem.Containers,
			deployments[cacheItem.DeploymentId],
			deploymentsContainers[cacheItem.DeploymentId],
			deploymentsVolumes[cacheItem.DeploymentId],
			cache,
		)
		if err != nil {
			result.ErrorResult = models_error.NewErrorResult(err.Error())
		}
		results = append(results, result)
	}
	return results, nil
}

func (h *Handler) updateDeployment(
	ctx context.Context,
	module models_handler_modules.Module,
	userInput models_handler_deployments.UserInput,
	deploymentId string,
	cacheContainers map[string]containerCacheItem,
	currentDeployment models_handler_database.Deployment,
	currentContainers map[string]models_handler_database.DeploymentContainer,
	currentVolumes map[string]models_handler_database.DeploymentVolume,
	cache cacheCollection,
) error {
	newDeployment, err := getDeployment(module, deploymentId)
	if err != nil {
		return err
	}
	newDeployment.Enabled = currentDeployment.Enabled
	newDeployment.Created = currentDeployment.Created
	newDeployment.Updated = helper_time.Now()
	defaultData, err := getDefaultData(module)
	if err != nil {
		return err
	}
	userData, err := getUserData(module, defaultData, userInput, deploymentId)
	if err != nil {
		return err
	}
	err = h.updateCaches(
		ctx,
		module.Dependencies,
		userData.HostResources,
		userData.Secrets,
		userData.GlobalConfigs,
		cache,
	)
	if err != nil {
		return err
	}
	mergedConfigs, mergedFiles, err := mergeDefaultAndUserData(
		module,
		defaultData,
		userData.Configs,
		userData.GlobalConfigs,
		userData.Files,
		cache.GlobalConfigs,
	)
	if err != nil {
		return err
	}
	newContainers, err := getNewContainers(module.Services, cacheContainers, deploymentId)
	if err != nil {
		return err
	}
	err = h.stopContainers(ctx, currentContainers)
	if err != nil {
		return err
	}
	updatedVolumes := updateVolumes(module.Volumes, currentVolumes, deploymentId)
	err = h.removeDeploymentEnvironment(
		ctx,
		deploymentId,
		currentDeployment.DirName,
		currentDeployment.FilesDirName,
		currentContainers,
	)
	if err != nil {
		return err
	}
	err = h.databaseHandler.UpdateDeployment(
		ctx,
		newDeployment,
		slices.Collect(maps.Values(userData.HostResources)),
		slices.Collect(maps.Values(userData.Secrets)),
		slices.Collect(maps.Values(userData.Configs)),
		slices.Collect(maps.Values(userData.GlobalConfigs)),
		slices.Collect(maps.Values(userData.Files)),
		slices.Collect(maps.Values(userData.FileGroups)),
		slices.Collect(maps.Values(updatedVolumes)),
		slices.Collect(maps.Values(newContainers)),
	)
	if err != nil {
		return err
	}
	err = h.ensureDeploymentEnvironment(
		ctx,
		module.Services,
		module.FileSystem,
		deploymentId,
		newDeployment.DirName,
		newDeployment.FilesDirName,
		updatedVolumes,
	)
	if err != nil {
		return err
	}
	bindMounts, err := h.getBindMounts(
		ctx,
		deploymentId,
		newDeployment.FilesDirName,
		userData.FileGroups,
		userData.Secrets,
		mergedFiles,
	)
	if err != nil {
		return err
	}
	// TODO "mount secrets" must be "unloaded" if one of the following steps fail
	err = h.createHttpEndpoints(ctx, module.Services, module.ID, newContainers)
	if err != nil {
		logger.Error(err.Error()) // TODO
	}
	err = h.createContainers(
		ctx,
		module.Configs,
		module.Services,
		deploymentId,
		newDeployment.DirName,
		newDeployment.FilesDirName,
		userData.Secrets,
		userData.HostResources,
		newContainers,
		updatedVolumes,
		mergedConfigs,
		bindMounts,
		cache.SecretValues,
		cache.Deployments,
		cache.HostResources,
	)
	if err != nil {
		logger.Error(err.Error()) // TODO
	}
	return nil
}

func initDeploymentsCacheFromModulesAndDeployments(
	modules map[string]models_handler_modules.Module,
	deployments map[string]models_handler_database.Deployment,
	deploymentsContainers map[string]map[string]models_handler_database.DeploymentContainer,
) (map[string]deploymentsCacheItem, error) {
	cache := make(map[string]deploymentsCacheItem)
	deployments = helper_maps.CollectFunc(maps.Values(deployments), func(value models_handler_database.Deployment) string {
		return value.ModuleId
	})
	for moduleId, module := range modules {
		deployment, ok := deployments[moduleId]
		if !ok {
			continue
		}
		containers := make(map[string]containerCacheItem)
		for reference := range module.Services {
			existingContainer := deploymentsContainers[deployment.Id][reference]
			name, err := helper_naming.NewContainerName(models_constants.DeploymentAbbreviation)
			if err != nil {
				return nil, err
			}
			alias := existingContainer.Alias
			if alias == "" {
				alias = helper_naming.NewContainerAlias(deployment.Id, reference)
			}
			containers[reference] = containerCacheItem{
				Name:  name,
				Alias: alias,
			}
		}
		cache[moduleId] = deploymentsCacheItem{
			DeploymentId: deployment.Id,
			Containers:   containers,
		}
	}
	return cache, nil
}

func (h *Handler) removeDeploymentDirs(deploymentDirName, deploymentFilesDirName string) error {
	err := removeDeploymentDir(h.config.WorkDirPath, deploymentDirName)
	if err != nil {
		return err
	}
	return removeFilesDir(h.config.WorkDirPath, deploymentFilesDirName)
}

func removeDeploymentDir(workDirPath, deploymentDirName string) error {
	return os.RemoveAll(path.Join(workDirPath, deploymentDirName))
}

func updateVolumes(
	moduleVolumes map[string]struct{},
	deploymentVolumes map[string]models_handler_database.DeploymentVolume,
	deploymentId string,
) map[string]models_handler_database.DeploymentVolume {
	volumes := make(map[string]models_handler_database.DeploymentVolume)
	for reference := range moduleVolumes {
		volume := deploymentVolumes[reference]
		name := volume.Name
		if name == "" {
			name = helper_naming.NewVolumeName(models_constants.DeploymentAbbreviation, deploymentId, reference)
		}
		volumes[reference] = models_handler_database.DeploymentVolume{
			DeploymentId: deploymentId,
			Reference:    reference,
			Name:         name,
		}
	}
	return volumes
}
