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
	"maps"
	"os"
	"path"
	"slices"
	"strings"

	helper_maps "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/maps"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_deployment "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployment"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) UpdateDeployments(
	ctx context.Context,
	selectedModules map[string]models_handler_module.Module,
	userInputs map[string]models_handler_deployment.UserInput,
) error {
	currentDeployments, err := h.storageHdl.ReadDeployments(ctx, models_handler_storage.DeploymentsFilter{
		ModuleIds: slices.Collect(maps.Keys(selectedModules)),
	})
	if err != nil {
		return err
	}
	currentDeploymentIds := slices.Collect(maps.Keys(currentDeployments))
	currentDeploymentsContainers, err := h.storageHdl.ReadDeploymentsContainers(ctx, currentDeploymentIds)
	if err != nil {
		return err
	}
	currentDeploymentsVolumes, err := h.storageHdl.ReadDeploymentsVolumes(ctx, currentDeploymentIds)
	if err != nil {
		return err
	}
	// map deployments to module IDs
	currentDeployments = helper_maps.CollectFunc(maps.Values(currentDeployments), func(value models_handler_storage.Deployment) string {
		return value.ModuleId
	})
	cacheHostResources := make(map[string]models_external.HostResource)
	cacheGlobalConfigs := make(map[string]models_handler_storage.GlobalConfig)
	cacheSecretValues := make(map[string]models_external.SecretValueVariant)
	cacheDeployments := initDeploymentsCacheFromModulesAndDeployments(selectedModules, currentDeployments, currentDeploymentsContainers)
	var errs []string
	for moduleId, module := range selectedModules {
		cacheItem, ok := cacheDeployments[moduleId]
		if !ok {
			errs = append(errs, "module "+moduleId+" not deployed") // TODO
			continue
		}
		// prepare new deployment with user input
		userInput := userInputs[moduleId]
		newDeployment, err := getDeployment(module, userInput.Name, cacheItem.DeploymentId)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		defaultData, err := getDefaultData(module)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		userData, err := getUserData(module, defaultData, userInput, newDeployment.Id)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.updateGlobalConfigsCache(ctx, userData.GlobalConfigs, cacheGlobalConfigs)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		mergedConfigs := mergeConfigs(defaultData.Configs, userData.Configs, userData.GlobalConfigs, cacheGlobalConfigs)
		err = checkConfigs(module.Configs, mergedConfigs)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		mergedFiles := mergeFiles(defaultData.Files, userData.Files)
		err = checkFiles(module.Files, mergedFiles)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		containers, err := newContainers2(module.Services, cacheItem.ContainerAliases, newDeployment.Id)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		currentDeployment := currentDeployments[moduleId]
		currentVolumes := currentDeploymentsVolumes[currentDeployment.Id]
		volumes := updateVolumes(module.Volumes, currentVolumes, newDeployment.Id)
		// remove containers, unmount secrets and remove deployment dirs
		currentContainers := currentDeploymentsContainers[currentDeployment.Id]
		err = h.removeContainers(ctx, currentContainers)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.removeSecretMounts(ctx, currentDeployment.Id)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.removeDeploymentDirs(currentDeployment.DirName, currentDeployment.FilesDirName)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		// update deployment in db
		err = h.storageHdl.UpdateDeployment(
			ctx,
			newDeployment,
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
			errs = append(errs, err.Error())
			continue
		}
		// update caches
		err = h.updateDeploymentsCache(ctx, module.Dependencies, cacheDeployments)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.updateHostResourcesCache(ctx, userData.HostResources, cacheHostResources)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.updateSecretValuesCache(ctx, userData.Secrets, cacheSecretValues)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.ensureContainerImages(ctx, module.Services)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.ensureContainerVolumes(ctx, volumes, newDeployment.Id)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.createDeploymentDirs(module.FileSystem, newDeployment.DirName, newDeployment.FilesDirName)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		bindMounts, err := h.getBindMounts(
			ctx,
			newDeployment.Id,
			newDeployment.FilesDirName,
			userData.FileGroups,
			userData.Secrets,
			mergedFiles,
		)
		// TODO "mount secrets" must be "unloaded" if one of the following steps fail
		err = h.createHttpEndpoints(ctx, module.Services, moduleId, containers)
		if err != nil {
			errs = append(errs, err.Error())
		}
		createdContainers, err := h.createContainers(
			ctx,
			module.Configs,
			module.Services,
			newDeployment.Id,
			newDeployment.DirName,
			newDeployment.FilesDirName,
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

// provided deployment map must use module IDs as keys
func initDeploymentsCacheFromModulesAndDeployments(
	modules map[string]models_handler_module.Module,
	deployments map[string]models_handler_storage.Deployment,
	deploymentsContainers map[string]map[string]models_handler_storage.DeploymentContainer,
) map[string]deploymentsCacheItem {
	cache := make(map[string]deploymentsCacheItem)
	for moduleId, module := range modules {
		deployment, ok := deployments[moduleId]
		if !ok {
			continue
		}
		aliases := make(map[string]string)
		for reference := range module.Services {
			existingContainer := deploymentsContainers[deployment.Id][reference]
			alias := existingContainer.Alias
			if alias == "" {
				alias = helper_naming.NewContainerAlias(deployment.Id, reference)
			}
			aliases[reference] = alias
		}
		cache[moduleId] = deploymentsCacheItem{
			DeploymentId:     deployment.Id,
			ContainerAliases: aliases,
		}
	}
	return cache
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
	deploymentVolumes map[string]models_handler_storage.DeploymentVolume,
	deploymentId string,
) map[string]models_handler_storage.DeploymentVolume {
	volumes := make(map[string]models_handler_storage.DeploymentVolume)
	for reference := range moduleVolumes {
		volume := deploymentVolumes[reference]
		name := volume.Name
		if name == "" {
			name = helper_naming.NewVolumeName(deploymentId, reference)
		}
		volumes[reference] = models_handler_storage.DeploymentVolume{
			DeploymentId: deploymentId,
			Reference:    reference,
			Name:         name,
		}
	}
	return volumes
}
