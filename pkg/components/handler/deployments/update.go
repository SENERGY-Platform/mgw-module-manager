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
	h.mu.Lock()
	defer h.mu.Unlock()
	deployments, err := h.storageHdl.ReadDeployments(ctx, models_handler_storage.DeploymentsFilter{
		ModuleIds: slices.Collect(maps.Keys(selectedModules)),
	})
	if err != nil {
		return err
	}
	deploymentIds := slices.Collect(maps.Keys(deployments))
	deploymentsContainers, err := h.storageHdl.ReadDeploymentsContainers(ctx, deploymentIds)
	if err != nil {
		return err
	}
	deploymentsVolumes, err := h.storageHdl.ReadDeploymentsVolumes(ctx, deploymentIds)
	if err != nil {
		return err
	}
	cacheHostResources := make(map[string]models_external.HostResource)
	cacheGlobalConfigs := make(map[string]models_handler_storage.GlobalConfig)
	cacheSecretValues := make(map[string]models_external.SecretValueVariant)
	cacheDeployments := initDeploymentsCacheFromModulesAndDeployments(selectedModules, deployments, deploymentsContainers)
	var errs []string
	for moduleId, module := range selectedModules {
		cacheItem, ok := cacheDeployments[moduleId]
		if !ok {
			errs = append(errs, "module "+moduleId+" not deployed")
			continue
		}
		err = h.updateDeployment(
			ctx,
			module,
			userInputs[moduleId],
			cacheItem.DeploymentId,
			cacheItem.ContainerAliases,
			deployments[cacheItem.DeploymentId],
			deploymentsContainers[cacheItem.DeploymentId],
			deploymentsVolumes[cacheItem.DeploymentId],
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

func (h *Handler) updateDeployment(
	ctx context.Context,
	module models_handler_module.Module,
	userInput models_handler_deployment.UserInput,
	deploymentId string,
	containerAliases map[string]string,
	currentDeployment models_handler_storage.Deployment,
	currentDeploymentContainers map[string]models_handler_storage.DeploymentContainer,
	currentDeploymentVolumes map[string]models_handler_storage.DeploymentVolume,
	cacheHostResources map[string]models_external.HostResource,
	cacheGlobalConfigs map[string]models_handler_storage.GlobalConfig,
	cacheSecretValues map[string]models_external.SecretValueVariant,
	cacheDeployments map[string]deploymentsCacheItem,
) error {
	newDeployment, err := getDeployment(module, userInput.Name, deploymentId)
	if err != nil {
		return err
	}
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
		cacheHostResources,
		cacheGlobalConfigs,
		cacheSecretValues,
		cacheDeployments,
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
		cacheGlobalConfigs,
	)
	if err != nil {
		return err
	}
	containers, err := newContainers2(module.Services, containerAliases, deploymentId)
	if err != nil {
		return err
	}
	volumes := updateVolumes(module.Volumes, currentDeploymentVolumes, deploymentId)
	err = h.removeContainers(ctx, currentDeploymentContainers)
	if err != nil {
		return err
	}
	err = h.removeSecretMounts(ctx, deploymentId)
	if err != nil {
		return err
	}
	err = h.removeDeploymentDirs(currentDeployment.DirName, currentDeployment.FilesDirName)
	if err != nil {
		return err
	}
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
		return err
	}
	err = h.ensureContainerImages(ctx, module.Services)
	if err != nil {
		return err
	}
	err = h.ensureContainerVolumes(ctx, volumes, deploymentId)
	if err != nil {
		return err
	}
	err = h.createDeploymentDirs(module.FileSystem, newDeployment.DirName, newDeployment.FilesDirName)
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
	// TODO "mount secrets" must be "unloaded" if one of the following steps fail
	err = h.createHttpEndpoints(ctx, module.Services, module.ID, containers)
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

func initDeploymentsCacheFromModulesAndDeployments(
	modules map[string]models_handler_module.Module,
	deployments map[string]models_handler_storage.Deployment,
	deploymentsContainers map[string]map[string]models_handler_storage.DeploymentContainer,
) map[string]deploymentsCacheItem {
	cache := make(map[string]deploymentsCacheItem)
	deployments = helper_maps.CollectFunc(maps.Values(deployments), func(value models_handler_storage.Deployment) string {
		return value.ModuleId
	})
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
