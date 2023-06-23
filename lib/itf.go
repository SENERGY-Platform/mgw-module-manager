/*
 * Copyright 2023 InfAI (CC SES)
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

package lib

import (
	"context"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

type Api interface {
	AddModule(ctx context.Context, mr model.ModAddRequest) (string, error)
	GetModules(ctx context.Context, filter model.ModFilter) ([]model.ModuleMeta, error)
	GetModule(ctx context.Context, id string) (model.Module, error)
	DeleteModule(ctx context.Context, id string, orphans, force bool) error
	GetModuleDeployTemplate(ctx context.Context, id string) (model.ModDeployTemplate, error)
	CheckModuleUpdates(ctx context.Context) (string, error)
	GetModuleUpdates(ctx context.Context) map[string]model.ModUpdateInfo
	GetModuleUpdate(ctx context.Context, id string) (model.ModUpdateInfo, error)
	CancelPendingModuleUpdate(ctx context.Context, id string) error
	CreateDeployment(ctx context.Context, dr model.DepRequest) (string, error)
	GetDeployments(ctx context.Context, filter model.DepFilter) ([]model.DepMeta, error)
	GetDeployment(ctx context.Context, id string) (*model.Deployment, error)
	StartDeployment(ctx context.Context, id string) error
	StopDeployment(ctx context.Context, id string, dependencies bool) (string, error)
	UpdateDeployment(ctx context.Context, id string, dr model.DepRequestBase) (string, error)
	DeleteDeployment(ctx context.Context, id string, orphans bool) error
	GetDeploymentUpdateTemplate(ctx context.Context, id string) (model.DepUpdateTemplate, error)
	GetJobs(ctx context.Context, filter model.JobFilter) ([]model.Job, error)
	GetJob(ctx context.Context, id string) (model.Job, error)
	CancelJob(ctx context.Context, id string) error
}
