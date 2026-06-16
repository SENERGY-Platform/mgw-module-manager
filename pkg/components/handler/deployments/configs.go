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
	"strings"

	helper_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/configs"
	helper_errors "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/errors"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) updateGlobalConfigsCache(
	ctx context.Context,
	userDataGlobalConfigs map[string]pkg_models.DeploymentGlobalConfig,
	cacheGlobalConfigs map[string]pkg_models.Config,
) error {
	selectedIds := helper_slices.CollectFunc(maps.Values(userDataGlobalConfigs), func(item pkg_models.DeploymentGlobalConfig) string {
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
	var notFound []string
	for _, id := range idsNotInCache {
		if _, ok := cacheGlobalConfigs[id]; !ok {
			notFound = append(notFound, id)
		}
	}
	if len(notFound) > 0 {
		return errors.New(fmt.Sprintf("global configs not found: %s", strings.Join(notFound, ", ")))
	}
	return nil
}

func checkConfigs(
	moduleConfigs external_models.ModuleLibConfigs,
	configs map[string]pkg_models.Value,
) error {
	var required []string
	for reference, moduleConfig := range moduleConfigs {
		_, ok := configs[reference]
		if !ok && moduleConfig.Required {
			required = append(required, reference)
		}
	}
	if len(required) > 0 {
		return errors.New(fmt.Sprintf("required configs: %s", strings.Join(required, ", ")))
	}
	return nil
}

func mergeConfigs(
	defaultConfigs map[string]pkg_models.Value,
	userDataConfigs map[string]pkg_models.DeploymentUserConfig,
	userDataGlobalConfigs map[string]pkg_models.DeploymentGlobalConfig,
	cacheGlobalConfigs map[string]pkg_models.Config,
) map[string]pkg_models.Value {
	configs := make(map[string]pkg_models.Value)
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

func getDefaultConfigs(moduleConfigs external_models.ModuleLibConfigs) (map[string]pkg_models.Value, error) {
	configs := make(map[string]pkg_models.Value)
	var errs []error
	for reference, moduleConfig := range moduleConfigs {
		if moduleConfig.Default == nil {
			continue
		}
		value, err := helper_configs.GetValueModule(moduleConfig.Default, moduleConfig, false)
		if err != nil {
			errs = append(errs, fmt.Errorf("'%s' %w", reference, err))
			continue
		}
		configs[reference] = value
	}
	if len(errs) > 0 {
		return nil, helper_errors.Joinp("get default configs:", errs...)
	}
	return configs, nil
}

func getSelectedGlobalConfigs(
	moduleConfigs external_models.ModuleLibConfigs,
	userInputGlobalConfigs map[string]string,
	deploymentId string,
) map[string]pkg_models.DeploymentGlobalConfig {
	configs := make(map[string]pkg_models.DeploymentGlobalConfig)
	for reference := range moduleConfigs {
		id, ok := userInputGlobalConfigs[reference]
		if !ok {
			continue
		}
		configs[reference] = pkg_models.DeploymentGlobalConfig{
			Id:           id,
			DeploymentId: deploymentId,
			Reference:    reference,
		}
	}
	return configs
}

func getProvidedConfigs(
	moduleConfigs external_models.ModuleLibConfigs,
	defaultConfigs map[string]pkg_models.Value,
	userInputConfigs map[string]pkg_models.Value,
	deploymentId string,
) map[string]pkg_models.DeploymentUserConfig {
	configs := make(map[string]pkg_models.DeploymentUserConfig)
	for reference := range moduleConfigs {
		config, ok := userInputConfigs[reference]
		if !ok {
			continue
		}
		defaultConfig, ok := defaultConfigs[reference]
		if ok && helper_configs.ValueIsEqual(config, defaultConfig) {
			continue
		}
		configs[reference] = pkg_models.DeploymentUserConfig{
			DeploymentId: deploymentId,
			Reference:    reference,
			Id:           deploymentId + "_" + reference,
			Value:        config,
		}
	}
	return configs
}
