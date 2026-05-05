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

package service

import (
	"context"

	lib_service "github.com/SENERGY-Platform/mgw-module-manager/lib/models/service"
	helper_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/configs"
	models_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/configs"
)

func (s *Service) CreateGlobalConfig(ctx context.Context, input lib_service.GlobalConfigInput) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := helper_configs.GetValue(input.Value, input.DataType, input.IsSlice)
	if err != nil {
		return "", err
	}
	return s.globalConfigsHandler.CreateGlobalConfig(ctx, input.Name, value)
}

func (s *Service) ReadGlobalConfig(ctx context.Context, id string) (lib_service.GlobalConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	config, err := s.globalConfigsHandler.ReadGlobalConfig(ctx, id)
	if err != nil {
		return lib_service.GlobalConfig{}, err
	}
	return newGlobalConfig(config), nil
}

func (s *Service) ReadGlobalConfigs(ctx context.Context, ids []string) (map[string]lib_service.GlobalConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tmp, err := s.globalConfigsHandler.ReadGlobalConfigs(ctx, ids)
	if err != nil {
		return nil, err
	}
	globalConfigs := make(map[string]lib_service.GlobalConfig)
	for id, tmpConfig := range tmp {
		globalConfigs[id] = newGlobalConfig(tmpConfig)
	}
	return globalConfigs, nil
}

func (s *Service) UpdateGlobalConfig(ctx context.Context, config lib_service.GlobalConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := helper_configs.GetValue(config.Value, config.DataType, config.IsSlice)
	if err != nil {
		return err
	}
	return s.globalConfigsHandler.UpdateGlobalConfig(ctx, models_configs.Config{
		Id:    config.Id,
		Name:  config.Name,
		Value: value,
	})
}

func (s *Service) DeleteGlobalConfig(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.globalConfigsHandler.DeleteGlobalConfig(ctx, id)
}

func (s *Service) DeleteGlobalConfigs(ctx context.Context, ids []string, allowAll bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.globalConfigsHandler.DeleteGlobalConfigs(ctx, ids, allowAll)
}

func newGlobalConfig(config models_configs.Config) lib_service.GlobalConfig {
	return lib_service.GlobalConfig{
		Id:   config.Id,
		Name: config.Name,
		InterfaceValue: lib_service.InterfaceValue{
			DataType: config.DataType,
			IsSlice:  config.IsSlice,
			Value:    helper_configs.ValueToInterface(config.Value),
		},
	}
}
