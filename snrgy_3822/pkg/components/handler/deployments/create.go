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
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"

	module_lib_validation_configs "github.com/SENERGY-Platform/mgw-module-lib/validation/configs"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	helper_uuid "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/uuid"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_deployment "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployment"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func newDeploymentStorage(module models_handler_module.Module, userInput models_handler_deployment.UserInput) (deploymentWrapper, error) {
	id, err := helper_uuid.New()
	if err != nil {
		return deploymentWrapper{}, err
	}
	dirName, err := helper_uuid.New()
	if err != nil {
		return deploymentWrapper{}, err
	}
	name := userInput.Name
	if name == "" {
		name = module.Name
	}
	configs, err := newConfigsStorage(module.Configs, userInput.Configs, id)
	if err != nil {
		return deploymentWrapper{}, err
	}
	return deploymentWrapper{
		Deployment: models_handler_storage.Deployment{
			Id:            id,
			ModuleId:      module.ID,
			ModuleSource:  module.Source,
			ModuleChannel: module.Channel,
			ModuleVersion: module.Version,
			Name:          name,
			DirName:       dirName,
			Enabled:       false,
			Created:       helper_time.Now(),
		},
		HostResources:    newHostResourcesStorage(module.HostResources, userInput.HostResources, id),
		Secrets:          newSecretsStorage(module.Secrets, module.Services, userInput.Secrets, id),
		Configs:          configs,
		GlobalConfigs:    newGlobalConfigsStorage(module.Configs, userInput.GlobalConfigs, id),
		Module:           module.Module,
		ModuleFileSystem: module.FileSystem,
	}, nil
}

func newSecretsStorage(moduleSecrets map[string]models_external.ModuleSecret, moduleServices map[string]*models_external.ModuleService, userInputs map[string]string, deploymentID string) []models_handler_storage.DeploymentSecret {
	var secrets []models_handler_storage.DeploymentSecret
	for reference := range moduleSecrets {
		id, ok := userInputs[reference]
		if ok {
			secrets = append(secrets, models_handler_storage.DeploymentSecret{
				Id:           id,
				DeploymentId: deploymentID,
				Reference:    reference,
				Items:        newSecretItemsStorage(reference, moduleServices),
			})
		}
	}
	return secrets
}

