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
	"maps"
	"slices"
	"time"

	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	helper_uuid "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/uuid"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_deployment "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployment"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) CreateDeployments(ctx context.Context, selectedModules map[string]models_handler_module.Module, userInputs map[string]models_handler_deployment.UserInput) (map[string]models_handler_deployment.Deployment, error) {
	deploymentWrappers, err := newDeploymentWrappers(selectedModules)
	if err != nil {
		return nil, err
	}
	cache := newCache(deploymentWrappers)
	for moduleId, deployment := range deploymentWrappers {
		if deployment == nil || deployment.Error != nil {
			continue
		}
		userInput := userInputs[moduleId]
		if userInput.Name != "" {
			deployment.Name = userInput.Name
		}
		defaultFiles, err := getDefaultFiles(deployment.Module.Files, deployment.ModuleFileSystem)
		if err != nil {
			deployment.Error = err
			continue
		}
		defaultConfigs, err := getDefaultConfigs(deployment.Module.Configs)
		if err != nil {
			deployment.Error = err
			continue
		}
		deployment.Configs, deployment.Error = getProvidedConfigs(deployment.Module.Configs, defaultConfigs, userInput.Configs, deployment.Id)
		if deployment.Error != nil {
			continue
		}
		deployment.GlobalConfigs = getSelectedGlobalConfigs(deployment.Module.Configs, userInput.GlobalConfigs, deployment.Id)
		configs := mergeConfigs(defaultConfigs, deployment.Configs, deployment.GlobalConfigs, cache.GlobalConfigs)
		deployment.Error = checkConfigs(deployment.Module.Configs, configs)
		if deployment.Error != nil {
			continue
		}
		deployment.HostResources, deployment.Error = getSelectedHostResources(deployment.Module.HostResources, userInput.HostResources, deployment.Id)
		if deployment.Error != nil {
			continue
		}
		deployment.Secrets, deployment.Error = getSelectedSecrets(deployment.Module.Secrets, deployment.Module.Services, userInput.Secrets, deployment.Id)
		if deployment.Error != nil {
			continue
		}
		deployment.Files, deployment.Error = getProvidedFiles(deployment.Module.Files, defaultFiles, userInput.Files, deployment.Id)
		if deployment.Error != nil {
			continue
		}
		deployment.FileGroups = getProvidedFileGroups(deployment.Module.FileGroups, userInput.FileGroups, deployment.Id)
		deployment.Error = h.storageHdl.CreateDeployment(
			ctx,
			deployment.Deployment,
			slices.Collect(maps.Values(deployment.HostResources)),
			slices.Collect(maps.Values(deployment.Secrets)),
			slices.Collect(maps.Values(deployment.Configs)),
			slices.Collect(maps.Values(deployment.GlobalConfigs)),
			slices.Collect(maps.Values(deployment.Files)),
			slices.Collect(maps.Values(deployment.FileGroups)),
			slices.Collect(maps.Values(deployment.Volumes)),
			helper_slices.CollectFunc(maps.Values(deployment.Containers), func(item containerWrapper) models_handler_storage.DeploymentContainer {
				return item.DeploymentContainer
			}),
		)
		if deployment.Error != nil {
			continue
		}
		// --------------------------------------------------------------------------
		deployment.Error = h.updateExternalDependenciesCache(ctx, cache.ExternalDependencies, deployment.Module.Dependencies)
		if deployment.Error != nil {
			continue
		}
		deployment.Error = h.updateGlobalConfigsCache(ctx, cache.GlobalConfigs, deployment.GlobalConfigs)
		if deployment.Error != nil {
			continue
		}
		deployment.Error = h.updateHostResourcesCache(ctx, cache.HostResources, deployment.HostResources)
		if deployment.Error != nil {
			continue
		}
		deployment.Error = h.updateSecretValuesCache(ctx, cache.SecretValues, deployment.Secrets)
		if deployment.Error != nil {
			continue
		}
		secretMounts, err := h.getSecretMounts(ctx, deployment.Secrets, deployment.Id) // add unload secrets if error
		if err != nil {
			deployment.Error = err
			continue
		}
		configStrings := configsToStrings(
			deployment.Module.Configs,
			configs,
		)
	}
	return nil, nil
}

func newDeploymentWrappers(modules map[string]models_handler_module.Module) (map[string]*deploymentWrapper, error) {
	deployments := make(map[string]*deploymentWrapper)
	for _, module := range modules {
		id, err := helper_uuid.New()
		if err != nil {
			return nil, err
		}
		dirName, err := helper_uuid.New()
		if err != nil {
			return nil, err
		}
		containerWrappers, err := newContainerWrappers(module.Services, id)
		if err != nil {
			return nil, err
		}
		deployment := &deploymentWrapper{
			Deployment: models_handler_storage.Deployment{
				Id:            id,
				ModuleId:      module.ID,
				ModuleSource:  module.Source,
				ModuleChannel: module.Channel,
				ModuleVersion: module.Version,
				Name:          module.Name,
				DirName:       dirName,
				Created:       helper_time.Now(),
			},
			Containers:       containerWrappers,
			Volumes:          newVolumes(module.Volumes, id),
			Module:           module.Module,
			ModuleFileSystem: module.FileSystem,
		}
		deployments[module.ID] = deployment
	}
	return deployments, nil
}

func newContainerWrappers(moduleServices map[string]models_external.ModuleService, deploymentId string) (map[string]containerWrapper, error) {
	containerWrappers := make(map[string]containerWrapper)
	for ref := range moduleServices {
		containerName, err := helper_naming.NewContainerName("dep")
		if err != nil {
			return nil, err
		}
		containerWrappers[ref] = containerWrapper{
			DeploymentContainer: models_handler_storage.DeploymentContainer{
				DeploymentId: deploymentId,
				Reference:    ref,
				Alias:        helper_naming.NewContainerAlias(deploymentId, ref),
			},
			Name: containerName,
		}
	}
	return containerWrappers, nil
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

func newCache(deployments map[string]*deploymentWrapper) *cacheWrapper {
	return &cacheWrapper{
		ExternalDependencies: newExternalDependenciesCache(deployments),
		HostResources:        make(map[string]models_external.HostResource),
		GlobalConfigs:        make(map[string]models_handler_storage.GlobalConfig),
		SecretValues:         make(map[string]models_external.SecretValueVariant),
	}
}
