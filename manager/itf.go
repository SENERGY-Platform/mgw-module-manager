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

package manager

import (
	"context"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"time"
)

type ModuleHandler interface {
	List(ctx context.Context, filter lib_model.ModFilter, dependencyInfo bool) (map[string]model.Module, error)
	Get(ctx context.Context, mID string, dependencyInfo bool) (model.Module, error)
	Add(ctx context.Context, mod *module_lib.Module, modDir dir_fs.DirFS, modFile string) error
	Update(ctx context.Context, mod *module_lib.Module, modDir dir_fs.DirFS, modFile string) error
	Delete(ctx context.Context, mID string, force bool) error
	GetTree(ctx context.Context, mID string) (map[string]model.Module, error)
	AppendModTree(ctx context.Context, tree map[string]model.Module) error
}

type ModStagingHandler interface {
	Prepare(ctx context.Context, modules map[string]*module_lib.Module, mID, ver string) (model.Stage, error)
}

type ModUpdateHandler interface {
	Check(ctx context.Context, modules map[string]*module_lib.Module) error
	List(ctx context.Context) map[string]lib_model.ModUpdate
	Get(ctx context.Context, mID string) (lib_model.ModUpdate, error)
	Remove(ctx context.Context, mID string) error
	Prepare(ctx context.Context, modules map[string]*module_lib.Module, stage model.Stage, mID string) error
	GetPending(ctx context.Context, mID string) (model.Stage, map[string]struct{}, map[string]struct{}, map[string]struct{}, error)
	CancelPending(ctx context.Context, mID string) error
}

type DeploymentHandler interface {
	List(ctx context.Context, filter lib_model.DepFilter, dependencyInfo, assets, containers, containerInfo bool) (map[string]lib_model.Deployment, error)
	Get(ctx context.Context, dID string, dependencyInfo, assets, containers, containerInfo bool) (lib_model.Deployment, error)
	Create(ctx context.Context, mod *module_lib.Module, depReq lib_model.DepInput, incl dir_fs.DirFS, indirect bool) (string, error)
	Delete(ctx context.Context, dID string, force bool) error
	DeleteAll(ctx context.Context, filter lib_model.DepFilter, force bool) ([]string, error)
	Update(ctx context.Context, dID string, mod *module_lib.Module, depReq lib_model.DepInput, incl dir_fs.DirFS) error
	Start(ctx context.Context, dID string, dependencies bool) ([]string, error)
	StartAll(ctx context.Context, filter lib_model.DepFilter, dependencies bool) ([]string, error)
	Stop(ctx context.Context, dID string, force bool) error
	StopAll(ctx context.Context, filter lib_model.DepFilter, force bool) ([]string, error)
	Restart(ctx context.Context, id string) error
	RestartAll(ctx context.Context, filter lib_model.DepFilter) ([]string, error)
}

type AuxDeploymentHandler interface {
	List(ctx context.Context, dID string, filter lib_model.AuxDepFilter, assets, containerInfo bool) (map[string]lib_model.AuxDeployment, error)
	Get(ctx context.Context, dID, aID string, assets, containerInfo bool) (lib_model.AuxDeployment, error)
	Create(ctx context.Context, mod *module_lib.Module, dep lib_model.Deployment, requiredDep map[string]lib_model.Deployment, auxReq lib_model.AuxDepReq, forcePullImg bool) (string, error)
	Update(ctx context.Context, aID string, mod *module_lib.Module, dep lib_model.Deployment, requiredDep map[string]lib_model.Deployment, auxReq lib_model.AuxDepReq, forcePullImg, incremental bool) error
	UpdateAll(ctx context.Context, mod *module_lib.Module, dep lib_model.Deployment, requiredDep map[string]lib_model.Deployment) ([]string, error)
	Delete(ctx context.Context, dID, aID string, force bool) error
	DeleteAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter, force bool) ([]string, error)
	Start(ctx context.Context, dID, aID string) error
	StartAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter) ([]string, error)
	Stop(ctx context.Context, dID, aID string, noStore bool) error
	StopAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter, noStore bool) ([]string, error)
	Restart(ctx context.Context, dID, aID string) error
	RestartAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter) ([]string, error)
}

type AuxJobHandler interface {
	Add(dID, jID string)
	Check(dID, jID string) bool
	Purge(maxAge time.Duration)
}

type DepAdvertisementHandler interface {
	List(ctx context.Context, filter lib_model.DepAdvFilter) ([]lib_model.DepAdvertisement, error)
	Get(ctx context.Context, dID, ref string) (lib_model.DepAdvertisement, error)
	GetAll(ctx context.Context, dID string) (map[string]lib_model.DepAdvertisement, error)
	Put(ctx context.Context, mID, dID string, adv lib_model.DepAdvertisementBase) error
	PutAll(ctx context.Context, mID, dID string, ads map[string]lib_model.DepAdvertisementBase) error
	Delete(ctx context.Context, dID, ref string) error
	DeleteAll(ctx context.Context, dID string) error
}
