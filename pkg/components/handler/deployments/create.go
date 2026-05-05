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

package handler_deployments

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path"
	"slices"

	models_error "github.com/SENERGY-Platform/mgw-module-manager/lib/models/results"
	lib_models_service "github.com/SENERGY-Platform/mgw-module-manager/lib/models/service"
	helper_file_sys "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/file_sys"
	helper_maps "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/maps"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	helper_uuid "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/uuid"
	models_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/configs"
	models_constants "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	models_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/deployments"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/modules"
)

func (h *Handler) CreateDeployments(
	ctx context.Context,
	selectedModules map[string]models_module.Module,
	userInputs map[string]models_deployments.UserInput,
) ([]lib_models_service.DeploymentResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	cache := cacheCollection{
		HostResources: make(map[string]models_external.HostResource),
		GlobalConfigs: make(map[string]models_configs.Config),
		SecretValues:  make(map[string]models_external.SecretValueVariant),
	}
	var err error
	selectedModules, err = h.filterSelectedModules(ctx, selectedModules)
	if err != nil {
		return nil, err
	}
	cache.Deployments, err = initDeploymentsCacheFromModules(selectedModules)
	if err != nil {
		return nil, err
	}
	var results []lib_models_service.DeploymentResult
	for moduleId, module := range selectedModules {
		result := lib_models_service.DeploymentResult{ModuleId: moduleId}
		cacheItem, ok := cache.Deployments[moduleId]
		if !ok {
			result.ErrorResult = models_error.NewErrorResult("not installed")
			results = append(results, result)
			continue
		}
		result.Id = cacheItem.DeploymentId
		err = h.createDeployment(
			ctx,
			module,
			userInputs[moduleId],
			cacheItem.DeploymentId,
			cacheItem.Containers,
			cache,
		)
		if err != nil {
			result.ErrorResult = models_error.NewErrorResult(err.Error())
		}
		results = append(results, result)
	}
	return results, nil
}

func (h *Handler) createDeployment(
	ctx context.Context,
	module models_module.Module,
	userInput models_deployments.UserInput,
	deploymentId string,
	cacheContainers map[string]containerCacheItem,
	cache cacheCollection,
) error {
	newDeployment, err := getDeployment(module, deploymentId)
	if err != nil {
		return err
	}
	newDeployment.Created = helper_time.Now()
	newDeployment.Updated = newDeployment.Created
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
		return err
	}
	err = h.createHttpEndpoints(ctx, module.Services, module.ID, newContainers)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) getBindMounts(
	ctx context.Context,
	deploymentId,
	deploymentFilesDirName string,
	userDataFileGroups map[string]models_deployments.DeploymentFileGroup,
	userDataSecrets map[string]models_deployments.DeploymentSecret,
	mergedFiles map[string][]byte,
) (bindMountDataCollection, error) {
	var bindMounts bindMountDataCollection
	var err error
	bindMounts.Files, err = createFiles(deploymentId, deploymentFilesDirName, mergedFiles, h.config.WorkDirPath)
	if err != nil {
		return bindMountDataCollection{}, err
	}
	bindMounts.FileGroups, err = createFileGroups(deploymentFilesDirName, userDataFileGroups, h.config.WorkDirPath)
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
	err := createDeploymentDir(moduleFileSystem, h.config.WorkDirPath, deploymentDirName)
	if err != nil {
		return err
	}
	err = createFilesDir(h.config.WorkDirPath, deploymentFilesDirName)
	if err != nil {
		return err
	}
	return err
}

func (h *Handler) filterSelectedModules(
	ctx context.Context,
	selectedModules map[string]models_module.Module,
) (map[string]models_module.Module, error) {
	deployments, err := h.databaseHandler.ReadDeployments(ctx, models_deployments.DeploymentsFilter{
		ModuleIds: slices.Collect(maps.Keys(selectedModules)),
	})
	if err != nil {
		return nil, err
	}
	deployments = helper_maps.CollectFunc(maps.Values(deployments), func(value models_deployments.DeploymentBase) string {
		return value.ModuleId
	})
	filteredModules := make(map[string]models_module.Module)
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

func getDefaultData(module models_module.Module) (defaultDataCollection, error) {
	var data defaultDataCollection
	var err error
	data.Files, err = getDefaultFiles(module.Files, module.FileSystem)
	if err != nil {
		return defaultDataCollection{}, err
	}
	data.Configs, err = getDefaultConfigs(module.Configs)
	if err != nil {
		return defaultDataCollection{}, err
	}
	return data, nil
}

func getUserData(
	module models_module.Module,
	defaultData defaultDataCollection,
	userInput models_deployments.UserInput,
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
	data.Configs, err = getProvidedConfigs(module.Configs, defaultData.Configs, userInput.Configs, deploymentId)
	if err != nil {
		return userDataCollection{}, err
	}
	data.Files = getProvidedFiles(module.Files, defaultData.Files, userInput.Files, deploymentId)
	data.FileGroups = getProvidedFileGroups(module.FileGroups, userInput.FileGroups, deploymentId)
	return data, nil
}

func getDeployment(
	module models_module.Module,
	deploymentId string,
) (models_deployments.DeploymentBase, error) {
	if deploymentId == "" {
		return models_deployments.DeploymentBase{}, errors.New("empty deployment id")
	}
	dirName, err := helper_uuid.New()
	if err != nil {
		return models_deployments.DeploymentBase{}, err
	}
	return models_deployments.DeploymentBase{
		Id:            deploymentId,
		ModuleId:      module.ID,
		ModuleSource:  module.Source,
		ModuleChannel: module.Channel,
		ModuleVersion: module.Version,
		DirName:       dirName,
		FilesDirName:  dirName + "_files",
	}, nil
}

func initDeploymentsCacheFromModules(modules map[string]models_module.Module) (map[string]deploymentsCacheItem, error) {
	cache := make(map[string]deploymentsCacheItem)
	for moduleId, module := range modules {
		id, err := helper_uuid.New()
		if err != nil {
			return nil, err
		}
		containers := make(map[string]containerCacheItem)
		for reference := range module.Services {
			name, err := helper_naming.NewContainerName(models_constants.DeploymentAbbreviation)
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

func getNewVolumes(moduleVolumes map[string]struct{}, deploymentId string) map[string]models_deployments.DeploymentVolume {
	volumes := make(map[string]models_deployments.DeploymentVolume)
	for reference := range moduleVolumes {
		volumes[reference] = models_deployments.DeploymentVolume{
			DeploymentId: deploymentId,
			Reference:    reference,
			Name:         helper_naming.NewVolumeName(models_constants.DeploymentAbbreviation, deploymentId, reference),
		}
	}
	return volumes
}

func getNewContainers(
	moduleServices map[string]models_external.ModuleLibService,
	cacheContainers map[string]containerCacheItem,
	deploymentId string,
) (map[string]models_deployments.ContainerBase, error) {
	containers := make(map[string]models_deployments.ContainerBase)
	for reference := range moduleServices {
		cacheItem, ok := cacheContainers[reference]
		if !ok {
			return nil, errors.New("missing container alias")
		}
		containers[reference] = models_deployments.ContainerBase{
			Name:         cacheItem.Name,
			DeploymentId: deploymentId,
			Reference:    reference,
			Alias:        cacheItem.Alias,
		}
	}
	return containers, nil
}
