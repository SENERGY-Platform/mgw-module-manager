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
	cacheHostResources := make(map[string]models_external.HostResource)
	cacheGlobalConfigs := make(map[string]models_handler_storage.GlobalConfig)
	cacheSecretValues := make(map[string]models_external.SecretValueVariant)
	cacheDeployments, err := initDeploymentsCacheFromModules(selectedModules)
	if err != nil {
		return err
	}
	var errs []string
	for moduleId, module := range selectedModules {
		cacheItem, ok := cacheDeployments[moduleId]
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
			cacheHostResources,
			cacheGlobalConfigs,
			cacheSecretValues,
			cacheDeployments,
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
	cacheHostResources map[string]models_external.HostResource,
	cacheGlobalConfigs map[string]models_handler_storage.GlobalConfig,
	cacheSecretValues map[string]models_external.SecretValueVariant,
	cacheDeployments map[string]deploymentsCacheItem,
) error {
	deployment, err := getDeployment(module, userInput.Name, deploymentId)
	if err != nil {
		return err
	}
	defaultData, err := getDefaultData(module)
	if err != nil {
		return err
	}
	userData, err := getUserData(module, defaultData, userInput, deployment.Id)
	if err != nil {
		return err
	}
	err = h.updateGlobalConfigsCache(ctx, userData.GlobalConfigs, cacheGlobalConfigs)
	if err != nil {
		return err
	}
	mergedConfigs := mergeConfigs(defaultData.Configs, userData.Configs, userData.GlobalConfigs, cacheGlobalConfigs)
	err = checkConfigs(module.Configs, mergedConfigs)
	if err != nil {
		return err
	}
	mergedFiles := mergeFiles(defaultData.Files, userData.Files)
	err = checkFiles(module.Files, mergedFiles)
	if err != nil {
		return err
	}
	containers, err := newContainers2(module.Services, containerAliases, deployment.Id)
	if err != nil {
		return err
	}
	volumes := newVolumes(module.Volumes, deployment.Id)
	err = h.storageHdl.CreateDeployment(
		ctx,
		deployment,
		slices.Collect(maps.Values(userData.HostResources)),
		slices.Collect(maps.Values(userData.Secrets)),
		slices.Collect(maps.Values(userData.Configs)),
		slices.Collect(maps.Values(userData.GlobalConfigs)),
		slices.Collect(maps.Values(userData.Files)),
		slices.Collect(maps.Values(userData.FileGroups)),
		slices.Collect(maps.Values(volumes)),
		slices.Collect(maps.Values(containers)),
	)
	if err != nil {
		return err
	}
	err = h.updateDeploymentsCache(ctx, module.Dependencies, cacheDeployments)
	if err != nil {
		return err
	}
	err = h.updateHostResourcesCache(ctx, userData.HostResources, cacheHostResources)
	if err != nil {
		return err
	}
	err = h.updateSecretValuesCache(ctx, userData.Secrets, cacheSecretValues)
	if err != nil {
		return err
	}
	err = h.ensureContainerImages(ctx, module.Services)
	if err != nil {
		return err
	}
	err = h.ensureContainerVolumes(ctx, volumes, deployment.Id)
	if err != nil {
		return err
	}
	err = h.createDeploymentDirs(module.FileSystem, deployment.DirName, deployment.FilesDirName)
	if err != nil {
		return err
	}
	bindMounts, err := h.getBindMounts(
		ctx,
		deployment.Id,
		deployment.FilesDirName,
		userData.FileGroups,
		userData.Secrets,
		mergedFiles,
	)
	// TODO "mount secrets" must be "unloaded" if one of the following steps fail
	err = h.createHttpEndpoints(ctx, module.Services, module.ID, containers)
	if err != nil {
		// TODO log error?
	}
	createdContainers, err := h.createContainers(
		ctx,
		module.Configs,
		module.Services,
		deployment.Id,
		deployment.DirName,
		deployment.FilesDirName,
		userData.Secrets,
		userData.HostResources,
		containers,
		volumes,
		mergedConfigs,
		bindMounts,
		cacheSecretValues,
		cacheDeployments,
		cacheHostResources,
	)
	if err != nil {
		// TODO log error?
	}
	err = h.storageHdl.UpdateDeploymentContainerIds(ctx, createdContainers)
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
		Created:       helper_time.Now(),
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

func newVolumes(moduleVolumes map[string]struct{}, deploymentId string) map[string]models_handler_storage.DeploymentVolume {
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

func newContainers2(
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
