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
	"math"
	"slices"

	module_lib_validation_configs "github.com/SENERGY-Platform/mgw-module-lib/validation/configs"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	helper_uuid "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/uuid"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_deployment "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployment"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) CreateDeployments(ctx context.Context, modules map[string]models_handler_module.Module, userInputs map[string]models_handler_deployment.UserInput) (map[string]models_handler_deployment.Deployment, error) {
	deployments := newDeploymentWrappers(modules, userInputs)
	deploymentDependenciesCache := make(map[string]models_handler_storage.Deployment)
	hostResourcesCache := make(map[string]models_external.HostResource)
	globalConfigsCache := make(map[string]models_handler_storage.GlobalConfig)
	for _, deployment := range deployments {
		if deployment.Error != nil {
			continue
		}
		userInput := userInputs[deployment.Module.ID]
		deploymentUserConfigs, err := getDeploymentUserConfigs(deployment.Module.Configs, userInput.Configs, deployment.Id)
		if err != nil {
			deployment.Error = err
			continue
		}
		deploymentGlobalConfigs := newDeploymentGlobalConfigs(deployment.Module.Configs, userInput.GlobalConfigs, deployment.Id)
		deploymentHostResources := newDeploymentHostResources(deployment.Module.HostResources, userInput.HostResources, deployment.Id)
		deploymentSecrets := newDeploymentSecrets(deployment.Module.Secrets, deployment.Module.Services, userInput.Secrets, deployment.Id)
		deployment.Error = h.storageHdl.CreateDeployment(
			ctx,
			deployment.Deployment,
			deploymentHostResources,
			deploymentSecrets,
			slices.Collect(maps.Values(deploymentUserConfigs)),
			slices.Collect(maps.Values(deploymentGlobalConfigs)),
		)
		if deployment.Error != nil {
			continue
		}
		deployment.Error = h.getDeploymentDependencies(
			ctx,
			deploymentDependenciesCache,
			slices.Collect(maps.Keys(deployment.Module.Dependencies)),
		)
		if deployment.Error != nil {
			continue
		}
		deployment.Error = h.getHostResources(
			ctx,
			hostResourcesCache,
			helper_slices.CollectFunc(slices.Values(deploymentHostResources), func(item models_handler_storage.DeploymentHostResource) string {
				return item.Id
			}),
		)
		if deployment.Error != nil {
			continue
		}
		deployment.Error = h.getGlobalConfigs(
			ctx,
			globalConfigsCache,
			helper_slices.CollectFunc(maps.Values(deploymentGlobalConfigs), func(item models_handler_storage.DeploymentGlobalConfig) string {
				return item.Id
			}),
		)
		if deployment.Error != nil {
			continue
		}
		configs, err := getConfigs(deployment.Module.Configs, deploymentUserConfigs, deploymentGlobalConfigs, globalConfigsCache)
		if err != nil {
			deployment.Error = err
			continue
		}

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
		return errors.New(fmt.Sprintf("global confgis %v not found", idsNotFound)) // TODO
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
	var idsNotFound [][2]string
	for _, id := range idsNotInCache {
		hostResource, err := h.hmClient.GetHostResource(ctx, id)
		if err != nil {
			idsNotFound = append(idsNotFound, [2]string{id, err.Error()})
			continue
		}
		hostResourcesCache[hostResource.ID] = hostResource
	}
	if len(idsNotFound) > 0 {
		return errors.New(fmt.Sprintf("host resources %v not found", idsNotFound)) // TODO
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
		return errors.New(fmt.Sprintf("dependencies %v not found", idsNotFound)) // TODO
	}
	return nil
}

func getConfigs(
	moduleConfigs models_external.ModuleConfigs,
	deploymentUserConfigs map[string]models_handler_storage.DeploymentUserConfig,
	deploymentGlobalConfigs map[string]models_handler_storage.DeploymentGlobalConfig,
	globalConfigsCache map[string]models_handler_storage.GlobalConfig,
) (map[string]models_handler_storage.Config, error) {
	configs := make(map[string]models_handler_storage.Config)
	for reference, moduleConfig := range moduleConfigs {
		deploymentUserConfig, ok := deploymentUserConfigs[reference]
		if ok {
			configs[reference] = deploymentUserConfig.Config
			continue
		}
		deploymentGlobalConfig, ok := deploymentGlobalConfigs[reference]
		if ok {
			globalConfig, ok := globalConfigsCache[deploymentGlobalConfig.Id]
			if ok {
				configs[reference] = globalConfig.Config
				continue
			}
		}
		if moduleConfig.Default != nil {
			defaultConfig, err := moduleConfigValueToConfig(moduleConfig.Default, moduleConfig)
			if err != nil {
				return nil, err
			}
			configs[reference] = defaultConfig
			continue
		}
		if moduleConfig.Required {
			return nil, errors.New("required module config is missing") // TODO
		}
	}
	return configs, nil
}

func newDeploymentWrappers(modules map[string]models_handler_module.Module, userInputs map[string]models_handler_deployment.UserInput) map[string]*deploymentWrapper {
	deployments := make(map[string]*deploymentWrapper)
	for _, module := range modules {
		name := userInputs[module.ID].Name
		if name == "" {
			name = module.Name
		}
		deployment := &deploymentWrapper{
			Deployment: models_handler_storage.Deployment{
				ModuleId:      module.ID,
				ModuleSource:  module.Source,
				ModuleChannel: module.Channel,
				ModuleVersion: module.Version,
				Name:          name,
				Created:       helper_time.Now(),
			},
			Module:           module.Module,
			ModuleFileSystem: module.FileSystem,
		}
		deployment.Id, deployment.Error = helper_uuid.New()
		deployments[module.ID] = deployment
	}
	return deployments
}

func newDeploymentSecrets(moduleSecrets map[string]models_external.ModuleSecret, moduleServices map[string]*models_external.ModuleService, userInputs map[string]string, deploymentID string) []models_handler_storage.DeploymentSecret {
	var secrets []models_handler_storage.DeploymentSecret
	for reference := range moduleSecrets {
		id, ok := userInputs[reference]
		if ok {
			secrets = append(secrets, models_handler_storage.DeploymentSecret{
				Id:           id,
				DeploymentId: deploymentID,
				Reference:    reference,
				Items:        newDeploymentSecretItems(reference, moduleServices),
			})
		}
	}
	return secrets
}

func newDeploymentSecretItems(reference string, moduleServices map[string]*models_external.ModuleService) []models_handler_storage.DeploymentSecretItem {
	items := make(map[string]models_handler_storage.DeploymentSecretItem)
	for _, moduleService := range moduleServices {
		for _, target := range moduleService.SecretVars {
			if target.Ref == reference {
				var name string
				if target.Item != nil {
					name = *target.Item
				}
				item, ok := items[name]
				if !ok {
					item.Name = name
				}
				item.AsEnv = true
				items[name] = item
			}
		}
		for _, target := range moduleService.SecretMounts {
			if target.Ref == reference {
				var name string
				if target.Item != nil {
					name = *target.Item
				}
				item, ok := items[name]
				if !ok {
					item.Name = name
				}
				item.AsMount = true
				items[name] = item
			}
		}
	}
	return slices.Collect(maps.Values(items))
}

func newDeploymentHostResources(moduleHostResources map[string]models_external.ModuleHostResource, userInputs map[string]string, deploymentID string) []models_handler_storage.DeploymentHostResource {
	var hostResources []models_handler_storage.DeploymentHostResource
	for reference := range moduleHostResources {
		id, ok := userInputs[reference]
		if ok {
			hostResources = append(hostResources, models_handler_storage.DeploymentHostResource{
				Id:           id,
				DeploymentId: deploymentID,
				Reference:    reference,
			})
		}
	}
	return hostResources
}

func newDeploymentGlobalConfigs(moduleConfigs models_external.ModuleConfigs, userInputs map[string]string, deploymentId string) map[string]models_handler_storage.DeploymentGlobalConfig {
	configs := make(map[string]models_handler_storage.DeploymentGlobalConfig)
	for reference := range moduleConfigs {
		id, ok := userInputs[reference]
		if ok {
			configs[reference] = models_handler_storage.DeploymentGlobalConfig{
				Id:           id,
				DeploymentId: deploymentId,
				Reference:    reference,
			}
		}
	}
	return configs
}

func getDeploymentUserConfigs(moduleConfigs models_external.ModuleConfigs, userInputs map[string]any, deploymentId string) (map[string]models_handler_storage.DeploymentUserConfig, error) {
	configs := make(map[string]models_handler_storage.DeploymentUserConfig)
	for reference, moduleConfig := range moduleConfigs {
		val, ok := userInputs[reference]
		if ok && val != nil {
			config, err := moduleConfigValueToConfig(val, moduleConfig)
			if err != nil {
				return nil, err
			}
			config.Id = deploymentId + "_" + reference
			configs[reference] = models_handler_storage.DeploymentUserConfig{
				DeploymentId: deploymentId,
				Reference:    reference,
				Config:       config,
			}
		}
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
