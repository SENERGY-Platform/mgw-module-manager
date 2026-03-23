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
	"slices"
	"strings"

	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) RecreateDeployments(ctx context.Context, selectedModules map[string]models_handler_module.Module) error {
	deployments, err := h.storageHdl.ReadDeployments(ctx, models_handler_storage.DeploymentsFilter{
		ModuleIds: slices.Collect(maps.Keys(selectedModules)),
	})
	if err != nil {
		return err
	}
	deploymentIds := slices.Collect(maps.Keys(deployments))
	deploymentsHostResources, err := h.storageHdl.ReadDeploymentsHostResources(ctx, models_handler_storage.DeploymentsHostResourcesFilter{
		DeploymentIds: deploymentIds,
	})
	if err != nil {
		return err
	}
	deploymentsSecrets, err := h.storageHdl.ReadDeploymentsSecrets(ctx, models_handler_storage.DeploymentsSecretsFilter{
		DeploymentIds: deploymentIds,
	})
	if err != nil {
		return err
	}
	deploymentsConfigs, err := h.storageHdl.ReadDeploymentsConfigs(ctx, deploymentIds)
	if err != nil {
		return err
	}
	deploymentsGlobalConfigs, err := h.storageHdl.ReadDeploymentsGlobalConfigs(ctx, models_handler_storage.DeploymentGlobalConfigsFilter{
		DeploymentIds: deploymentIds,
	})
	if err != nil {
		return err
	}
	deploymentsFiles, err := h.storageHdl.ReadDeploymentsFiles(ctx, deploymentIds)
	if err != nil {
		return err
	}
	deploymentsFileGroups, err := h.storageHdl.ReadDeploymentsFileGroups(ctx, deploymentIds)
	if err != nil {
		return err
	}
	deploymentsVolumes, err := h.storageHdl.ReadDeploymentsVolumes(ctx, deploymentIds)
	if err != nil {
		return err
	}
	deploymentsContainers, err := h.storageHdl.ReadDeploymentsContainers(ctx, deploymentIds)
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
			errs = append(errs, "module "+moduleId+" not deployed") // TODO
			continue
		}
		err = h.recreateDeployment(
			ctx,
			module,
			userDataCollection{
				GlobalConfigs: deploymentsGlobalConfigs[cacheItem.DeploymentId],
				HostResources: deploymentsHostResources[cacheItem.DeploymentId],
				Secrets:       deploymentsSecrets[cacheItem.DeploymentId],
				Configs:       deploymentsConfigs[cacheItem.DeploymentId],
				Files:         deploymentsFiles[cacheItem.DeploymentId],
				FileGroups:    deploymentsFileGroups[cacheItem.DeploymentId],
			},
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

func (h *Handler) recreateDeployment(
	ctx context.Context,
	module models_handler_module.Module,
	userData userDataCollection,
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
	if currentDeployment.ModuleSource+currentDeployment.ModuleChannel+currentDeployment.ModuleVersion != module.Source+module.Channel+module.Version {
		return errors.New("module " + module.ID + " has changed and must be updated first")
	}
	defaultData, err := getDefaultData(module)
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
	containers, err := newContainers2(module.Services, containerAliases, deploymentId)
	if err != nil {
		return err
	}
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
	err = h.ensureContainerVolumes(ctx, currentDeploymentVolumes, deploymentId)
	if err != nil {
		return err
	}
	err = h.createDeploymentDirs(module.FileSystem, currentDeployment.DirName, currentDeployment.FilesDirName)
	if err != nil {
		return err
	}
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
