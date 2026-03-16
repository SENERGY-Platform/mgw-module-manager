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
	"strconv"
	"strings"

	module_lib_validation_configs "github.com/SENERGY-Platform/mgw-module-lib/validation/configs"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_deployment "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployment"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) updateGlobalConfigsCache(
	ctx context.Context,
	userData userDataCollection,
	cache cacheCollection,
) error {
	selectedIds := helper_slices.CollectFunc(maps.Values(userData.GlobalConfigs), func(item models_handler_storage.DeploymentGlobalConfig) string {
		return item.Id
	})
	var idsNotInCache []string
	for _, id := range selectedIds {
		if _, ok := cache.GlobalConfigs[id]; ok {
			idsNotInCache = append(idsNotInCache, id)
		}
	}
	if len(idsNotInCache) == 0 {
		return nil
	}
	globalConfigs, err := h.storageHdl.ReadGlobalConfigs(ctx, idsNotInCache)
	if err != nil {
		return err
	}
	for id, globalConfig := range globalConfigs {
		cache.GlobalConfigs[id] = globalConfig
	}
	var errs []string
	for _, id := range idsNotInCache {
		if _, ok := cache.GlobalConfigs[id]; !ok {
			errs = append(errs, fmt.Sprintf("global config %s not found", id))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func configsToStrings(module models_handler_module.Module, configs map[string]models_handler_storage.Config) map[string]string {
	configValues := make(map[string]string)
	for reference, config := range configs {
		if config.IsSlice {
			moduleConfig := module.Configs[reference]
			configValues[reference] = sliceConfigToString(config, moduleConfig.Delimiter)
		} else {
			configValues[reference] = configToString(config)
		}
	}
	return configValues
}

func configToString(config models_handler_storage.Config) string {
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
	return ""
}

func sliceConfigToString(config models_handler_storage.Config, delimiter string) string {
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
}

func checkConfigs(
	module models_handler_module.Module,
	configs map[string]models_handler_storage.Config,
) error {
	var errs []string
	for reference, moduleConfig := range module.Configs {
		_, ok := configs[reference]
		if !ok && moduleConfig.Required {
			errs = append(errs, fmt.Sprintf("config %s required", reference))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func mergeConfigs(
	defaultData defaultDataCollection,
	userData userDataCollection,
	cache cacheCollection,
) map[string]models_handler_storage.Config {
	configs := make(map[string]models_handler_storage.Config)
	maps.Copy(configs, defaultData.Configs)
	for reference, providedConfig := range userData.Configs {
		configs[reference] = providedConfig.Config
	}
	for reference, selectedGlobalConfig := range userData.GlobalConfigs {
		globalConfig, ok := cache.GlobalConfigs[selectedGlobalConfig.Id]
		if ok {
			configs[reference] = globalConfig.Config
		}
	}
	return configs
}

func getDefaultConfigs(module models_handler_module.Module) (map[string]models_handler_storage.Config, error) {
	configs := make(map[string]models_handler_storage.Config)
	var errs []string
	for reference, moduleConfig := range module.Configs {
		if moduleConfig.Default == nil {
			continue
		}
		config, err := getConfig(moduleConfig.Default, moduleConfig)
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

func getSelectedGlobalConfigs(
	module models_handler_module.Module,
	userInputs models_handler_deployment.UserInput,
	deploymentId string,
) map[string]models_handler_storage.DeploymentGlobalConfig {
	configs := make(map[string]models_handler_storage.DeploymentGlobalConfig)
	for reference := range module.Configs {
		id, ok := userInputs.GlobalConfigs[reference]
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

func getProvidedConfigs(
	module models_handler_module.Module,
	defaultConfigs defaultDataCollection,
	userInputs models_handler_deployment.UserInput,
	deploymentId string,
) (map[string]models_handler_storage.DeploymentUserConfig, error) {
	configs := make(map[string]models_handler_storage.DeploymentUserConfig)
	var errs []string
	for reference, moduleConfig := range module.Configs {
		val, ok := userInputs.Configs[reference]
		if !ok || val == nil {
			continue
		}
		config, err := getConfig(val, moduleConfig)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		defaultConfig, ok := defaultConfigs.Configs[reference]
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

func getConfig(val any, moduleConfig models_external.ModuleConfig) (models_handler_storage.Config, error) {
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
