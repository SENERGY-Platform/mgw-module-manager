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

package handler_aux_deployments

import (
	"errors"
	"maps"
	"strings"

	helper_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/configs"
	models_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/deployments"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func getDeploymentConfigs(
	moduleConfigs models_external.ModuleLibConfigs,
	moduleAuxServiceConfigs map[string]string,
	deploymentConfigs map[string]models_deployments.DeploymentUserConfig,
) (map[string]string, error) {
	configs := make(map[string]string)
	var errs []string
	for varName, reference := range moduleAuxServiceConfigs {
		moduleConfig, ok := moduleConfigs[reference]
		if !ok {
			continue
		}
		var value string
		deploymentConfig, ok := deploymentConfigs[reference]
		if ok {
			if moduleConfig.IsSlice {
				value = helper_configs.SliceValueToString(deploymentConfig.Value, moduleConfig.Delimiter)
			} else {
				value = helper_configs.ValueToString(deploymentConfig.Value)
			}
		} else {
			if moduleConfig.Default == nil {
				continue
			}
			defaultValue, err := helper_configs.GetValueWithValidation(moduleConfig.Default, moduleConfig)
			if err != nil {
				errs = append(errs, err.Error())
				continue
			}
			if moduleConfig.IsSlice {
				value = helper_configs.SliceValueToString(defaultValue, moduleConfig.Delimiter)
			} else {
				value = helper_configs.ValueToString(defaultValue)
			}
		}
		configs[varName] = value
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return configs, nil
}

func mergeConfigs(
	deploymentConfigs map[string]string,
	serviceInputConfigs map[string]string,
) map[string]string {
	configs := make(map[string]string)
	maps.Copy(configs, deploymentConfigs)
	maps.Copy(configs, serviceInputConfigs)
	return configs
}
