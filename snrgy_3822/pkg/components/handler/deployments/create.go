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
	"fmt"
	"maps"
	"slices"
	"strings"

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
	deploymentWrappers, err := getDeploymentWrappers(selectedModules)
	if err != nil {
		return nil, err
	}
	dependenciesCache := make(map[string]deploymentWrapper)
	hostResourcesCache := make(map[string]models_external.HostResource)
	globalConfigsCache := make(map[string]models_handler_storage.GlobalConfig)
	secretValuesCache := make(map[string]models_external.SecretValueVariant)
	for _, deployment := range deploymentWrappers {
		if deployment.Error != nil {
			continue
		}
		userInput := userInputs[deployment.Module.ID]
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
		providedConfigs, err := getProvidedConfigs(deployment.Module.Configs, defaultConfigs, userInput.Configs, deployment.Id)
		if err != nil {
			deployment.Error = err
			continue
		}
		selectedGlobalConfigs := getSelectedGlobalConfigs(deployment.Module.Configs, userInput.GlobalConfigs, deployment.Id)
		deployment.Error = checkConfigs(deployment.Module.Configs, defaultConfigs, providedConfigs, selectedGlobalConfigs)
		if deployment.Error != nil {
			continue
		}
		selectedHostResources, err := getSelectedHostResources(deployment.Module.HostResources, userInput.HostResources, deployment.Id)
		if err != nil {
			deployment.Error = err
			continue
		}
		selectedSecrets, err := getSelectedSecrets(deployment.Module.Secrets, deployment.Module.Services, userInput.Secrets, deployment.Id)
		if err != nil {
			deployment.Error = err
			continue
		}
		providedFiles, err := getProvidedFiles(deployment.Module.Files, defaultFiles, userInput.Files, deployment.Id)
		if err != nil {
			deployment.Error = err
			continue
		}
		providedFileGroups := getProvidedFileGroups(deployment.Module.FileGroups, userInput.FileGroups, deployment.Id)
		deployment.Error = h.storageHdl.CreateDeployment(
			ctx,
			deployment.Deployment,
			selectedHostResources,
			slices.Collect(maps.Values(selectedSecrets)),
			slices.Collect(maps.Values(providedConfigs)),
			slices.Collect(maps.Values(selectedGlobalConfigs)),
			slices.Collect(maps.Values(providedFiles)),
			slices.Collect(maps.Values(providedFileGroups)),
			helper_slices.CollectFunc(maps.Values(deployment.Containers), func(item containerWrapper) models_handler_storage.DeploymentContainer {
				return item.DeploymentContainer
			}),
		)
		if deployment.Error != nil {
			continue
		}
		// --------------------------------------------------------------------------
		deployment.Error = h.updateDependenciesCache(ctx, dependenciesCache, deployment.Module.Dependencies)
		if deployment.Error != nil {
			continue
		}
		deployment.Error = h.updateGlobalConfigsCache(ctx, globalConfigsCache, selectedGlobalConfigs)
		if deployment.Error != nil {
			continue
		}
		deployment.Error = h.updateHostResourcesCache(ctx, hostResourcesCache, selectedHostResources)
		if deployment.Error != nil {
			continue
		}
		deployment.Error = h.updateSecretValuesCache(ctx, secretValuesCache, selectedSecrets)
		if deployment.Error != nil {
			continue
		}
		secretMounts, err := h.getSecretMounts(ctx, selectedSecrets, deployment.Id)
		if err != nil {
			deployment.Error = err
			continue
		}
		configStrings := configsToStrings(
			deployment.Module.Configs,
			mergeConfigs(defaultConfigs, providedConfigs, selectedGlobalConfigs, globalConfigsCache),
		)
	}
	return nil, nil
}

func getDeploymentWrappers(modules map[string]models_handler_module.Module) (map[string]*deploymentWrapper, error) {
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
		containerWrappers := make(map[string]containerWrapper)
		for ref := range module.Services {
			containerName, err := helper_naming.NewContainerName("dep")
			if err != nil {
				return nil, err
			}
			containerWrappers[ref] = containerWrapper{
				DeploymentContainer: models_handler_storage.DeploymentContainer{
					DeploymentId: id,
					Reference:    ref,
					Alias:        helper_naming.NewContainerAlias(id, ref),
				},
				Name: containerName,
			}
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
			Module:           module.Module,
			ModuleFileSystem: module.FileSystem,
		}
		deployments[module.ID] = deployment
	}
	return deployments, nil
}