func newSecretItemsStorage(reference string, moduleServices map[string]*models_external.ModuleService) []models_handler_storage.DeploymentSecretItem {
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

func newHostResourcesStorage(moduleHostResources map[string]models_external.ModuleHostResource, userInputs map[string]string, deploymentID string) []models_handler_storage.DeploymentHostResource {
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

func newGlobalConfigsStorage(moduleConfigs models_external.ModuleConfigs, userInputs map[string]string, deploymentId string) []models_handler_storage.DeploymentGlobalConfig {
	var configs []models_handler_storage.DeploymentGlobalConfig
	for reference := range moduleConfigs {
		id, ok := userInputs[reference]
		if ok {
			configs = append(configs, models_handler_storage.DeploymentGlobalConfig{
				Id:           id,
				DeploymentId: deploymentId,
				Reference:    reference,
			})
		}
	}
	return configs
}

func newConfigsStorage(moduleConfigs models_external.ModuleConfigs, userInputs map[string]any, deploymentId string) (map[string]models_handler_storage.DeploymentConfig, error) {
	configs := make(map[string]models_handler_storage.DeploymentConfig)
	for reference, moduleConfig := range moduleConfigs {
		val, ok := userInputs[reference]
		if ok && val != nil {
			config, err := newConfigStorage(val, reference, moduleConfig, deploymentId)
			if err != nil {
				return nil, err
			}
			configs[reference] = config
		}
	}
	return configs, nil
}

func newConfigStorage(val any, reference string, moduleConfigValue models_external.ModuleConfig, deploymentId string) (models_handler_storage.DeploymentConfig, error) {
	config := models_handler_storage.DeploymentConfig{
		DeploymentId: deploymentId,
		Reference:    reference,
		Config: models_handler_storage.Config{
			Id:      deploymentId + "_" + reference,
			IsSlice: moduleConfigValue.IsSlice,
		},
	}
	if moduleConfigValue.IsSlice {
		anySlice, ok := val.([]any)
		if !ok {
			return models_handler_storage.DeploymentConfig{}, fmt.Errorf("invalid data type '%T'", val) // TODO
		}
		switch moduleConfigValue.DataType {
		case models_external.ModuleConfigStringType:
			config.DataType = models_handler_storage.StringType
			for _, item := range anySlice {
				v, err := toString(item)
				if err != nil {
					return models_handler_storage.DeploymentConfig{}, err
				}
				err = validateValue(v, moduleConfigValue)
				if err != nil {
					return models_handler_storage.DeploymentConfig{}, err
				}
				config.StringSlice = append(config.StringSlice, v)
			}
		case models_external.ModuleConfigBoolType:
			config.DataType = models_handler_storage.BoolType
			for _, item := range anySlice {
				v, err := toBool(item)
				if err != nil {
					return models_handler_storage.DeploymentConfig{}, err
				}
				err = validateValue(v, moduleConfigValue)
				if err != nil {
					return models_handler_storage.DeploymentConfig{}, err
				}
				config.BoolSlice = append(config.BoolSlice, v)
			}
		case models_external.ModuleConfigInt64Type:
			config.DataType = models_handler_storage.Int64Type
			for _, item := range anySlice {
				v, err := toInt64(item)
				if err != nil {
					return models_handler_storage.DeploymentConfig{}, err
				}
				err = validateValue(v, moduleConfigValue)
				if err != nil {
					return models_handler_storage.DeploymentConfig{}, err
				}
				config.Int64Slice = append(config.Int64Slice, v)
			}
		case models_external.ModuleConfigFloat64Type:
			config.DataType = models_handler_storage.Float64Type
			for _, item := range anySlice {
				v, err := toFloat64(item)
				if err != nil {
					return models_handler_storage.DeploymentConfig{}, err
				}
				err = validateValue(v, moduleConfigValue)
				if err != nil {
					return models_handler_storage.DeploymentConfig{}, err
				}
				config.Float64Slice = append(config.Float64Slice, v)
			}
		default:
			return models_handler_storage.DeploymentConfig{}, fmt.Errorf("unknown data type '%s'", moduleConfigValue.DataType) // TODO
		}
	} else {
		switch moduleConfigValue.DataType {
		case models_external.ModuleConfigStringType:
			config.DataType = models_handler_storage.StringType
			v, err := toString(val)
			if err != nil {
				return models_handler_storage.DeploymentConfig{}, err
			}
			config.String = v
			err = validateValue(v, moduleConfigValue)
			if err != nil {
				return models_handler_storage.DeploymentConfig{}, err
			}
		case models_external.ModuleConfigBoolType:
			config.DataType = models_handler_storage.BoolType
			v, err := toBool(val)
			if err != nil {
				return models_handler_storage.DeploymentConfig{}, err
			}
			config.Bool = v
			err = validateValue(v, moduleConfigValue)
			if err != nil {
				return models_handler_storage.DeploymentConfig{}, err
			}
		case models_external.ModuleConfigInt64Type:
			config.DataType = models_handler_storage.Int64Type
			v, err := toInt64(val)
			if err != nil {
				return models_handler_storage.DeploymentConfig{}, err
			}
			config.Int64 = v
			err = validateValue(v, moduleConfigValue)
			if err != nil {
				return models_handler_storage.DeploymentConfig{}, err
			}
		case models_external.ModuleConfigFloat64Type:
			config.DataType = models_handler_storage.Float64Type
			v, err := toFloat64(val)
			if err != nil {
				return models_handler_storage.DeploymentConfig{}, err
			}
			config.Float64 = v
			err = validateValue(v, moduleConfigValue)
			if err != nil {
				return models_handler_storage.DeploymentConfig{}, err
			}
		default:
			return models_handler_storage.DeploymentConfig{}, fmt.Errorf("unknown data type '%s'", moduleConfigValue.DataType) // TODO
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
