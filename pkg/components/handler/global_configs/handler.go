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
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
)

type Handler struct {
	databaseHandler databaseHandler
}

func New(databaseHandler databaseHandler) *Handler {
	return &Handler{databaseHandler: databaseHandler}
}

func (h *Handler) CreateGlobalConfig(ctx context.Context, name string, value pkg_models.Value) (string, error) {
	id, err := helper_uuid.New()
	if err != nil {
		logger.ErrorContext(ctx, "create global config, generate id", slog_keys.Name, name, slog_keys.Error, err)
		return "", err
	}
	err = h.databaseHandler.CreateGlobalConfig(ctx, pkg_models.Config{
		Id:    id,
		Value: value,
		Name:  name,
	})
	if err != nil {
		logger.ErrorContext(ctx, "create global config, write to database", slog_keys.Name, name, slog_keys.Error, err)
		return "", err
	}
	return id, nil
}

func (h *Handler) GetGlobalConfig(ctx context.Context, id string) (pkg_models.Config, error) {
	config, err := h.databaseHandler.ReadGlobalConfig(ctx, id)
	if err != nil {
		logger.ErrorContext(ctx, "get global config", slog_keys.GlobalConfigId, id, slog_keys.Error, err)
		return pkg_models.Config{}, err
	}
	return config, nil
}

func (h *Handler) GetGlobalConfigs(ctx context.Context, ids []string) (map[string]pkg_models.Config, error) {
	configs, err := h.databaseHandler.ReadGlobalConfigs(ctx, ids)
	if err != nil {
		logger.ErrorContext(ctx, "get global configs", slog_keys.Filter, ids, slog_keys.Error, err)
		return nil, err
	}
	return configs, nil
}

func (h *Handler) UpdateGlobalConfig(ctx context.Context, config pkg_models.Config) error {
	err := h.databaseHandler.UpdateGlobalConfig(ctx, config)
	if err != nil {
		logger.ErrorContext(ctx, "update global config", slog_keys.GlobalConfigId, config.Id, slog_keys.Error, err)
		return err
	}
	return nil
}

func (h *Handler) DeleteGlobalConfig(ctx context.Context, id string) error {
	err := h.databaseHandler.DeleteGlobalConfig(ctx, id)
	if err != nil {
		logger.ErrorContext(ctx, "delete global config", slog_keys.GlobalConfigId, id, slog_keys.Error, err)
		return err
	}
	return nil
}

func (h *Handler) DeleteGlobalConfigs(ctx context.Context, ids []string, allowAll bool) error {
	if !allowAll && len(ids) == 0 {
		return nil
	}
	if allowAll {
		logger.WarnContext(ctx, "delete global configs", slog_keys.Filter, ids, slog_keys.AllowAll, allowAll)
	}
	err := h.databaseHandler.DeleteGlobalConfigs(ctx, ids)
	if err != nil {
		logger.ErrorContext(ctx, "delete global configs", slog_keys.Filter, ids, slog_keys.AllowAll, allowAll, slog_keys.Error, err)
		return err
	}
	return nil
}
