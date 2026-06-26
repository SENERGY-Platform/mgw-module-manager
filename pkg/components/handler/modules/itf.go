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

	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

type databaseHandler interface {
	ReadModules(ctx context.Context, filter pkg_models.ModulesFilter) (map[string]pkg_models.DatabaseModule, error)
	ReadModule(ctx context.Context, id string) (pkg_models.DatabaseModule, error)
	CreateModule(ctx context.Context, mod pkg_models.DatabaseModule) error
	UpdateModule(ctx context.Context, mod pkg_models.DatabaseModule) error
	DeleteModule(ctx context.Context, id string) error
}

type containerEngineWrapperClient interface {
	GetImage(ctx context.Context, id string) (external_models.CewImage, error)
	AddImage(ctx context.Context, img string) (jobId string, err error)
	RemoveImage(ctx context.Context, id string) error
	GetJob(ctx context.Context, id string) (external_models.JobLibJob, error)
	CancelJob(ctx context.Context, id string) error
}
