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

package handler_deployments

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/configs"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/config"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/global_configs"
)

func (h *Handler) updateGlobalConfigsCache(
	ctx context.Context,
	userDataGlobalConfigs map[string]models_handler_database.DeploymentGlobalConfig,
	cacheGlobalConfigs map[string]models_handler_global_configs.Config,
) error {
	selectedIds := helper_slices.CollectFunc(maps.Values(userDataGlobalConfigs), func(item models_handler_database.DeploymentGlobalConfig) string {
		return item.Id
	})
	var idsNotInCache []string
	for _, id := range selectedIds {
		if _, ok := cacheGlobalConfigs[id]; ok {
			idsNotInCache = append(idsNotInCache, id)
		}
	}
	if len(idsNotInCache) == 0 {
		return nil
	}
	globalConfigs, err := h.databaseHandler.ReadGlobalConfigs(ctx, idsNotInCache)
	if err != nil {
		return err
	}
	for id, globalConfig := range globalConfigs {
		cacheGlobalConfigs[id] = globalConfig
	}
	var errs []string
	for _, id := range idsNotInCache {
		if _, ok := cacheGlobalConfigs[id]; !ok {
			errs = append(errs, fmt.Sprintf("global config %s not found", id))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func checkConfigs(
	moduleConfigs models_external.ModuleLibConfigs,
	configs map[string]models_config.Value,
) error {
	var errs []string
	for reference, moduleConfig := range moduleConfigs {
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
	defaultConfigs map[string]models_config.Value,
	userDataConfigs map[string]models_handler_database.DeploymentUserConfig,
	userDataGlobalConfigs map[string]models_handler_database.DeploymentGlobalConfig,
	cacheGlobalConfigs map[string]models_handler_global_configs.Config,
) map[string]models_config.Value {
	configs := make(map[string]models_config.Value)
	maps.Copy(configs, defaultConfigs)
	for reference, providedConfig := range userDataConfigs {
		configs[reference] = providedConfig.Value
	}
	for reference, selectedGlobalConfig := range userDataGlobalConfigs {
		globalConfig, ok := cacheGlobalConfigs[selectedGlobalConfig.Id]
		if ok {
			configs[reference] = globalConfig.Value
		}
	}
	return configs
}

func getDefaultConfigs(moduleConfigs models_external.ModuleLibConfigs) (map[string]models_config.Value, error) {
	configs := make(map[string]models_config.Value)
	var errs []string
	for reference, moduleConfig := range moduleConfigs {
		if moduleConfig.Default == nil {
			continue
		}
		value, err := helper_configs.GetValueWithValidation(moduleConfig.Default, moduleConfig)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		configs[reference] = value
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return configs, nil
}

func getSelectedGlobalConfigs(
	moduleConfigs models_external.ModuleLibConfigs,
	userInputGlobalConfigs map[string]string,
	deploymentId string,
) map[string]models_handler_database.DeploymentGlobalConfig {
	configs := make(map[string]models_handler_database.DeploymentGlobalConfig)
	for reference := range moduleConfigs {
		id, ok := userInputGlobalConfigs[reference]
		if !ok {
			continue
		}
		configs[reference] = models_handler_database.DeploymentGlobalConfig{
			Id:           id,
			DeploymentId: deploymentId,
			Reference:    reference,
		}
	}
	return configs
}

func getProvidedConfigs(
	moduleConfigs models_external.ModuleLibConfigs,
	defaultConfigs map[string]models_config.Value,
	userInputConfigs map[string]models_config.Value,
	deploymentId string,
) (map[string]models_handler_database.DeploymentUserConfig, error) {
	configs := make(map[string]models_handler_database.DeploymentUserConfig)
	var errs []string
	for reference := range moduleConfigs {
		config, ok := userInputConfigs[reference]
		if !ok {
			continue
		}
		defaultConfig, ok := defaultConfigs[reference]
		if ok && helper_configs.ValueIsEqual(config, defaultConfig) {
			continue
		}
		configs[reference] = models_handler_database.DeploymentUserConfig{
			DeploymentId: deploymentId,
			Reference:    reference,
			Id:           deploymentId + "_" + reference,
			Value:        config,
		}
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return configs, nil
}
