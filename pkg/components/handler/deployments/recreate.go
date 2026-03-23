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

	helper_maps "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/maps"
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
	// map deployments to module IDs
	deployments = helper_maps.CollectFunc(maps.Values(deployments), func(value models_handler_storage.Deployment) string {
		return value.ModuleId
	})
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
		deployment := deployments[moduleId]
		if deployment.ModuleSource+deployment.ModuleChannel+deployment.ModuleVersion != module.Source+module.Channel+module.Version {
			errs = append(errs, "module "+moduleId+" has changed and must be updated first")
			continue
		}
		defaultData, err := getDefaultData(module)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		userDataGlobalConfigs := deploymentsGlobalConfigs[deployment.Id]
		userDataHostResources := deploymentsHostResources[deployment.Id]
		userDataSecrets := deploymentsSecrets[deployment.Id]
		userDataConfigs := deploymentsConfigs[deployment.Id]
		userDataFiles := deploymentsFiles[deployment.Id]
		userDataFileGroups := deploymentsFileGroups[deployment.Id]
		err = h.updateGlobalConfigsCache(ctx, userDataGlobalConfigs, cacheGlobalConfigs)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		mergedConfigs := mergeConfigs(defaultData.Configs, userDataConfigs, userDataGlobalConfigs, cacheGlobalConfigs)
		err = checkConfigs(module.Configs, mergedConfigs)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		mergedFiles := mergeFiles(defaultData.Files, userDataFiles)
		err = checkFiles(module.Files, mergedFiles)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		containers, err := newContainers2(module.Services, cacheItem.ContainerAliases, deployment.Id)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		volumes := deploymentsVolumes[deployment.Id]
		currentContainers := deploymentsContainers[deployment.Id]
		err = h.removeContainers(ctx, currentContainers)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.removeSecretMounts(ctx, deployment.Id)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.removeDeploymentDirs(deployment.DirName, deployment.FilesDirName)
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
		err = h.updateHostResourcesCache(ctx, userDataHostResources, cacheHostResources)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.updateSecretValuesCache(ctx, userDataSecrets, cacheSecretValues)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.ensureContainerImages(ctx, module.Services)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.ensureContainerVolumes(ctx, volumes, deployment.Id)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.createDeploymentDirs(module.FileSystem, deployment.DirName, deployment.FilesDirName)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		bindMounts, err := h.getBindMounts(
			ctx,
			deployment.Id,
			deployment.FilesDirName,
			userDataFileGroups,
			userDataSecrets,
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
			deployment.Id,
			deployment.DirName,
			deployment.FilesDirName,
			userDataSecrets,
			userDataHostResources,
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
