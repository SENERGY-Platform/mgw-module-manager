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
	"maps"
	"os"
	"path"
	"slices"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_maps "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/maps"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) UpdateDeployments(
	ctx context.Context,
	selectedModules map[string]pkg_models.Module,
	userInputs map[string]pkg_models.DeploymentUserInput,
) ([]lib_models.DeploymentResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	moduleIds := slices.Collect(maps.Keys(selectedModules))
	deployments, err := h.databaseHandler.ReadDeployments(ctx, pkg_models.DeploymentsFilter{
		ModuleIds: moduleIds,
	})
	if err != nil {
		logger.ErrorContext(ctx, "update deployments, read from database", slog_keys.ModuleIds, moduleIds, slog_keys.Error, err)
		return nil, err
	}
	deploymentIds := slices.Collect(maps.Keys(deployments))
	deploymentsVolumes, deploymentsContainers, err := h.getDeploymentsVolumesAndContainersFromDB(ctx, deploymentIds)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"update deployments, read volume and container data from database",
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
		logger.ErrorContext(ctx, "update deployments, initialize cache", slog_keys.Error, err)
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
			result.ErrorResult = lib_models.NewErrorResult(err.Error())
		}
		results = append(results, result)
	}
	return results, nil
}

func (h *Handler) updateDeployment(
	ctx context.Context,
	module pkg_models.Module,
	userInput pkg_models.DeploymentUserInput,
	deploymentId string,
	cacheContainers map[string]containerCacheItem,
	currentDeployment pkg_models.DeploymentBase,
	currentContainers map[string]pkg_models.DeploymentContainerBase,
	currentVolumes map[string]pkg_models.DeploymentVolume,
	cache cacheCollection,
) error {
	newDeployment, err := getDeployment(module, deploymentId)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"update deployment, generate new deployment",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	newDeployment.Enabled = currentDeployment.Enabled
	newDeployment.Created = currentDeployment.Created
	newDeployment.Updated = helper_time.Now()
	defaultData, err := getDefaultData(module)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"update deployment, get default data",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	userData, err := getUserData(module, defaultData, userInput, deploymentId)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"update deployment, get user data",
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
			"update deployment, get dependencies and external resources",
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
			"update deployment, merge default and user data",
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
			"update deployment, generate new containers",
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
			"update deployment, stop containers",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
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
		logger.ErrorContext(
			ctx,
			"update deployment, remove environment",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
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
		logger.ErrorContext(
			ctx,
			"update deployment, write to database",
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
		newDeployment.DirName,
		newDeployment.FilesDirName,
		updatedVolumes,
	)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"update deployment, ensure environment",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
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
		logger.ErrorContext(
			ctx,
			"update deployment, get bind mounts",
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
		logger.ErrorContext(
			ctx,
			"update deployment, create containers",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	err = h.createHttpEndpoints(ctx, module.Services, module.ID, deploymentId, newContainers)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"update deployment, create http endpoints",
			slog_keys.ModuleId, module.ID,
			slog_keys.DeploymentId, deploymentId,
			slog_keys.Error, err,
		)
		return err
	}
	return nil
}

func initDeploymentsCacheFromModulesAndDeployments(
	modules map[string]pkg_models.Module,
	deployments map[string]pkg_models.DeploymentBase,
	deploymentsContainers map[string]map[string]pkg_models.DeploymentContainerBase,
) (map[string]deploymentsCacheItem, error) {
	cache := make(map[string]deploymentsCacheItem)
	deployments = helper_maps.CollectFunc(maps.Values(deployments), func(value pkg_models.DeploymentBase) string {
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
			name, err := helper_naming.NewContainerName(constants.DeploymentAbbreviation)
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
	err := removeDeploymentDir(h.config.WorkdirPath, deploymentDirName)
	if err != nil {
		return err
	}
	return removeFilesDir(h.config.WorkdirPath, deploymentFilesDirName)
}

func removeDeploymentDir(workDirPath, deploymentDirName string) error {
	return os.RemoveAll(path.Join(workDirPath, deploymentDirName))
}

func updateVolumes(
	moduleVolumes map[string]struct{},
	deploymentVolumes map[string]pkg_models.DeploymentVolume,
	deploymentId string,
) map[string]pkg_models.DeploymentVolume {
	volumes := make(map[string]pkg_models.DeploymentVolume)
	for reference := range moduleVolumes {
		volume := deploymentVolumes[reference]
		name := volume.Name
		if name == "" {
			name = helper_naming.NewVolumeName(constants.DeploymentAbbreviation, deploymentId, reference)
		}
		volumes[reference] = pkg_models.DeploymentVolume{
			DeploymentId: deploymentId,
			Reference:    reference,
			Name:         name,
		}
	}
	return volumes
}
