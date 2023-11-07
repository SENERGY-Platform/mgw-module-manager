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
	AddModule(ctx context.Context, mID, version string) (string, error)
	GetModules(ctx context.Context, filter model.ModFilter) ([]model.ModuleMeta, error)
	GetModule(ctx context.Context, mID string) (model.Module, error)
	DeleteModule(ctx context.Context, mID string, orphans, force bool) error
	GetModuleDeployTemplate(ctx context.Context, mID string) (model.ModDeployTemplate, error)
	CheckModuleUpdates(ctx context.Context) (string, error)
	GetModuleUpdates(ctx context.Context) (map[string]model.ModUpdate, error)
	GetModuleUpdate(ctx context.Context, mID string) (model.ModUpdate, error)
	PrepareModuleUpdate(ctx context.Context, mID, version string) (string, error)
	CancelPendingModuleUpdate(ctx context.Context, mID string) error
	UpdateModule(ctx context.Context, mID string, depInput model.DepInput, dependencies map[string]model.DepInput, orphans bool) (string, error)
	GetModuleUpdateTemplate(ctx context.Context, id string) (model.ModUpdateTemplate, error)
	CreateDeployment(ctx context.Context, mID string, depInput model.DepInput, dependencies map[string]model.DepInput) (string, error)
	GetDeployments(ctx context.Context, filter model.DepFilter) ([]model.DepBase, error)
	GetDeployment(ctx context.Context, dID string) (model.Deployment, error)
	UpdateDeployment(ctx context.Context, dID string, depInput model.DepInput) (string, error)
	DeleteDeployment(ctx context.Context, dID string, force bool) error
	DeleteDeployments(ctx context.Context, dIDs []string, force bool) error
	StartDeployment(ctx context.Context, dID string, dependencies bool) error
	StartDeployments(ctx context.Context, dIDs []string, dependencies bool) error
	StartAllDeployments(ctx context.Context, filter model.DepFilter, dependencies bool) error
	StopDeployment(ctx context.Context, dID string, force bool) (string, error)
	StopDeployments(ctx context.Context, dIDs []string, force bool) (string, error)
	StopAllDeployments(ctx context.Context, filter model.DepFilter, force bool) (string, error)
	RestartDeployment(ctx context.Context, dID string) (string, error)
	RestartDeployments(ctx context.Context, dIDs []string) (string, error)
	RestartAllDeployments(ctx context.Context, filter model.DepFilter) (string, error)
	GetDeploymentUpdateTemplate(ctx context.Context, dID string) (model.DepUpdateTemplate, error)
	GetDeploymentsHealth(ctx context.Context) (map[string]model.DepHealthInfo, error)
	GetDeploymentHealth(ctx context.Context, dID string) (model.DepHealthInfo, error)
	GetJobs(ctx context.Context, filter model.JobFilter) ([]model.Job, error)
	GetJob(ctx context.Context, jID string) (model.Job, error)
	CancelJob(ctx context.Context, jID string) error
}
