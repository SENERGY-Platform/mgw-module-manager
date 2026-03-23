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
	"io/fs"
	"maps"
	"slices"
	"strings"

	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) RecreateDeployments(ctx context.Context, selectedModules map[string]models_handler_module.Module) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	deployments, err := h.storageHdl.ReadDeployments(ctx, models_handler_storage.DeploymentsFilter{
		ModuleIds: slices.Collect(maps.Keys(selectedModules)),
	})
	if err != nil {
		return err
	}
	deploymentIds := slices.Collect(maps.Keys(deployments))
	deploymentsUserData, err := h.getDeploymentsUserDataFromDB(ctx, deploymentIds)
	if err != nil {
		return err
	}
	deploymentsVolumes, deploymentsContainers, err := h.getDeploymentsVolumesAndContainersFromDB(ctx, deploymentIds)
	if err != nil {
		return err
	}
	cache := cacheCollection{
		HostResources: make(map[string]models_external.HostResource),
		GlobalConfigs: make(map[string]models_handler_storage.GlobalConfig),
		SecretValues:  make(map[string]models_external.SecretValueVariant),
		Deployments:   initDeploymentsCacheFromModulesAndDeployments(selectedModules, deployments, deploymentsContainers),
	}
	var errs []string
	for moduleId, module := range selectedModules {
		cacheItem, ok := cache.Deployments[moduleId]
		if !ok {
			errs = append(errs, "module "+moduleId+" not deployed") // TODO
			continue
		}
		err = h.recreateDeployment(
			ctx,
			module,
			deploymentsUserData[cacheItem.DeploymentId],
			cacheItem.DeploymentId,
			cacheItem.ContainerAliases,
			deployments[cacheItem.DeploymentId],
			deploymentsContainers[cacheItem.DeploymentId],
			deploymentsVolumes[cacheItem.DeploymentId],
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

func (h *Handler) recreateDeployment(
	ctx context.Context,
	module models_handler_module.Module,
	userData userDataCollection,
	deploymentId string,
	containerAliases map[string]string,
	currentDeployment models_handler_storage.Deployment,
	currentDeploymentContainers map[string]models_handler_storage.DeploymentContainer,
	currentDeploymentVolumes map[string]models_handler_storage.DeploymentVolume,
	cache cacheCollection,
) error {
	if currentDeployment.ModuleSource+currentDeployment.ModuleChannel+currentDeployment.ModuleVersion != module.Source+module.Channel+module.Version {
		return errors.New("module " + module.ID + " has changed and must be updated first")
	}
	defaultData, err := getDefaultData(module)
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
	containers, err := newContainers2(module.Services, containerAliases, deploymentId)
	if err != nil {
		return err
	}
	err = h.removeDeploymentEnvironment(
		ctx,
		deploymentId,
		currentDeployment.DirName,
		currentDeployment.FilesDirName,
		currentDeploymentContainers,
	)
	err = h.ensureDeploymentEnvironment(
		ctx,
		module.Services,
		module.FileSystem,
		deploymentId,
		currentDeployment.DirName,
		currentDeployment.FilesDirName,
		currentDeploymentVolumes,
	)
	bindMounts, err := h.getBindMounts(
		ctx,
		deploymentId,
		currentDeployment.FilesDirName,
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
		currentDeployment.DirName,
		currentDeployment.FilesDirName,
		userData.Secrets,
		userData.HostResources,
		containers,
		currentDeploymentVolumes,
		mergedConfigs,
		bindMounts,
		cache.SecretValues,
		cache.Deployments,
		cache.HostResources,
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

func (h *Handler) getDeploymentsUserDataFromDB(ctx context.Context, deploymentIds []string) (map[string]userDataCollection, error) {
	deploymentsHostResources, err := h.storageHdl.ReadDeploymentsHostResources(ctx, models_handler_storage.DeploymentsHostResourcesFilter{
		DeploymentIds: deploymentIds,
	})
	if err != nil {
		return nil, err
	}
	deploymentsSecrets, err := h.storageHdl.ReadDeploymentsSecrets(ctx, models_handler_storage.DeploymentsSecretsFilter{
		DeploymentIds: deploymentIds,
	})
	if err != nil {
		return nil, err
	}
	deploymentsConfigs, err := h.storageHdl.ReadDeploymentsConfigs(ctx, deploymentIds)
	if err != nil {
		return nil, err
	}
	deploymentsGlobalConfigs, err := h.storageHdl.ReadDeploymentsGlobalConfigs(ctx, models_handler_storage.DeploymentGlobalConfigsFilter{
		DeploymentIds: deploymentIds,
	})
	if err != nil {
		return nil, err
	}
	deploymentsFiles, err := h.storageHdl.ReadDeploymentsFiles(ctx, deploymentIds)
	if err != nil {
		return nil, err
	}
	deploymentsFileGroups, err := h.storageHdl.ReadDeploymentsFileGroups(ctx, deploymentIds)
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
	map[string]map[string]models_handler_storage.DeploymentVolume,
	map[string]map[string]models_handler_storage.DeploymentContainer,
	error,
) {
	deploymentsVolumes, err := h.storageHdl.ReadDeploymentsVolumes(ctx, deploymentIds)
	if err != nil {
		return nil, nil, err
	}
	deploymentsContainers, err := h.storageHdl.ReadDeploymentsContainers(ctx, deploymentIds)
	if err != nil {
		return nil, nil, err
	}
	return deploymentsVolumes, deploymentsContainers, nil
}

func (h *Handler) updateCaches(
	ctx context.Context,
	moduleDependencies map[string]string,
	userDataHostResources map[string]models_handler_storage.DeploymentHostResource,
	userDataSecrets map[string]models_handler_storage.DeploymentSecret,
	userDataGlobalConfigs map[string]models_handler_storage.DeploymentGlobalConfig,
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
	moduleServices map[string]models_external.ModuleLibService,
	moduleFileSystem fs.FS,
	deploymentId string,
	deploymentDirName string,
	deploymentFilesDirName string,
	volumes map[string]models_handler_storage.DeploymentVolume,
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
	deploymentContainers map[string]models_handler_storage.DeploymentContainer,
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
	module models_handler_module.Module,
	defaultData defaultDataCollection,
	userDataConfigs map[string]models_handler_storage.DeploymentUserConfig,
	userDataGlobalConfigs map[string]models_handler_storage.DeploymentGlobalConfig,
	userDataFiles map[string]models_handler_storage.DeploymentFile,
	cacheGlobalConfigs map[string]models_handler_storage.GlobalConfig,
) (map[string]models_handler_storage.Config, map[string][]byte, error) {
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
