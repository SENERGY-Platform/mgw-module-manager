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
	"math"
	"slices"
	"strconv"
	"strings"

	module_lib_validation_configs "github.com/SENERGY-Platform/mgw-module-lib/validation/configs"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	helper_uuid "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/uuid"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_deployment "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployment"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) CreateDeployments(ctx context.Context, modules map[string]models_handler_module.Module, userInputs map[string]models_handler_deployment.UserInput) (map[string]models_handler_deployment.Deployment, error) {
	deployments, err := newDeploymentWrappers(modules, userInputs)
	if err != nil {
		return nil, err
	}
	dependenciesCache := make(map[string]models_handler_storage.Deployment)
	hostResourcesCache := make(map[string]models_external.HostResource)
	globalConfigsCache := make(map[string]models_handler_storage.GlobalConfig)
	secretValuesCache := make(map[string]models_external.SecretValueVariant)
	for _, deployment := range deployments {
		if deployment.Error != nil {
			continue
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
		userInput := userInputs[deployment.Module.ID]
		providedConfigs, err := extractUserConfigs(deployment.Module.Configs, defaultConfigs, userInput.Configs, deployment.Id)
		if err != nil {
			deployment.Error = err
			continue
		}
		selectedGlobalConfigs := extractGlobalConfigs(deployment.Module.Configs, userInput.GlobalConfigs, deployment.Id)
		deployment.Error = checkConfigs(deployment.Module.Configs, defaultConfigs, providedConfigs, selectedGlobalConfigs)
		if deployment.Error != nil {
			continue
		}
		selectedHostResources, err := extractHostResources(deployment.Module.HostResources, userInput.HostResources, deployment.Id)
		if err != nil {
			deployment.Error = err
			continue
		}
		selectedSecrets, err := extractSecrets(deployment.Module.Secrets, deployment.Module.Services, userInput.Secrets, deployment.Id)
		if err != nil {
			deployment.Error = err
			continue
		}
		providedFiles, err := extractFiles(deployment.Module.Files, defaultFiles, userInput.Files, deployment.Id)
		if err != nil {
			deployment.Error = err
			continue
		}
		providedFileGroups := extractFileGroups(deployment.Module.FileGroups, userInput.FileGroups, deployment.Id)
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
		deployment.Error = h.updateDependenciesCache(
			ctx,
			dependenciesCache,
			slices.Collect(maps.Keys(deployment.Module.Dependencies)),
		)
		if deployment.Error != nil {
			continue
		}
		deployment.Error = h.updateGlobalConfigsCache(
			ctx,
			globalConfigsCache,
			helper_slices.CollectFunc(maps.Values(selectedGlobalConfigs), func(item models_handler_storage.DeploymentGlobalConfig) string {
				return item.Id
			}),
		)
		if deployment.Error != nil {
			continue
		}
		deployment.Error = h.updateHostResourcesCache(
			ctx,
			hostResourcesCache,
			helper_slices.CollectFunc(slices.Values(selectedHostResources), func(item models_handler_storage.DeploymentHostResource) string {
				return item.Id
			}),
		)
		if deployment.Error != nil {
			continue
		}
		secrets, err := h.getSecrets(ctx, deployment.Module.Secrets, secretValuesCache, selectedSecrets, deployment.Id) // mount secrets müssen "unloaded" werden
		if err != nil {
			deployment.Error = err
			continue
		}
		configStrings := configsToStrings(deployment.Module.Configs, defaultConfigs, providedConfigs, selectedGlobalConfigs, globalConfigsCache)

	}
	return nil, nil
}

func (h *Handler) getGlobalConfigs(ctx context.Context, globalConfigsCache map[string]models_handler_storage.GlobalConfig, globalConfigIds []string) error {
	var idsNotInCache []string
	for _, globalConfigId := range globalConfigIds {
		if _, ok := globalConfigsCache[globalConfigId]; ok {
			idsNotInCache = append(idsNotInCache, globalConfigId)
		}
	}
	if len(idsNotInCache) == 0 {
		return nil
	}
	globalConfigs, err := h.storageHdl.ReadGlobalConfigs(ctx, idsNotInCache)
	if err != nil {
		return err
	}
	for _, globalConfig := range globalConfigs {
		globalConfigsCache[globalConfig.Id] = globalConfig
	}
	var idsNotFound []string
	for _, globalConfigId := range idsNotInCache {
		if _, ok := globalConfigsCache[globalConfigId]; !ok {
			idsNotFound = append(idsNotFound, globalConfigId)
		}
	}
	if len(idsNotFound) > 0 {
		return fmt.Errorf("global confgis %v not found", idsNotFound) // TODO
	}
	return nil
}

func (h *Handler) getHostResources(ctx context.Context, hostResourcesCache map[string]models_external.HostResource, hostResourceIds []string) error {
	var idsNotInCache []string
	for _, hostResourceId := range hostResourceIds {
		if _, ok := hostResourcesCache[hostResourceId]; ok {
			idsNotInCache = append(idsNotInCache, hostResourceId)
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

func (h *Handler) getDeploymentDependencies(ctx context.Context, dependenciesCache map[string]models_handler_storage.Deployment, moduleIds []string) error {
	var idsNotInCache []string
	for _, moduleId := range moduleIds {
		if _, ok := dependenciesCache[moduleId]; !ok {
			idsNotInCache = append(idsNotInCache, moduleId)
		}
	}
	if len(idsNotInCache) == 0 {
		return nil
	}
	deployments, err := h.storageHdl.ReadDeployments(ctx, models_handler_storage.DeploymentsFilter{ModuleIds: idsNotInCache})
	if err != nil {
		return err
	}
	for _, deployment := range deployments {
		dependenciesCache[deployment.ModuleId] = deployment
	}
	var idsNotFound []string
	for _, moduleId := range idsNotInCache {
		if _, ok := dependenciesCache[moduleId]; !ok {
			idsNotFound = append(idsNotFound, moduleId)
		}
	}
	if len(idsNotFound) > 0 {
		return fmt.Errorf("dependencies %v not found", idsNotFound) // TODO
	}
	return nil
}

func (h *Handler) getSecrets(
	ctx context.Context,
	moduleSecrets map[string]models_external.ModuleSecret,
	secretValuesCache map[string]models_external.SecretValueVariant,
	deploymentSecrets map[string]models_handler_storage.DeploymentSecret,
	deploymentId string,
) (map[string]models_external.SecretPathVariant, error) {
	secretMounts := make(map[string]models_external.SecretPathVariant)
	var errs []string
	for reference, moduleSecret := range moduleSecrets {
		deploymentSecret, ok := deploymentSecrets[reference]
		if !ok {
			if moduleSecret.Required {
				errs = append(errs, fmt.Sprintf("missing required secret '%s'", reference))
			}
			continue
		}
		for _, item := range deploymentSecret.Items {
			key := deploymentSecret.Id + item.Name
			var reqItem *string
			if item.Name != "" {
				reqItem = &item.Name
			}
			if item.AsEnv {
				_, ok := secretValuesCache[key]
				if !ok {
					valueVariant, err, _ := h.smClient.GetValueVariant(ctx, models_external.SecretVariantRequest{
						ID:   deploymentSecret.Id,
						Item: reqItem,
					})
					if err != nil {
						errs = append(errs, err.Error())
						continue
					}
					secretValuesCache[key] = valueVariant
				}
			}
			if item.AsMount {
				_, ok := secretMounts[key]
				if !ok {
					pathVariant, err, _ := h.smClient.InitPathVariant(ctx, models_external.SecretVariantRequest{
						ID:        deploymentSecret.Id,
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
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return secretMounts, nil
}

func configToString(config models_handler_storage.Config, delimiter string) string {
	if config.IsSlice {
		var values []string
		switch config.DataType {
		case models_handler_storage.StringType:
			values = config.StringSlice
		case models_handler_storage.Int64Type:
			for _, i := range config.Int64Slice {
				values = append(values, strconv.FormatInt(i, 10))
			}
		case models_handler_storage.Float64Type:
			for _, f := range config.Float64Slice {
				values = append(values, strconv.FormatFloat(f, 'f', -1, 64))
			}
		case models_handler_storage.BoolType:
			for _, b := range config.BoolSlice {
				values = append(values, strconv.FormatBool(b))
			}
		}
		return strings.Join(values, delimiter)
	} else {
		switch config.DataType {
		case models_handler_storage.StringType:
			return config.String
		case models_handler_storage.Int64Type:
			return strconv.FormatInt(config.Int64, 10)
		case models_handler_storage.Float64Type:
			return strconv.FormatFloat(config.Float64, 'f', -1, 64)
		case models_handler_storage.BoolType:
			return strconv.FormatBool(config.Bool)
		}
	}
	return ""
}

func checkConfigs(
	moduleConfigs models_external.ModuleConfigs,
	defaultConfigs map[string]models_handler_storage.Config,
	deploymentUserConfigs map[string]models_handler_storage.DeploymentUserConfig,
	deploymentGlobalConfigs map[string]models_handler_storage.DeploymentGlobalConfig,
) error {
	var errs []string
	for reference, moduleConfig := range moduleConfigs {
		_, ok := deploymentUserConfigs[reference]
		if ok {
			continue
		}
		_, ok = deploymentGlobalConfigs[reference]
		if ok {
			continue
		}
		_, ok = defaultConfigs[reference]
		if ok {
			continue
		}
		if moduleConfig.Required {
			errs = append(errs, fmt.Sprintf("config %s required", reference))
			continue
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func configsToStrings(
	moduleConfigs models_external.ModuleConfigs,
	defaultConfigs map[string]models_handler_storage.Config,
	deploymentUserConfigs map[string]models_handler_storage.DeploymentUserConfig,
	deploymentGlobalConfigs map[string]models_handler_storage.DeploymentGlobalConfig,
	globalConfigsCache map[string]models_handler_storage.GlobalConfig,
) map[string]string {
	configs := make(map[string]string)
	for reference, moduleConfig := range moduleConfigs {
		deploymentUserConfig, ok := deploymentUserConfigs[reference]
		if ok {
			configs[reference] = configToString(deploymentUserConfig.Config, moduleConfig.Delimiter)
			continue
		}
		deploymentGlobalConfig, ok := deploymentGlobalConfigs[reference]
		if ok {
			globalConfig, ok := globalConfigsCache[deploymentGlobalConfig.Id]
			if ok {
				configs[reference] = configToString(globalConfig.Config, moduleConfig.Delimiter)
				continue
			}
		}
		defaultConfig, ok := defaultConfigs[reference]
		if ok {
			configs[reference] = configToString(defaultConfig, moduleConfig.Delimiter)
		}
	}
	return configs
}

func getDefaultConfigs(moduleConfigs models_external.ModuleConfigs) (map[string]models_handler_storage.Config, error) {
	configs := make(map[string]models_handler_storage.Config)
	var errs []string
	for reference, moduleConfig := range moduleConfigs {
		if moduleConfig.Default == nil {
			continue
		}
		config, err := moduleConfigValueToConfig(moduleConfig.Default, moduleConfig)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		configs[reference] = config
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return configs, nil
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

func newDeploymentWrappers(modules map[string]models_handler_module.Module, userInputs map[string]models_handler_deployment.UserInput) (map[string]*deploymentWrapper, error) {
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
		name := userInputs[module.ID].Name
		if name == "" {
			name = module.Name
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
				Name:          name,
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

func getDeploymentSecrets(
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
			Items:        newDeploymentSecretItems(reference, moduleServices),
		}
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return secrets, nil
}

func newDeploymentSecretItems(reference string, moduleServices map[string]models_external.ModuleService) []models_handler_storage.DeploymentSecretItem {
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

func getDeploymentHostResources(moduleHostResources map[string]models_external.ModuleHostResource, userInputs map[string]string, deploymentID string) ([]models_handler_storage.DeploymentHostResource, error) {
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

func getDeploymentGlobalConfigs(moduleConfigs models_external.ModuleConfigs, userInputs map[string]string, deploymentId string) map[string]models_handler_storage.DeploymentGlobalConfig {
	configs := make(map[string]models_handler_storage.DeploymentGlobalConfig)
	for reference := range moduleConfigs {
		id, ok := userInputs[reference]
		if !ok {
			continue
		}
		configs[reference] = models_handler_storage.DeploymentGlobalConfig{
			Id:           id,
			DeploymentId: deploymentId,
			Reference:    reference,
		}
	}
	return configs
}

func getDeploymentFiles(
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

func getDeploymentFileGroups(moduleFileGroups map[string]struct{}, userInputs map[string]map[string]models_handler_deployment.FileGroupUserInput, deploymentId string) map[string]models_handler_storage.DeploymentFileGroup {
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

func configIsEqual(a, b models_handler_storage.Config) bool {
	if a.DataType != b.DataType {
		return false
	}
	if a.IsSlice != b.IsSlice {
		return false
	}
	if a.IsSlice {
		switch a.DataType {
		case models_handler_storage.StringType:
			return slices.Equal(a.StringSlice, b.StringSlice)
		case models_handler_storage.Int64Type:
			return slices.Equal(a.Int64Slice, b.Int64Slice)
		case models_handler_storage.Float64Type:
			return slices.Equal(a.Float64Slice, b.Float64Slice)
		case models_handler_storage.BoolType:
			return slices.Equal(a.BoolSlice, b.BoolSlice)
		}
	} else {
		switch a.DataType {
		case models_handler_storage.StringType:
			return a.String == b.String
		case models_handler_storage.Int64Type:
			return a.Int64 == b.Int64
		case models_handler_storage.Float64Type:
			return a.Float64 == b.Float64
		case models_handler_storage.BoolType:
			return a.Bool == b.Bool
		}
	}
	return false
}

func getDeploymentUserConfigs(moduleConfigs models_external.ModuleConfigs, defaultConfigs map[string]models_handler_storage.Config, userInputs map[string]any, deploymentId string) (map[string]models_handler_storage.DeploymentUserConfig, error) {
	configs := make(map[string]models_handler_storage.DeploymentUserConfig)
	var errs []string
	for reference, moduleConfig := range moduleConfigs {
		val, ok := userInputs[reference]
		if !ok || val == nil {
			continue
		}
		config, err := moduleConfigValueToConfig(val, moduleConfig)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		defaultConfig, ok := defaultConfigs[reference]
		if ok && configIsEqual(config, defaultConfig) {
			continue
		}
		config.Id = deploymentId + "_" + reference
		configs[reference] = models_handler_storage.DeploymentUserConfig{
			DeploymentId: deploymentId,
			Reference:    reference,
			Config:       config,
		}
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return configs, nil
}

func moduleConfigValueToConfig(val any, moduleConfig models_external.ModuleConfig) (models_handler_storage.Config, error) {
	config := models_handler_storage.Config{
		IsSlice: moduleConfig.IsSlice,
	}
	if moduleConfig.IsSlice {
		anySlice, ok := val.([]any)
		if !ok {
			return models_handler_storage.Config{}, fmt.Errorf("invalid data type '%T'", val) // TODO
		}
		switch moduleConfig.DataType {
		case models_external.ModuleConfigStringType:
			config.DataType = models_handler_storage.StringType
			for _, item := range anySlice {
				v, err := toString(item)
				if err != nil {
					return models_handler_storage.Config{}, err
				}
				err = validateValue(v, moduleConfig)
				if err != nil {
					return models_handler_storage.Config{}, err
				}
				config.StringSlice = append(config.StringSlice, v)
			}
		case models_external.ModuleConfigBoolType:
			config.DataType = models_handler_storage.BoolType
			for _, item := range anySlice {
				v, err := toBool(item)
				if err != nil {
					return models_handler_storage.Config{}, err
				}
				err = validateValue(v, moduleConfig)
				if err != nil {
					return models_handler_storage.Config{}, err
				}
				config.BoolSlice = append(config.BoolSlice, v)
			}
		case models_external.ModuleConfigInt64Type:
			config.DataType = models_handler_storage.Int64Type
			for _, item := range anySlice {
				v, err := toInt64(item)
				if err != nil {
					return models_handler_storage.Config{}, err
				}
				err = validateValue(v, moduleConfig)
				if err != nil {
					return models_handler_storage.Config{}, err
				}
				config.Int64Slice = append(config.Int64Slice, v)
			}
		case models_external.ModuleConfigFloat64Type:
			config.DataType = models_handler_storage.Float64Type
			for _, item := range anySlice {
				v, err := toFloat64(item)
				if err != nil {
					return models_handler_storage.Config{}, err
				}
				err = validateValue(v, moduleConfig)
				if err != nil {
					return models_handler_storage.Config{}, err
				}
				config.Float64Slice = append(config.Float64Slice, v)
			}
		default:
			return models_handler_storage.Config{}, fmt.Errorf("unknown data type '%s'", moduleConfig.DataType) // TODO
		}
	} else {
		switch moduleConfig.DataType {
		case models_external.ModuleConfigStringType:
			config.DataType = models_handler_storage.StringType
			v, err := toString(val)
			if err != nil {
				return models_handler_storage.Config{}, err
			}
			config.String = v
			err = validateValue(v, moduleConfig)
			if err != nil {
				return models_handler_storage.Config{}, err
			}
		case models_external.ModuleConfigBoolType:
			config.DataType = models_handler_storage.BoolType
			v, err := toBool(val)
			if err != nil {
				return models_handler_storage.Config{}, err
			}
			config.Bool = v
			err = validateValue(v, moduleConfig)
			if err != nil {
				return models_handler_storage.Config{}, err
			}
		case models_external.ModuleConfigInt64Type:
			config.DataType = models_handler_storage.Int64Type
			v, err := toInt64(val)
			if err != nil {
				return models_handler_storage.Config{}, err
			}
			config.Int64 = v
			err = validateValue(v, moduleConfig)
			if err != nil {
				return models_handler_storage.Config{}, err
			}
		case models_external.ModuleConfigFloat64Type:
			config.DataType = models_handler_storage.Float64Type
			v, err := toFloat64(val)
			if err != nil {
				return models_handler_storage.Config{}, err
			}
			config.Float64 = v
			err = validateValue(v, moduleConfig)
			if err != nil {
				return models_handler_storage.Config{}, err
			}
		default:
			return models_handler_storage.Config{}, fmt.Errorf("unknown data type '%s'", moduleConfig.DataType) // TODO
		}
	}
	return config, nil
}

func validateValue[T comparable](val T, moduleConfig models_external.ModuleConfig) error {
	err := module_lib_validation_configs.ValidateValue(moduleConfig.Type, moduleConfig.TypeOpt, val)
	if err != nil {
		return err
	}
	if moduleConfig.Options != nil && !moduleConfig.OptExt {
		ok, err := module_lib_validation_configs.ValidateValueInOptions(val, moduleConfig.Options)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("not in options") // TODO
		}
	}
	return nil
}

func toString(val any) (string, error) {
	v, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("invalid data type '%T'", val) // TODO
	}
	return v, nil
}

func toBool(val any) (bool, error) {
	v, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("invalid data type '%T'", val) // TODO
	}
	return v, nil
}

func float64ToInt64(val float64) (int64, error) {
	i, fr := math.Modf(val)
	if fr > 0 {
		return 0, fmt.Errorf("invalid data type '%T'", val) // TODO
	}
	return int64(i), nil
}

func toInt64(val any) (int64, error) {
	var i int64
	var err error
	switch v := val.(type) {
	case int:
		i = int64(v)
	case int8:
		i = int64(v)
	case int16:
		i = int64(v)
	case int32:
		i = int64(v)
	case int64:
		i = v
	case float32:
		i, err = float64ToInt64(float64(v))
	case float64:
		i, err = float64ToInt64(v)
	default:
		err = fmt.Errorf("invalid data type '%T'", val) // TODO
	}
	return i, err
}

func toFloat64(val any) (float64, error) {
	var f float64
	switch v := val.(type) {
	case float32:
		f = float64(v)
	case float64:
		f = v
	default:
		return f, fmt.Errorf("invalid data type '%T'", val) // TODO
	}
	return f, nil
}
