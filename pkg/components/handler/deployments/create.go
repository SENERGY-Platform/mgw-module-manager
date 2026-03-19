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
	cache, err := initCache(selectedModules)
	if err != nil {
		return err
	}
	var errs []string
	for moduleId, module := range selectedModules {
		userInput := userInputs[moduleId]
		deployment, err := getExtendedDeployment(module, userInput, cache)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.storageHdl.CreateDeployment(
			ctx,
			deployment.Deployment,
			slices.Collect(maps.Values(deployment.UserData.HostResources)),
			slices.Collect(maps.Values(deployment.UserData.Secrets)),
			slices.Collect(maps.Values(deployment.UserData.Configs)),
			slices.Collect(maps.Values(deployment.UserData.GlobalConfigs)),
			slices.Collect(maps.Values(deployment.UserData.Files)),
			slices.Collect(maps.Values(deployment.UserData.FileGroups)),
			slices.Collect(maps.Values(deployment.Volumes)),
			slices.Collect(maps.Values(deployment.Containers)),
		)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.updateCaches(ctx, module.Dependencies, deployment.UserData, cache)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		containerEnvironmentData, err := h.ensureContainerEnvironment(ctx, module, deployment)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		// TODO "mount secrets" must be "unloaded" if one of the following steps fails
		err = h.createHttpEndpoints(ctx, module.Services, moduleId, deployment.Containers)
		if err != nil {
			errs = append(errs, err.Error())
		}
		createdContainers, err := h.createContainers(ctx, module.Services, deployment, containerEnvironmentData, cache)
		if err != nil {
			errs = append(errs, err.Error())
		}
		err = h.storageHdl.UpdateDeploymentContainerIds(ctx, createdContainers)
		if err != nil {
			// TODO how to handle already created containers?
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (h *Handler) ensureContainerEnvironment(
	ctx context.Context,
	module models_handler_module.Module,
	deployment extendedDeployment,
) (containerEnvironmentDataCollection, error) {
	var data containerEnvironmentDataCollection
	var err error
	err = h.ensureContainerImages(ctx, module.Services)
	if err != nil {
		return containerEnvironmentDataCollection{}, err
	}
	err = h.ensureContainerVolumes(ctx, deployment)
	if err != nil {
		return containerEnvironmentDataCollection{}, err
	}
	err = h.createDeploymentDir(module.FileSystem, deployment.DirName)
	if err != nil {
		return containerEnvironmentDataCollection{}, err
	}
	err = h.createFilesDir(deployment.FilesDirName)
	if err != nil {
		return containerEnvironmentDataCollection{}, err
	}
	data.FileMounts, err = createFiles(deployment.Id, deployment.FilesDirName, deployment.MergedFiles, h.config.WorkDirPath)
	if err != nil {
		return containerEnvironmentDataCollection{}, err
	}
	data.FileGroupMounts, err = createFileGroups(deployment.FilesDirName, deployment.UserData.FileGroups, h.config.WorkDirPath)
	if err != nil {
		return containerEnvironmentDataCollection{}, err
	}
	data.SecretMounts, err = h.createSecretMounts(ctx, deployment.Id, deployment.UserData.Secrets)
	if err != nil {
		return containerEnvironmentDataCollection{}, err
	}
	data.Configs = configsToStrings(module.Configs, deployment.MergedConfigs)
	return data, nil
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

func (h *Handler) updateCaches(ctx context.Context, moduleDependencies map[string]string, userData userDataCollection, cache cacheCollection) error {
	err := h.updateContainerAliasesCache(ctx, moduleDependencies, cache.ContainerAliases)
	if err != nil {
		return err
	}
	err = h.updateGlobalConfigsCache(ctx, userData.GlobalConfigs, cache.GlobalConfigs)
	if err != nil {
		return err
	}
	err = h.updateHostResourcesCache(ctx, userData.HostResources, cache.HostResources)
	if err != nil {
		return err
	}
	return h.updateSecretValuesCache(ctx, userData.Secrets, cache.SecretValues)
}

func getExtendedDeployment(
	module models_handler_module.Module,
	userInput models_handler_deployment.UserInput,
	cache cacheCollection,
) (extendedDeployment, error) {
	id := cache.DeploymentIds[module.ID]
	name := module.Name
	if userInput.Name != "" {
		name = userInput.Name
	}
	dirName, err := helper_uuid.New()
	if err != nil {
		return extendedDeployment{}, err
	}
	userData, mergedConfigs, mergedFiles, err := getDeploymentData(module, userInput, cache, id)
	if err != nil {
		return extendedDeployment{}, err
	}
	return extendedDeployment{
		Deployment: models_handler_storage.Deployment{
			Id:            id,
			ModuleId:      module.ID,
			ModuleSource:  module.Source,
			ModuleChannel: module.Channel,
			ModuleVersion: module.Version,
			Name:          name,
			DirName:       dirName,
			FilesDirName:  dirName + "_files",
			Created:       helper_time.Now(),
		},
		UserData:      userData,
		Containers:    newContainers2(module.Services, cache.ContainerAliases[module.ID], id),
		Volumes:       newVolumes(module.Volumes, id),
		MergedConfigs: mergedConfigs,
		MergedFiles:   mergedFiles,
	}, nil
}

func getDeploymentData(
	module models_handler_module.Module,
	userInput models_handler_deployment.UserInput,
	cache cacheCollection,
	deploymentId string,
) (userDataCollection, map[string]models_handler_storage.Config, map[string][]byte, error) {
	defaultData, err := getDefaultData(module)
	if err != nil {
		return userDataCollection{}, nil, nil, err
	}
	userData, err := getUserData(module, defaultData, userInput, deploymentId)
	if err != nil {
		return userDataCollection{}, nil, nil, err
	}
	mergedConfigs := mergeConfigs(defaultData.Configs, userData.Configs, userData.GlobalConfigs, cache.GlobalConfigs)
	err = checkConfigs(module.Configs, mergedConfigs)
	if err != nil {
		return userDataCollection{}, nil, nil, err
	}
	mergedFiles := mergeFiles(defaultData.Files, userData.Files)
	err = checkFiles(module.Files, mergedFiles)
	if err != nil {
		return userDataCollection{}, nil, nil, err
	}
	return userData, mergedConfigs, mergedFiles, nil
}

func initCache(modules map[string]models_handler_module.Module) (cacheCollection, error) {
	deploymentIds := make(map[string]string)
	containerAliases := make(map[string]map[string]string)
	for moduleId, module := range modules {
		id, err := helper_uuid.New()
		if err != nil {
			return cacheCollection{}, err
		}
		deploymentIds[moduleId] = id
		aliases := make(map[string]string)
		for reference := range module.Services {
			aliases[reference] = helper_naming.NewContainerAlias(id, reference)
		}
		containerAliases[moduleId] = aliases
	}
	return cacheCollection{
		HostResources:    make(map[string]models_external.HostResource),
		GlobalConfigs:    make(map[string]models_handler_storage.GlobalConfig),
		SecretValues:     make(map[string]models_external.SecretValueVariant),
		DeploymentIds:    deploymentIds,
		ContainerAliases: containerAliases,
	}, nil
}

func (h *Handler) createDeploymentDir(moduleFileSystem fs.FS, deploymentDirName string) error {
	dirPath := path.Join(h.config.WorkDirPath, deploymentDirName)
	err := os.Mkdir(dirPath, dirPerm)
	if err != nil {
		return err
	}
	return helper_file_sys.CopyAll(moduleFileSystem, dirPath)
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
) map[string]models_handler_storage.DeploymentContainer {
	containers := make(map[string]models_handler_storage.DeploymentContainer)
	for reference := range moduleServices {
		containers[reference] = models_handler_storage.DeploymentContainer{
			DeploymentId: deploymentId,
			Reference:    reference,
			Alias:        containerAliases[reference],
		}
	}
	return containers
}
