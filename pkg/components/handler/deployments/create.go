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
	"io/fs"
	"maps"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	helper_file_sys "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/file_sys"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	helper_uuid "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/uuid"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_deployment "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployment"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) CreateDeployments(
	ctx context.Context,
	selectedModules map[string]models_handler_module.Module,
	userInputs map[string]models_handler_deployment.UserInput,
) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	cache := cacheCollection{
		HostResources: make(map[string]models_external.HostResource),
		GlobalConfigs: make(map[string]models_handler_storage.GlobalConfig),
		SecretValues:  make(map[string]models_external.SecretValueVariant),
	}
	var err error
	cache.Deployments, err = initDeploymentsCacheFromModules(selectedModules)
	if err != nil {
		return err
	}
	timestamp := helper_time.Now()
	var errs []string
	for moduleId, module := range selectedModules {
		cacheItem, ok := cache.Deployments[moduleId]
		if !ok {
			errs = append(errs, "module "+moduleId+" not deployed")
			continue
		}
		err = h.createDeployment(
			ctx,
			module,
			userInputs[moduleId],
			cacheItem.DeploymentId,
			cacheItem.ContainerAliases,
			timestamp,
			cache,
		)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (h *Handler) createDeployment(
	ctx context.Context,
	module models_handler_module.Module,
	userInput models_handler_deployment.UserInput,
	deploymentId string,
	containerAliases map[string]string,
	timestamp time.Time,
	cache cacheCollection,
) error {
	newDeployment, err := getDeployment(module, userInput.Name, deploymentId)
	if err != nil {
		return err
	}
	newDeployment.Created = timestamp
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
	newContainers, err := getNewContainers(module.Services, containerAliases, deploymentId)
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
	bindMounts, err := h.getBindMounts(
		ctx,
		deploymentId,
		newDeployment.FilesDirName,
		userData.FileGroups,
		userData.Secrets,
		mergedFiles,
	)
	// TODO "mount secrets" must be "unloaded" if one of the following steps fail
	err = h.createHttpEndpoints(ctx, module.Services, module.ID, newContainers)
	if err != nil {
		// TODO log error?
	}
	createdContainers, err := h.createContainers(
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
		// TODO log error?
	}
	err = h.databaseHandler.UpdateDeploymentContainerIds(ctx, createdContainers)
	if err != nil {
		// TODO how to handle already created containers?
		// TODO log error?
	}
	return nil
}

func (h *Handler) getBindMounts(
	ctx context.Context,
	deploymentId,
	deploymentFilesDirName string,
	userDataFileGroups map[string]models_handler_storage.DeploymentFileGroup,
	userDataSecrets map[string]models_handler_storage.DeploymentSecret,
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

func createDeploymentDir(moduleFileSystem fs.FS, workDirPath, deploymentDirName string) error {
	dirPath := path.Join(workDirPath, deploymentDirName)
	err := os.Mkdir(dirPath, dirPerm)
	if err != nil {
		return err
	}
	return helper_file_sys.CopyAll(moduleFileSystem, dirPath)
}

func getDefaultData(module models_handler_module.Module) (defaultDataCollection, error) {
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
	module models_handler_module.Module,
	defaultData defaultDataCollection,
	userInput models_handler_deployment.UserInput,
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
	module models_handler_module.Module,
	userInputName string,
	deploymentId string,
) (models_handler_storage.Deployment, error) {
	if deploymentId == "" {
		return models_handler_storage.Deployment{}, errors.New("empty deployment id")
	}
	name := module.Name
	if userInputName != "" {
		name = userInputName
	}
	dirName, err := helper_uuid.New()
	if err != nil {
		return models_handler_storage.Deployment{}, err
	}
	return models_handler_storage.Deployment{
		Id:            deploymentId,
		ModuleId:      module.ID,
		ModuleSource:  module.Source,
		ModuleChannel: module.Channel,
		ModuleVersion: module.Version,
		Name:          name,
		DirName:       dirName,
		FilesDirName:  dirName + "_files",
	}, nil
}

func initDeploymentsCacheFromModules(modules map[string]models_handler_module.Module) (map[string]deploymentsCacheItem, error) {
	cache := make(map[string]deploymentsCacheItem)
	for moduleId, module := range modules {
		id, err := helper_uuid.New()
		if err != nil {
			return nil, err
		}
		aliases := make(map[string]string)
		for reference := range module.Services {
			aliases[reference] = helper_naming.NewContainerAlias(id, reference)
		}
		cache[moduleId] = deploymentsCacheItem{
			DeploymentId:     id,
			ContainerAliases: aliases,
		}
	}
	return cache, nil
}

func getNewVolumes(moduleVolumes map[string]struct{}, deploymentId string) map[string]models_handler_storage.DeploymentVolume {
	volumes := make(map[string]models_handler_storage.DeploymentVolume)
	for reference := range moduleVolumes {
		volumes[reference] = models_handler_storage.DeploymentVolume{
			DeploymentId: deploymentId,
			Reference:    reference,
			Name:         helper_naming.NewVolumeName(deploymentId, reference),
		}
	}
	return volumes
}

func getNewContainers(
	moduleServices map[string]models_external.ModuleLibService,
	containerAliases map[string]string,
	deploymentId string,
) (map[string]models_handler_storage.DeploymentContainer, error) {
	containers := make(map[string]models_handler_storage.DeploymentContainer)
	for reference := range moduleServices {
		alias, ok := containerAliases[reference]
		if !ok {
			return nil, errors.New("missing container alias")
		}
		containers[reference] = models_handler_storage.DeploymentContainer{
			DeploymentId: deploymentId,
			Reference:    reference,
			Alias:        alias,
		}
	}
	return containers, nil
}
