/*
 * Copyright 2025 InfAI (CC SES)
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
	"os"
	"path"
	"slices"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_file_sys "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/file_sys"
	helper_maps "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/maps"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	helper_uuid "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/uuid"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) CreateDeployments(
	ctx context.Context,
	selectedModules map[string]pkg_models.Module,
	userInputs map[string]pkg_models.DeploymentUserInput,
) ([]lib_models.DeploymentResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	cache := cacheCollection{
		HostResources: make(map[string]external_models.HmHostResource),
		GlobalConfigs: make(map[string]pkg_models.Config),
		SecretValues:  make(map[string]external_models.SmSecretValueVariant),
	}
	var err error
	selectedModules, err = h.filterSelectedModules(ctx, selectedModules)
	if err != nil {
		logger.ErrorContext(ctx, "create deployments, filter selected modules", slog_keys.Error, err)
		return nil, err
	}
	cache.Deployments, err = initDeploymentsCacheFromModules(selectedModules)
	if err != nil {
		logger.ErrorContext(ctx, "create deployments, initialize cache", slog_keys.Error, err)
		return nil, err
	}
	var results []lib_models.DeploymentResult
	for moduleId, module := range selectedModules {
		cacheItem := cache.Deployments[moduleId]
		result := lib_models.DeploymentResult{
			ModuleId: moduleId,
			Id:       cacheItem.DeploymentId,
		}
		err = h.createDeployment(
			ctx,
			module,
			userInputs[moduleId],
			cacheItem.DeploymentId,
			cacheItem.Containers,
			cache,
		)
		if err != nil {
			result.ErrorResult = lib_models.NewErrorResult(err.Error())
		}
		results = append(results, result)
	}
	return results, nil
}

func (h *Handler) createDeployment(
	ctx context.Context,
	module pkg_models.Module,
	userInput pkg_models.DeploymentUserInput,
	deploymentId string,
	cacheContainers map[string]containerCacheItem,
	cache cacheCollection,
) error {
	newDeployment, err := getDeployment(module, deploymentId)
	if err != nil {
		logger.ErrorContext(ctx, "create deployment, generate new deployment", slog_keys.ModuleId, module.ID, slog_keys.Error, err)
		return err
	}
	newDeployment.Created = helper_time.Now()
	newDeployment.Updated = newDeployment.Created
	defaultData, err := getDefaultData(module)
	if err != nil {
		logger.ErrorContext(ctx, "create deployment, get default data", slog_keys.ModuleId, module.ID, slog_keys.Error, err)
		return err
	}
	userData, err := getUserData(module, defaultData, userInput, deploymentId)
	if err != nil {
		logger.ErrorContext(ctx, "create deployment, get user data", slog_keys.ModuleId, module.ID, slog_keys.Error, err)
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
		logger.ErrorContext(ctx, "create deployment, get dependencies and external resources", slog_keys.ModuleId, module.ID, slog_keys.Error, err)
		return err
	}
	globalConfigs, err := getGlobalConfigs(module.Configs, userData.GlobalConfigs, cache.GlobalConfigs)
	if err != nil {
		logger.ErrorContext(ctx, "create deployment, get global configs", slog_keys.ModuleId, module.ID, slog_keys.Error, err)
		return err
	}
	mergedConfigs, mergedFiles, err := mergeDefaultAndUserData(
		module,
		defaultData,
		userData.Configs,
		userData.Files,
		globalConfigs,
	)
	if err != nil {
		logger.ErrorContext(ctx, "create deployment, merge default and user data", slog_keys.ModuleId, module.ID, slog_keys.Error, err)
		return err
	}
	newContainers, err := getNewContainers(module.Services, cacheContainers, deploymentId)
	if err != nil {
		logger.ErrorContext(ctx, "create deployment, generate new containers", slog_keys.ModuleId, module.ID, slog_keys.Error, err)
		return err
	}
	newVolumes := getNewVolumes(module.Volumes, deploymentId)
	err = h.databaseHandler.CreateDeployment(
		ctx,
		newDeployment,
		slices.Collect(maps.Values(userData.HostResources)),
		slices.Collect(maps.Values(userData.Secrets)),
		slices.Collect(maps.Values(userData.Configs)),
		slices.Collect(maps.Values(userData.GlobalConfigs)),
		slices.Collect(maps.Values(userData.Files)),
		slices.Collect(maps.Values(userData.FileGroups)),
		slices.Collect(maps.Values(newVolumes)),
		slices.Collect(maps.Values(newContainers)),
	)
	if err != nil {
		logger.ErrorContext(ctx, "create deployment, write to database", slog_keys.ModuleId, module.ID, slog_keys.Error, err)
		return err
	}
	err = h.ensureDeploymentEnvironment(
		ctx,
		module.Services,
		module.FileSystem,
		deploymentId,
		newDeployment.DirName,
		newDeployment.FilesDirName,
		newVolumes,
	)
	if err != nil {
		logger.ErrorContext(ctx, "create deployment, ensure environment", slog_keys.ModuleId, module.ID, slog_keys.Error, err)
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
		logger.ErrorContext(ctx, "create deployment, get bind mounts", slog_keys.ModuleId, module.ID, slog_keys.Error, err)
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
		newVolumes,
		mergedConfigs,
		bindMounts,
		cache.SecretValues,
		cache.Deployments,
		cache.HostResources,
	)
	if err != nil {
		logger.ErrorContext(ctx, "create deployment, create containers", slog_keys.ModuleId, module.ID, slog_keys.Error, err)
		return err
	}
	err = h.createHttpEndpoints(ctx, module.Services, module.ID, deploymentId, newContainers)
	if err != nil {
		logger.ErrorContext(ctx, "create deployment, create http endpoints", slog_keys.ModuleId, module.ID, slog_keys.Error, err)
		return err
	}
	logger.InfoContext(ctx, "create deployment", slog_keys.ModuleId, module.ID, slog_keys.DeploymentId, deploymentId)
	return nil
}

func (h *Handler) getBindMounts(
	ctx context.Context,
	deploymentId,
	deploymentFilesDirName string,
	userDataFileGroups map[string]pkg_models.DeploymentFileGroup,
	userDataSecrets map[string]pkg_models.DeploymentSecret,
	mergedFiles map[string][]byte,
) (bindMountDataCollection, error) {
	var bindMounts bindMountDataCollection
	var err error
	bindMounts.Files, err = createFiles(deploymentId, deploymentFilesDirName, mergedFiles, h.config.WorkdirPath)
	if err != nil {
		return bindMountDataCollection{}, err
	}
	bindMounts.FileGroups, err = createFileGroups(deploymentFilesDirName, userDataFileGroups, h.config.WorkdirPath)
	if err != nil {
		return bindMountDataCollection{}, err
	}
	bindMounts.Secrets, err = h.createSecretMounts(ctx, deploymentId, userDataSecrets)
	if err != nil {
		return bindMountDataCollection{}, err
	}
	return bindMounts, nil
}

func (h *Handler) createDeploymentDirs(moduleFileSystem fs.FS, deploymentDirName, deploymentFilesDirName string) error {
	err := createDeploymentDir(moduleFileSystem, h.config.WorkdirPath, deploymentDirName)
	if err != nil {
		return err
	}
	err = createFilesDir(h.config.WorkdirPath, deploymentFilesDirName)
	if err != nil {
		return err
	}
	return err
}

func (h *Handler) filterSelectedModules(
	ctx context.Context,
	selectedModules map[string]pkg_models.Module,
) (map[string]pkg_models.Module, error) {
	deployments, err := h.databaseHandler.ReadDeployments(ctx, pkg_models.DeploymentsFilter{
		ModuleIds: slices.Collect(maps.Keys(selectedModules)),
	})
	if err != nil {
		return nil, err
	}
	deployments = helper_maps.CollectFunc(maps.Values(deployments), func(value pkg_models.DeploymentBase) string {
		return value.ModuleId
	})
	filteredModules := make(map[string]pkg_models.Module)
	for moduleId, module := range selectedModules {
		_, ok := deployments[moduleId]
		if !ok {
			filteredModules[moduleId] = module
		}
	}
	return filteredModules, nil
}

func createDeploymentDir(moduleFileSystem fs.FS, workDirPath, deploymentDirName string) error {
	dirPath := path.Join(workDirPath, deploymentDirName)
	err := os.Mkdir(dirPath, dirPerm)
	if err != nil {
		return err
	}
	return helper_file_sys.CopyAll(moduleFileSystem, dirPath)
}

func getDefaultData(module pkg_models.Module) (defaultDataCollection, error) {
	var data defaultDataCollection
	var err error
	data.Configs, err = getDefaultConfigs(module.Configs)
	if err != nil {
		return defaultDataCollection{}, err
	}
	data.Files = getDefaultFiles(module.Files)
	return data, nil
}

func getUserData(
	module pkg_models.Module,
	defaultData defaultDataCollection,
	userInput pkg_models.DeploymentUserInput,
	deploymentId string,
) (userDataCollection, error) {
	var data userDataCollection
	var err error
	data.GlobalConfigs = getSelectedGlobalConfigs(module.Configs, userInput.GlobalConfigs, deploymentId)
	data.HostResources, err = getSelectedHostResources(module.HostResources, userInput.HostResources, deploymentId)
	if err != nil {
		return userDataCollection{}, err
	}
	data.Secrets, err = getSelectedSecrets(module, userInput.Secrets, deploymentId)
	if err != nil {
		return userDataCollection{}, err
	}
	data.Configs = getProvidedConfigs(module.Configs, defaultData.Configs, userInput.Configs, deploymentId)
	data.Files = getProvidedFiles(module.Files, defaultData.Files, userInput.Files, deploymentId)
	data.FileGroups = getProvidedFileGroups(module.FileGroups, userInput.FileGroups, deploymentId)
	return data, nil
}

func getDeployment(
	module pkg_models.Module,
	deploymentId string,
) (pkg_models.DeploymentBase, error) {
	if deploymentId == "" {
		return pkg_models.DeploymentBase{}, errors.New("empty deployment id")
	}
	dirName, err := helper_uuid.New()
	if err != nil {
		return pkg_models.DeploymentBase{}, err
	}
	return pkg_models.DeploymentBase{
		Id:            deploymentId,
		ModuleId:      module.ID,
		ModuleSource:  module.Source,
		ModuleChannel: module.Channel,
		ModuleVersion: module.Version,
		DirName:       dirName,
		FilesDirName:  dirName + "_files",
	}, nil
}

func initDeploymentsCacheFromModules(modules map[string]pkg_models.Module) (map[string]deploymentsCacheItem, error) {
	cache := make(map[string]deploymentsCacheItem)
	for moduleId, module := range modules {
		id, err := helper_uuid.New()
		if err != nil {
			return nil, err
		}
		containers := make(map[string]containerCacheItem)
		for reference := range module.Services {
			name, err := helper_naming.NewContainerName(constants.DeploymentAbbreviation)
			if err != nil {
				return nil, err
			}
			containers[reference] = containerCacheItem{
				Name:  name,
				Alias: helper_naming.NewContainerAlias(id, reference),
			}
		}
		cache[moduleId] = deploymentsCacheItem{
			DeploymentId: id,
			Containers:   containers,
		}
	}
	return cache, nil
}

func getNewVolumes(moduleVolumes map[string]struct{}, deploymentId string) map[string]pkg_models.DeploymentVolume {
	volumes := make(map[string]pkg_models.DeploymentVolume)
	for reference := range moduleVolumes {
		volumes[reference] = pkg_models.DeploymentVolume{
			DeploymentId: deploymentId,
			Reference:    reference,
			Name:         helper_naming.NewVolumeName(constants.DeploymentAbbreviation, deploymentId, reference),
		}
	}
	return volumes
}

func getNewContainers(
	moduleServices map[string]external_models.ModuleLibService,
	cacheContainers map[string]containerCacheItem,
	deploymentId string,
) (map[string]pkg_models.DeploymentContainerBase, error) {
	containers := make(map[string]pkg_models.DeploymentContainerBase)
	for reference := range moduleServices {
		cacheItem, ok := cacheContainers[reference]
		if !ok {
			return nil, errors.New(fmt.Sprintf("'%s' missing container alias", reference))
		}
		containers[reference] = pkg_models.DeploymentContainerBase{
			Name:         cacheItem.Name,
			DeploymentId: deploymentId,
			Reference:    reference,
			Alias:        cacheItem.Alias,
		}
	}
	return containers, nil
}
