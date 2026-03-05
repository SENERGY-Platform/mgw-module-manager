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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
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

func (h *Handler) updateHostResourcesCache(
	ctx context.Context,
	hostResourcesCache map[string]models_external.HostResource,
	selectedHostResources []models_handler_storage.DeploymentHostResource,
) error {
	selectedIds := helper_slices.CollectFunc(slices.Values(selectedHostResources), func(item models_handler_storage.DeploymentHostResource) string {
		return item.Id
	})
	var idsNotInCache []string
	for _, id := range selectedIds {
		if _, ok := hostResourcesCache[id]; ok {
			idsNotInCache = append(idsNotInCache, id)
		}
	}
	if len(idsNotInCache) == 0 {
		return nil
	}
	var errs []string
	for _, id := range idsNotInCache {
		hostResource, err := h.hmClient.GetHostResource(ctx, id)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		hostResourcesCache[hostResource.ID] = hostResource
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func (h *Handler) updateDependenciesCache(
	ctx context.Context,
	dependenciesCache map[string]deploymentWrapper,
	moduleDependencies map[string]string,
) error {
	moduleIds := slices.Collect(maps.Keys(moduleDependencies))
	var idsNotInCache []string
	for _, id := range moduleIds {
		if _, ok := dependenciesCache[id]; !ok {
			idsNotInCache = append(idsNotInCache, id)
		}
	}
	if len(idsNotInCache) == 0 {
		return nil
	}
	deployments, err := h.storageHdl.ReadDeployments(ctx, models_handler_storage.DeploymentsFilter{ModuleIds: idsNotInCache})
	if err != nil {
		return err
	}
	deploymentsContainers, err := h.storageHdl.ReadDeploymentsContainers(ctx, slices.Collect(maps.Keys(deployments)))
	if err != nil {
		return err
	}
	for id, deployment := range deployments {
		containers := make(map[string]containerWrapper)
		deploymentContainers := deploymentsContainers[id]
		for _, container := range deploymentContainers {
			containers[container.Reference] = containerWrapper{
				DeploymentContainer: container,
			}
		}
		dependenciesCache[deployment.ModuleId] = deploymentWrapper{
			Deployment: deployment,
			Containers: containers,
		}
	}
	var errs []string
	for _, id := range idsNotInCache {
		if _, ok := dependenciesCache[id]; !ok {
			errs = append(errs, fmt.Sprintf("dependency %v not found", id))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func (h *Handler) updateSecretValuesCache(
	ctx context.Context,
	secretValuesCache map[string]models_external.SecretValueVariant,
	selectedSecrets map[string]models_handler_storage.DeploymentSecret,
) error {
	var errs []string
	for _, secret := range selectedSecrets {
		for _, secretItem := range secret.Items {
			if secretItem.AsMount {
				continue
			}
			cacheKey := secret.Id + secretItem.Name
			var reqItem *string
			if secretItem.Name != "" {
				reqItem = &secretItem.Name
			}
			_, ok := secretValuesCache[cacheKey]
			if !ok {
				var err error
				valueVariant, err, _ := h.smClient.GetValueVariant(ctx, models_external.SecretVariantRequest{
					ID:   secret.Id,
					Item: reqItem,
				})
				if err != nil {
					errs = append(errs, err.Error())
					continue
				}
				secretValuesCache[cacheKey] = valueVariant
			}
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func (h *Handler) getSecretMounts(
	ctx context.Context,
	selectedSecrets map[string]models_handler_storage.DeploymentSecret,
	deploymentId string,
) (map[string]models_external.SecretPathVariant, error) {
	secretMounts := make(map[string]models_external.SecretPathVariant)
	var errs []string
	for _, secret := range selectedSecrets {
		for _, secretItem := range secret.Items {
			if secretItem.AsEnv {
				continue
			}
			key := secret.Id + secretItem.Name
			var reqItem *string
			if secretItem.Name != "" {
				reqItem = &secretItem.Name
			}
			_, ok := secretMounts[key]
			if !ok {
				pathVariant, err, _ := h.smClient.InitPathVariant(ctx, models_external.SecretVariantRequest{
					ID:        secret.Id,
					Item:      reqItem,
					Reference: deploymentId,
				})
				if err != nil {
					errs = append(errs, err.Error())
					continue
				}
				secretMounts[key] = pathVariant
			}
		}
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return secretMounts, nil
}

func fileToBytes(fSys fs.FS, path string) ([]byte, error) {
	f, err := fSys.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func getDefaultFiles(moduleFiles map[string]models_external.ModuleFile, moduleFS fs.FS) (map[string][]byte, error) {
	files := make(map[string][]byte)
	var errs []string
	for reference, file := range moduleFiles {
		if file.Source != "" {
			b, err := fileToBytes(moduleFS, file.Source)
			if err != nil {
				errs = append(errs, err.Error())
				continue
			}
			files[reference] = b
		}
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return files, nil
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

func getSelectedSecrets(
	moduleSecrets map[string]models_external.ModuleSecret,
	moduleServices map[string]models_external.ModuleService,
	userInputs map[string]string,
	deploymentID string,
) (map[string]models_handler_storage.DeploymentSecret, error) {
	secrets := make(map[string]models_handler_storage.DeploymentSecret)
	var errs []string
	for reference, secret := range moduleSecrets {
		id, ok := userInputs[reference]
		if !ok {
			if secret.Required {
				errs = append(errs, fmt.Sprintf("secret %s required", reference))
			}
			continue
		}
		secrets[reference] = models_handler_storage.DeploymentSecret{
			Id:           id,
			DeploymentId: deploymentID,
			Reference:    reference,
			Items:        getSecretItems(reference, moduleServices),
		}
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return secrets, nil
}

func getSecretItems(reference string, moduleServices map[string]models_external.ModuleService) []models_handler_storage.DeploymentSecretItem {
	items := make(map[string]models_handler_storage.DeploymentSecretItem)
	for _, moduleService := range moduleServices {
		for _, target := range moduleService.SecretVars {
			if target.Ref == reference {
				item, ok := items[target.Item]
				if !ok {
					item.Name = target.Item
				}
				item.AsEnv = true
				items[target.Item] = item
			}
		}
		for _, target := range moduleService.SecretMounts {
			if target.Ref == reference {
				item, ok := items[target.Item]
				if !ok {
					item.Name = target.Item
				}
				item.AsMount = true
				items[target.Item] = item
			}
		}
	}
	return slices.Collect(maps.Values(items))
}

func getSelectedHostResources(
	moduleHostResources map[string]models_external.ModuleHostResource,
	userInputs map[string]string,
	deploymentID string,
) ([]models_handler_storage.DeploymentHostResource, error) {
	var hostResources []models_handler_storage.DeploymentHostResource
	var errs []string
	for reference, hostResource := range moduleHostResources {
		id, ok := userInputs[reference]
		if !ok {
			if hostResource.Required {
				errs = append(errs, fmt.Sprintf("missing required host resource '%s'", reference))
			}
			continue
		}
		hostResources = append(hostResources, models_handler_storage.DeploymentHostResource{
			Id:           id,
			DeploymentId: deploymentID,
			Reference:    reference,
		})
	}
	return hostResources, nil
}

func getProvidedFiles(
	moduleFiles map[string]models_external.ModuleFile,
	defaultFiles map[string][]byte, userInputs map[string][]byte,
	deploymentId string,
) (map[string]models_handler_storage.DeploymentFile, error) {
	files := make(map[string]models_handler_storage.DeploymentFile)
	var errs []string
	for reference, file := range moduleFiles {
		defaultData, defaultOK := defaultFiles[reference]
		data, ok := userInputs[reference]
		if !ok {
			if file.Required && !defaultOK {
				errs = append(errs, fmt.Sprintf("missing required file '%s'", reference))
			}
			continue
		}
		if defaultOK && bytes.Equal(data, defaultData) {
			continue
		}
		files[reference] = models_handler_storage.DeploymentFile{
			DeploymentId: deploymentId,
			Reference:    reference,
			Data:         data,
		}
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return files, nil
}

func getProvidedFileGroups(moduleFileGroups map[string]struct{}, userInputs map[string]map[string]models_handler_deployment.FileGroupUserInput, deploymentId string) map[string]models_handler_storage.DeploymentFileGroup {
	fileGroups := make(map[string]models_handler_storage.DeploymentFileGroup)
	for reference := range moduleFileGroups {
		fg, ok := userInputs[reference]
		if !ok {
			continue
		}
		var files []models_handler_storage.DeploymentFileGroupFile
		for path, input := range fg {
			files = append(files, models_handler_storage.DeploymentFileGroupFile{
				Path:   path,
				Format: input.Format,
				Data:   input.Data,
			})
		}
		fileGroups[reference] = models_handler_storage.DeploymentFileGroup{
			Id:           deploymentId + "_" + reference,
			DeploymentId: deploymentId,
			Reference:    reference,
			Files:        files,
		}
	}
	return fileGroups
}
