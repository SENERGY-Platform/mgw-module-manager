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

package global_configs

import (
	"context"

	helper_uuid "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/uuid"
	models_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/configs"
)

type Handler struct {
	databaseHandler databaseHandler
}

func New(databaseHandler databaseHandler) *Handler {
	return &Handler{databaseHandler: databaseHandler}
}

func (h *Handler) CreateGlobalConfig(ctx context.Context, name string, value models_configs.Value) (string, error) {
	id, err := helper_uuid.New()
	if err != nil {
		return "", err
	}
	err = h.databaseHandler.CreateGlobalConfig(ctx, models_configs.Config{
		Id:    id,
		Value: value,
		Name:  name,
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

func (h *Handler) ReadGlobalConfig(ctx context.Context, id string) (models_configs.Config, error) {
	return h.databaseHandler.ReadGlobalConfig(ctx, id)
}

func (h *Handler) ReadGlobalConfigs(ctx context.Context, ids []string) (map[string]models_configs.Config, error) {
	return h.databaseHandler.ReadGlobalConfigs(ctx, ids)
}

func (h *Handler) UpdateGlobalConfig(ctx context.Context, config models_configs.Config) error {
	return h.databaseHandler.UpdateGlobalConfig(ctx, config)
}

func (h *Handler) DeleteGlobalConfig(ctx context.Context, id string) error {
	return h.databaseHandler.DeleteGlobalConfig(ctx, id)
}

func (h *Handler) DeleteGlobalConfigs(ctx context.Context, ids []string, allowAll bool) error {
	if !allowAll && len(ids) == 0 {
		return nil
	}
	return h.databaseHandler.DeleteGlobalConfigs(ctx, ids)
}
