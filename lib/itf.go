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
	job_hdl_lib "github.com/SENERGY-Platform/mgw-go-service-base/job-hdl/lib"
	srv_info_lib "github.com/SENERGY-Platform/mgw-go-service-base/srv-info-hdl/lib"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

type Api interface {
	AddModule(ctx context.Context, mID, version string) (string, error)
	GetModules(ctx context.Context, filter model.ModFilter) (map[string]model.Module, error)
	GetModule(ctx context.Context, mID string) (model.Module, error)
	DeleteModule(ctx context.Context, mID string, force bool) (string, error)
	GetModuleDeployTemplate(ctx context.Context, mID string) (model.ModDeployTemplate, error)
	CheckModuleUpdates(ctx context.Context) (string, error)
	GetModuleUpdates(ctx context.Context) (map[string]model.ModUpdate, error)
	GetModuleUpdate(ctx context.Context, mID string) (model.ModUpdate, error)
	PrepareModuleUpdate(ctx context.Context, mID, version string) (string, error)
	CancelPendingModuleUpdate(ctx context.Context, mID string) error
	UpdateModule(ctx context.Context, mID string, depInput model.DepInput, dependencies map[string]model.DepInput) (string, error)
	GetModuleUpdateTemplate(ctx context.Context, id string) (model.ModUpdateTemplate, error)
	CreateDeployment(ctx context.Context, mID string, depInput model.DepInput, dependencies map[string]model.DepInput) (string, error)
	GetDeployments(ctx context.Context, filter model.DepFilter, assets, containerInfo bool) (map[string]model.Deployment, error)
	GetDeployment(ctx context.Context, dID string, assets, containerInfo bool) (model.Deployment, error)
	UpdateDeployment(ctx context.Context, dID string, depInput model.DepInput) (string, error)
	DeleteDeployment(ctx context.Context, dID string, force bool) (string, error)
	DeleteDeployments(ctx context.Context, filter model.DepFilter, force bool) (string, error)
	StartDeployment(ctx context.Context, dID string, dependencies bool) (string, error)
	StartDeployments(ctx context.Context, filter model.DepFilter, dependencies bool) (string, error)
	StopDeployment(ctx context.Context, dID string, force bool) (string, error)
	StopDeployments(ctx context.Context, filter model.DepFilter, force bool) (string, error)
	RestartDeployment(ctx context.Context, dID string) (string, error)
	RestartDeployments(ctx context.Context, filter model.DepFilter) (string, error)
	GetDeploymentUpdateTemplate(ctx context.Context, dID string) (model.DepUpdateTemplate, error)
	AuxDeploymentApi
	DepAdvertisementApi
	job_hdl_lib.Api
	srv_info_lib.Api
}

type AuxDeploymentApi interface {
	GetAuxDeployments(ctx context.Context, dID string, filter model.AuxDepFilter, assets, containerInfo bool) (map[string]model.AuxDeployment, error)
	GetAuxDeployment(ctx context.Context, dID, aID string, assets, containerInfo bool) (model.AuxDeployment, error)
	CreateAuxDeployment(ctx context.Context, dID string, auxDepInput model.AuxDepReq, forcePullImg bool) (string, error)
	UpdateAuxDeployment(ctx context.Context, dID, aID string, auxDepInput model.AuxDepReq, incremental, forcePullImg bool) (string, error)
	DeleteAuxDeployment(ctx context.Context, dID, aID string, force bool) (string, error)
	DeleteAuxDeployments(ctx context.Context, dID string, filter model.AuxDepFilter, force bool) (string, error)
	StartAuxDeployment(ctx context.Context, dID, aID string) (string, error)
	StartAuxDeployments(ctx context.Context, dID string, filter model.AuxDepFilter) (string, error)
	StopAuxDeployment(ctx context.Context, dID, aID string) (string, error)
	StopAuxDeployments(ctx context.Context, dID string, filter model.AuxDepFilter) (string, error)
	RestartAuxDeployment(ctx context.Context, dID, aID string) (string, error)
	RestartAuxDeployments(ctx context.Context, dID string, filter model.AuxDepFilter) (string, error)
	GetAuxJobs(ctx context.Context, dID string, filter job_hdl_lib.JobFilter) ([]job_hdl_lib.Job, error)
	GetAuxJob(ctx context.Context, dID string, jID string) (job_hdl_lib.Job, error)
	CancelAuxJob(ctx context.Context, dID string, jID string) error
}

type DepAdvertisementApi interface {
	QueryDepAdvertisements(ctx context.Context, filter model.DepAdvFilter) ([]model.DepAdvertisement, error)
	GetDepAdvertisement(ctx context.Context, dID, ref string) (model.DepAdvertisement, error)
	GetDepAdvertisements(ctx context.Context, dID string) (map[string]model.DepAdvertisement, error)
	PutDepAdvertisement(ctx context.Context, dID string, adv model.DepAdvertisementBase) error
	PutDepAdvertisements(ctx context.Context, dID string, ads map[string]model.DepAdvertisementBase) error
	DeleteDepAdvertisement(ctx context.Context, dID, ref string) error
	DeleteDepAdvertisements(ctx context.Context, dID string) error
}
