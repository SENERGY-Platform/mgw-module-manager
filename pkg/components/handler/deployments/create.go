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
		defaultData, err := getDefaultData(module)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		userData, err := getUserData(module, deployment, defaultData, userInput)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		mergedConfigs := mergeConfigs(defaultData, userData, cache)
		err = checkConfigs(module, mergedConfigs)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		mergedFiles := mergeFiles(defaultData, userData)
		err = checkFiles(module, mergedFiles)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.storageHdl.CreateDeployment(
			ctx,
			deployment.Deployment,
			slices.Collect(maps.Values(userData.HostResources)),
			slices.Collect(maps.Values(userData.Secrets)),
			slices.Collect(maps.Values(userData.Configs)),
			slices.Collect(maps.Values(userData.GlobalConfigs)),
			slices.Collect(maps.Values(userData.Files)),
			slices.Collect(maps.Values(userData.FileGroups)),
			slices.Collect(maps.Values(deployment.Volumes)),
			slices.Collect(maps.Values(deployment.Containers)),
		)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.updateCaches(ctx, module, userData, cache)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		containerData, err := h.initContainerEnvironment(ctx, module, deployment, userData, mergedConfigs, mergedFiles)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.createHttpEndpoints(ctx, module, deployment)
		if err != nil {
			errs = append(errs, err.Error())
		}
		createdContainers, err := h.createContainers(ctx, module, deployment, userData, containerData, cache)
		if err != nil {
			errs = append(errs, err.Error())
		}
		err = h.storageHdl.UpdateDeploymentContainerIds(ctx, createdContainers)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (h *Handler) initContainerEnvironment(
	ctx context.Context,
	module models_handler_module.Module,
	deployment extendedDeployment,
	userData userDataCollection,
	mergedConfigs map[string]models_handler_storage.Config,
	mergedFiles map[string][]byte,
) (containerDataCollection, error) {
	var data containerDataCollection
	var err error
	data.SecretMounts, err = h.getSecretMounts(ctx, deployment, userData) // secrets must be "unloaded" if error
	if err != nil {
		return containerDataCollection{}, err
	}
	data.Configs = configsToStrings(module, mergedConfigs)
	err = h.addDeploymentContainerImages(ctx, module) // ensure that container images are available
	if err != nil {
		return containerDataCollection{}, err
	}
	err = h.createDeploymentContainerVolumes(ctx, deployment)
	if err != nil {
		return containerDataCollection{}, err
	}
	err = h.createDeploymentDir(module, deployment)
	if err != nil {
		return containerDataCollection{}, err
	}
	err = h.createFilesDir(deployment)
	if err != nil {
		return containerDataCollection{}, err
	}
	data.FileMounts, err = h.createFiles(deployment, mergedFiles)
	if err != nil {
		return containerDataCollection{}, err
	}
	data.FileGroupMounts, err = h.createFileGroups(deployment, userData)
	if err != nil {
		return containerDataCollection{}, err
	}
	return data, nil
}

func getDefaultData(module models_handler_module.Module) (defaultDataCollection, error) {
	var data defaultDataCollection
	var err error
	data.Files, err = getDefaultFiles(module)
	if err != nil {
		return defaultDataCollection{}, err
	}
	data.Configs, err = getDefaultConfigs(module)
	if err != nil {
		return defaultDataCollection{}, err
	}
	return data, nil
}

func getUserData(
	module models_handler_module.Module,
	deployment extendedDeployment,
	defaultData defaultDataCollection,
	userInput models_handler_deployment.UserInput,
) (userDataCollection, error) {
	var data userDataCollection
	var err error
	data.GlobalConfigs = getSelectedGlobalConfigs(module, userInput, deployment.Id)
	data.HostResources, err = getSelectedHostResources(module, userInput, deployment.Id)
	if err != nil {
		return userDataCollection{}, err
	}
	data.Secrets, err = getSelectedSecrets(module, userInput, deployment.Id)
	if err != nil {
		return userDataCollection{}, err
	}
	data.Configs, err = getProvidedConfigs(module, defaultData, userInput, deployment.Id)
	if err != nil {
		return userDataCollection{}, err
	}
	data.Files = getProvidedFiles(module, defaultData, userInput, deployment.Id)
	data.FileGroups = getProvidedFileGroups(module, userInput, deployment.Id)
	return data, nil
}

func (h *Handler) updateCaches(ctx context.Context, module models_handler_module.Module, userData userDataCollection, cache cacheCollection) error {
	err := h.updateContainerAliasesCache(ctx, module, cache)
	if err != nil {
		return err
	}
	err = h.updateGlobalConfigsCache(ctx, userData, cache)
	if err != nil {
		return err
	}
	err = h.updateHostResourcesCache(ctx, userData, cache)
	if err != nil {
		return err
	}
	return h.updateSecretValuesCache(ctx, userData, cache)
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
		Containers: newContainers2(module, cache.ContainerAliases[module.ID], id),
		Volumes:    newVolumes(module, id),
	}, nil
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

func (h *Handler) createDeploymentDir(module models_handler_module.Module, deployment extendedDeployment) error {
	dirPath := path.Join(h.config.WorkDirPath, deployment.DirName)
	err := os.Mkdir(dirPath, dirPerm)
	if err != nil {
		return err
	}
	return helper_file_sys.CopyAll(module.FileSystem, dirPath)
}

func newVolumes(module models_handler_module.Module, deploymentId string) map[string]models_handler_storage.DeploymentVolume {
	volumes := make(map[string]models_handler_storage.DeploymentVolume)
	for reference := range module.Volumes {
		volumes[reference] = models_handler_storage.DeploymentVolume{
			DeploymentId: deploymentId,
			Reference:    reference,
			Name:         helper_naming.NewVolumeName(deploymentId, reference),
		}
	}
	return volumes
}

func newContainers2(
	module models_handler_module.Module,
	containerAliases map[string]string,
	deploymentId string,
) map[string]models_handler_storage.DeploymentContainer {
	containers := make(map[string]models_handler_storage.DeploymentContainer)
	for reference := range module.Services {
		containers[reference] = models_handler_storage.DeploymentContainer{
			DeploymentId: deploymentId,
			Reference:    reference,
			Alias:        containerAliases[reference],
		}
	}
	return containers
}
