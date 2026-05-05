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
	"errors"
	"io/fs"
	"maps"
	"slices"

	models_error "github.com/SENERGY-Platform/mgw-module-manager/lib/models/results"
	models_config "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/config"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_database "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
	models_handler_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployments"
	models_handler_global_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/global_configs"
	models_handler_modules "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/modules"
)

func (h *Handler) RecreateDeployments(
	ctx context.Context,
	selectedModules map[string]models_handler_modules.Module,
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
	deploymentsUserData, err := h.getDeploymentsUserDataFromDB(ctx, deploymentIds)
	if err != nil {
		return nil, err
	}
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
		err = h.recreateDeployment(
			ctx,
			module,
			deploymentsUserData[cacheItem.DeploymentId],
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

func (h *Handler) recreateDeployment(
	ctx context.Context,
	module models_handler_modules.Module,
	userData userDataCollection,
	deploymentId string,
	cacheContainers map[string]containerCacheItem,
	currentDeployment models_handler_database.Deployment,
	currentContainers map[string]models_handler_database.DeploymentContainer,
	currentVolumes map[string]models_handler_database.DeploymentVolume,
	cache cacheCollection,
) error {
	if currentDeployment.ModuleSource+currentDeployment.ModuleChannel+currentDeployment.ModuleVersion != module.Source+module.Channel+module.Version {
		return errors.New("module " + module.ID + " has changed and must be updated first")
	}
	defaultData, err := getDefaultData(module)
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
	err = h.removeDeploymentEnvironment(
		ctx,
		deploymentId,
		currentDeployment.DirName,
		currentDeployment.FilesDirName,
		currentContainers,
	)
	err = h.databaseHandler.UpdateDeploymentContainerNames(ctx, slices.Collect(maps.Values(newContainers)))
	if err != nil {
		return err
	}
	err = h.ensureDeploymentEnvironment(
		ctx,
		module.Services,
		module.FileSystem,
		deploymentId,
		currentDeployment.DirName,
		currentDeployment.FilesDirName,
		currentVolumes,
	)
	if err != nil {
		return err
	}
	bindMounts, err := h.getBindMounts(
		ctx,
		deploymentId,
		currentDeployment.FilesDirName,
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
		currentDeployment.DirName,
		currentDeployment.FilesDirName,
		userData.Secrets,
		userData.HostResources,
		newContainers,
		currentVolumes,
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

func (h *Handler) getDeploymentsUserDataFromDB(ctx context.Context, deploymentIds []string) (map[string]userDataCollection, error) {
	deploymentsHostResources, err := h.databaseHandler.ReadDeploymentsHostResources(ctx, models_handler_database.DeploymentsHostResourcesFilter{
		DeploymentIds: deploymentIds,
	})
	if err != nil {
		return nil, err
	}
	deploymentsSecrets, err := h.databaseHandler.ReadDeploymentsSecrets(ctx, models_handler_database.DeploymentsSecretsFilter{
		DeploymentIds: deploymentIds,
	})
	if err != nil {
		return nil, err
	}
	deploymentsConfigs, err := h.databaseHandler.ReadDeploymentsConfigs(ctx, deploymentIds)
	if err != nil {
		return nil, err
	}
	deploymentsGlobalConfigs, err := h.databaseHandler.ReadDeploymentsGlobalConfigs(ctx, models_handler_database.DeploymentGlobalConfigsFilter{
		DeploymentIds: deploymentIds,
	})
	if err != nil {
		return nil, err
	}
	deploymentsFiles, err := h.databaseHandler.ReadDeploymentsFiles(ctx, deploymentIds)
	if err != nil {
		return nil, err
	}
	deploymentsFileGroups, err := h.databaseHandler.ReadDeploymentsFileGroups(ctx, deploymentIds)
	if err != nil {
		return nil, err
	}
	deploymentsData := make(map[string]userDataCollection)
	for _, deploymentId := range deploymentIds {
		deploymentsData[deploymentId] = userDataCollection{
			GlobalConfigs: deploymentsGlobalConfigs[deploymentId],
			HostResources: deploymentsHostResources[deploymentId],
			Secrets:       deploymentsSecrets[deploymentId],
			Configs:       deploymentsConfigs[deploymentId],
			Files:         deploymentsFiles[deploymentId],
			FileGroups:    deploymentsFileGroups[deploymentId],
		}
	}
	return deploymentsData, nil
}

func (h *Handler) getDeploymentsVolumesAndContainersFromDB(ctx context.Context, deploymentIds []string) (
	map[string]map[string]models_handler_database.DeploymentVolume,
	map[string]map[string]models_handler_database.DeploymentContainer,
	error,
) {
	deploymentsVolumes, err := h.databaseHandler.ReadDeploymentsVolumes(ctx, deploymentIds)
	if err != nil {
		return nil, nil, err
	}
	deploymentsContainers, err := h.databaseHandler.ReadDeploymentsContainers(ctx, deploymentIds)
	if err != nil {
		return nil, nil, err
	}
	return deploymentsVolumes, deploymentsContainers, nil
}

func (h *Handler) updateCaches(
	ctx context.Context,
	moduleDependencies map[string]string,
	userDataHostResources map[string]models_handler_database.DeploymentHostResource,
	userDataSecrets map[string]models_handler_database.DeploymentSecret,
	userDataGlobalConfigs map[string]models_handler_database.DeploymentGlobalConfig,
	cache cacheCollection,
) error {
	err := h.updateDeploymentsCache(ctx, moduleDependencies, cache.Deployments)
	if err != nil {
		return err
	}
	err = h.updateGlobalConfigsCache(ctx, userDataGlobalConfigs, cache.GlobalConfigs)
	if err != nil {
		return err
	}
	err = h.updateHostResourcesCache(ctx, userDataHostResources, cache.HostResources)
	if err != nil {
		return err
	}
	err = h.updateSecretValuesCache(ctx, userDataSecrets, cache.SecretValues)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) ensureDeploymentEnvironment(
	ctx context.Context,
	moduleServices map[string]models_external.ModuleLibService,
	moduleFileSystem fs.FS,
	deploymentId string,
	deploymentDirName string,
	deploymentFilesDirName string,
	volumes map[string]models_handler_database.DeploymentVolume,
) error {
	err := h.ensureContainerImages(ctx, moduleServices)
	if err != nil {
		return err
	}
	err = h.ensureContainerVolumes(ctx, volumes, deploymentId)
	if err != nil {
		return err
	}
	return h.createDeploymentDirs(moduleFileSystem, deploymentDirName, deploymentFilesDirName)
}

func (h *Handler) removeDeploymentEnvironment(
	ctx context.Context,
	deploymentId string,
	deploymentDirName string,
	deploymentFilesDirName string,
	deploymentContainers map[string]models_handler_database.DeploymentContainer,
) error {
	err := h.removeContainers(ctx, deploymentContainers)
	if err != nil {
		return err
	}
	err = h.removeSecretMounts(ctx, deploymentId)
	if err != nil {
		return err
	}
	return h.removeDeploymentDirs(deploymentDirName, deploymentFilesDirName)
}

func mergeDefaultAndUserData(
	module models_handler_modules.Module,
	defaultData defaultDataCollection,
	userDataConfigs map[string]models_handler_database.DeploymentUserConfig,
	userDataGlobalConfigs map[string]models_handler_database.DeploymentGlobalConfig,
	userDataFiles map[string]models_handler_database.DeploymentFile,
	cacheGlobalConfigs map[string]models_handler_global_configs.Config,
) (map[string]models_config.Value, map[string][]byte, error) {
	mergedConfigs := mergeConfigs(defaultData.Configs, userDataConfigs, userDataGlobalConfigs, cacheGlobalConfigs)
	err := checkConfigs(module.Configs, mergedConfigs)
	if err != nil {
		return nil, nil, err
	}
	mergedFiles := mergeFiles(defaultData.Files, userDataFiles)
	err = checkFiles(module.Files, mergedFiles)
	if err != nil {
		return nil, nil, err
	}
	return mergedConfigs, mergedFiles, nil
}
