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

	models_handler_database "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

func (s *Service) CreateGlobalConfig(ctx context.Context, config models_handler_database.GlobalConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.globalConfigsHandler.CreateGlobalConfig(ctx, config)
}

func (s *Service) ReadGlobalConfig(ctx context.Context, id string) (models_handler_database.GlobalConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.globalConfigsHandler.ReadGlobalConfig(ctx, id)
}

func (s *Service) ReadGlobalConfigs(ctx context.Context, ids []string) (map[string]models_handler_database.GlobalConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.globalConfigsHandler.ReadGlobalConfigs(ctx, ids)
}

func (s *Service) UpdateGlobalConfig(ctx context.Context, config models_handler_database.GlobalConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.globalConfigsHandler.UpdateGlobalConfig(ctx, config)
}

func (s *Service) DeleteGlobalConfig(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.globalConfigsHandler.DeleteGlobalConfig(ctx, id)
}

func (s *Service) DeleteGlobalConfigs(ctx context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.globalConfigsHandler.DeleteGlobalConfigs(ctx, ids)
}
