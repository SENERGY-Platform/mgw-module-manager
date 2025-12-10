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

package modules

import (
	"context"

	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/storage"
)

type storageHandler interface {
	Modules(ctx context.Context, filter models_storage.ModulesFilter) (map[string]models_storage.Module, error)
	Module(ctx context.Context, id string) (models_storage.Module, error)
	CreateModule(ctx context.Context, mod models_storage.Module) error
	UpdateModule(ctx context.Context, mod models_storage.Module) error
	DeleteModule(ctx context.Context, id string) error
}

type containerEngineWrapperClient interface {
	GetImage(ctx context.Context, id string) (models_external.Image, error)
	AddImage(ctx context.Context, img string) (jobId string, err error)
	RemoveImage(ctx context.Context, id string) error
	GetJob(ctx context.Context, id string) (models_external.Job, error)
	CancelJob(ctx context.Context, id string) error
}
