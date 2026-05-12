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
	"fmt"
	"io/fs"
	"maps"
	"slices"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) RecreateDeployments(
	ctx context.Context,
	selectedModules map[string]pkg_models.Module,
) ([]lib_models.DeploymentResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	moduleIds := slices.Collect(maps.Keys(selectedModules))
	deployments, err := h.databaseHandler.ReadDeployments(ctx, pkg_models.DeploymentsFilter{
		ModuleIds: moduleIds,
	})
	if err != nil {
		logger.ErrorContext(ctx, "recreate deployments, read from database", slog_keys.ModuleIds, moduleIds, slog_keys.Error, err)
		return nil, err
	}
	deploymentIds := slices.Collect(maps.Keys(deployments))
	deploymentsUserData, err := h.getDeploymentsUserDataFromDB(ctx, deploymentIds)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"recreate deployments, read user data from database",
			slog_keys.DeploymentIds, deploymentIds,
			slog_keys.Error, err,
		)
		return nil, err
	}
	deploymentsVolumes, deploymentsContainers, err := h.getDeploymentsVolumesAndContainersFromDB(ctx, deploymentIds)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"recreate deployments, read volume and container data from database",
			slog_keys.DeploymentIds, deploymentIds,
			slog_keys.Error, err,
		)
		return nil, err
	}
	cache := cacheCollection{
		HostResources: make(map[string]external_models.HmHostResource),
		GlobalConfigs: make(map[string]pkg_models.Config),
		SecretValues:  make(map[string]external_models.SmSecretValueVariant),
	}
	cache.Deployments, err = initDeploymentsCacheFromModulesAndDeployments(selectedModules, deployments, deploymentsContainers)
	if err != nil {
		logger.ErrorContext(ctx, "recreate deployments, initialize cache", slog_keys.Error, err)
		return nil, err
	}
	var results []lib_models.DeploymentResult
	for moduleId, module := range selectedModules {
		result := lib_models.DeploymentResult{ModuleId: moduleId}
		cacheItem, ok := cache.Deployments[moduleId]
		if !ok {
			result.ErrorResult = lib_models.NewErrorResult("not installed")
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
			result.ErrorResult = lib_models.NewErrorResult(err.Error())
		}
		results = append(results, result)
	}
	return results, nil
}

func (h *Handler) recreateDeployment(
	ctx context.Context,
	module pkg_models.Module,
	userData userDataCollection,
	deploymentId string,
	cacheContainers map[string]containerCacheItem,
	currentDeployment pkg_models.DeploymentBase,
	currentContainers map[string]pkg_models.DeploymentContainerBase,
	currentVolumes map[string]pkg_models.DeploymentVolume,
	cache cacheCollection,
) error {
	if currentDeployment.ModuleSource+currentDeployment.ModuleChannel+currentDeployment.ModuleVersion != module.Source+module.Channel+module.Version {
		msg := fmt.Sprintf("module '%s' has changed and must be updated first", module.ID)
		logger.ErrorContext(ctx, "recreate deployment", slog_keys.ModuleId, module.ID, slog_keys.DeploymentId, deploymentId, slog_keys.Error, msg)
		return errors.New(msg)
	}
	defaultData, err := getDefaultData(module)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"recreate deployment, get default data",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
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
		logger.ErrorContext(
			ctx,
			"recreate deployment, get dependencies and external resources",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
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
		logger.ErrorContext(
			ctx,
			"recreate deployment, merge default and user data",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	newContainers, err := getNewContainers(module.Services, cacheContainers, deploymentId)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"recreate deployment, generate new containers",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	err = h.stopContainers(ctx, currentContainers)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"recreate deployment, stop containers",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	err = h.removeDeploymentEnvironment(
		ctx,
		deploymentId,
		currentDeployment.DirName,
		currentDeployment.FilesDirName,
		currentContainers,
	)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"recreate deployment, remove environment",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	err = h.databaseHandler.UpdateDeploymentContainerNames(ctx, slices.Collect(maps.Values(newContainers)))
	if err != nil {
		logger.ErrorContext(
			ctx,
			"recreate deployment, write container names to database",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
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
		logger.ErrorContext(
			ctx,
			"recreate deployment, ensure environment",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
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
		logger.ErrorContext(
			ctx,
			"recreate deployment, get bind mounts",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	// TODO "mount secrets" must be "unloaded" if one of the following steps fail
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
		logger.ErrorContext(
			ctx,
			"recreate deployment, create containers",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	err = h.createHttpEndpoints(ctx, module.Services, module.ID, newContainers)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"recreate deployment, create http endpoints",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	return nil
}

func (h *Handler) getDeploymentsUserDataFromDB(ctx context.Context, deploymentIds []string) (map[string]userDataCollection, error) {
	deploymentsHostResources, err := h.databaseHandler.ReadDeploymentsHostResources(ctx, pkg_models.DeploymentsHostResourcesFilter{
		DeploymentIds: deploymentIds,
	})
	if err != nil {
		return nil, err
	}
	deploymentsSecrets, err := h.databaseHandler.ReadDeploymentsSecrets(ctx, pkg_models.DeploymentsSecretsFilter{
		DeploymentIds: deploymentIds,
	})
	if err != nil {
		return nil, err
	}
	deploymentsConfigs, err := h.databaseHandler.ReadDeploymentsConfigs(ctx, deploymentIds)
	if err != nil {
		return nil, err
	}
	deploymentsGlobalConfigs, err := h.databaseHandler.ReadDeploymentsGlobalConfigs(ctx, pkg_models.DeploymentGlobalConfigsFilter{
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
	map[string]map[string]pkg_models.DeploymentVolume,
	map[string]map[string]pkg_models.DeploymentContainerBase,
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
	userDataHostResources map[string]pkg_models.DeploymentHostResource,
	userDataSecrets map[string]pkg_models.DeploymentSecret,
	userDataGlobalConfigs map[string]pkg_models.DeploymentGlobalConfig,
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
	moduleServices map[string]external_models.ModuleLibService,
	moduleFileSystem fs.FS,
	deploymentId string,
	deploymentDirName string,
	deploymentFilesDirName string,
	volumes map[string]pkg_models.DeploymentVolume,
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
	deploymentContainers map[string]pkg_models.DeploymentContainerBase,
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
	module pkg_models.Module,
	defaultData defaultDataCollection,
	userDataConfigs map[string]pkg_models.DeploymentUserConfig,
	userDataGlobalConfigs map[string]pkg_models.DeploymentGlobalConfig,
	userDataFiles map[string]pkg_models.DeploymentFile,
	cacheGlobalConfigs map[string]pkg_models.Config,
) (map[string]pkg_models.Value, map[string][]byte, error) {
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
