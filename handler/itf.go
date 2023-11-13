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

package handler

import (
	"context"
	"database/sql/driver"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"io/fs"
)

type ModuleHandler interface {
	List(ctx context.Context, filter model.ModFilter) (map[string]model.Module, error)
	Get(ctx context.Context, mID string) (model.Module, error)
	GetReq(ctx context.Context, mID string) (model.Module, map[string]model.Module, error)
	GetDir(ctx context.Context, mID string) (dir_fs.DirFS, error)
	Add(ctx context.Context, mod *module.Module, modDir dir_fs.DirFS, modFile string, indirect bool) error
	Update(ctx context.Context, mod *module.Module, modDir dir_fs.DirFS, modFile string, indirect bool) error
	Delete(ctx context.Context, mID string, force bool) error
}

type ModFileHandler interface {
	GetModule(file fs.File) (*module.Module, error)
	GetModFile(dir dir_fs.DirFS) (fs.File, string, error)
}

type ModStorageHandler interface {
	List(ctx context.Context, filter model.ModFilter) (map[string]model.Module, error)
	Get(ctx context.Context, mID string) (model.Module, error)
	GetDir(ctx context.Context, mID string) (dir_fs.DirFS, error)
	Add(ctx context.Context, mod model.Module, modDir dir_fs.DirFS, modFile string) error
	Update(ctx context.Context, mod model.Module, modDir dir_fs.DirFS, modFile string) error
	Delete(ctx context.Context, mID string) error
}

type ModTransferHandler interface {
	Get(ctx context.Context, mID string) (ModRepo, error)
}

type ModStagingHandler interface {
	Prepare(ctx context.Context, modules map[string]*module.Module, mID, ver string) (Stage, error)
}

type ModUpdateHandler interface {
	Check(ctx context.Context, modules map[string]*module.Module) error
	List(ctx context.Context) map[string]model.ModUpdate
	Get(ctx context.Context, mID string) (model.ModUpdate, error)
	Remove(ctx context.Context, mID string) error
	Prepare(ctx context.Context, modules map[string]*module.Module, stage Stage, mID string) error
	GetPending(ctx context.Context, mID string) (Stage, map[string]struct{}, map[string]struct{}, map[string]struct{}, error)
	CancelPending(ctx context.Context, mID string) error
}

type DeploymentHandler interface {
	List(ctx context.Context, filter model.DepFilter, dependencyInfo, assets, containers, containerInfo bool) (map[string]model.Deployment, error)
	Get(ctx context.Context, dID string, dependencyInfo, assets, containers, containerInfo bool) (model.Deployment, error)
	Create(ctx context.Context, mod *module.Module, depReq model.DepInput, incl dir_fs.DirFS, indirect bool) (string, error)
	Delete(ctx context.Context, dID string, force bool) error
	DeleteAll(ctx context.Context, filter model.DepFilter, force bool) error
	Update(ctx context.Context, dID string, mod *module.Module, depReq model.DepInput, incl dir_fs.DirFS) error
	Start(ctx context.Context, dID string, dependencies bool) error
	StartAll(ctx context.Context, filter model.DepFilter, dependencies bool) error
	Stop(ctx context.Context, dID string, force bool) error
	StopAll(ctx context.Context, filter model.DepFilter, force bool) error
	Restart(ctx context.Context, id string) error
	RestartAll(ctx context.Context, filter model.DepFilter) error
}

type DepStorageHandler interface {
	BeginTransaction(ctx context.Context) (driver.Tx, error)
	ListDep(ctx context.Context, filter model.DepFilter, dependencyInfo, assets, containers bool) (map[string]model.Deployment, error)
	ReadDep(ctx context.Context, dID string, dependencyInfo, assets, containers bool) (model.Deployment, error)
	CreateDep(ctx context.Context, tx driver.Tx, depBase model.DepBase) (string, error)
	UpdateDep(ctx context.Context, tx driver.Tx, depBase model.DepBase) error
	DeleteDep(ctx context.Context, tx driver.Tx, dID string) error
	ReadDepTree(ctx context.Context, dID string, assets, containers bool) (map[string]model.Deployment, error)
	AppendDepTree(ctx context.Context, tree map[string]model.Deployment, assets, containers bool) error
	CreateDepAssets(ctx context.Context, tx driver.Tx, dID string, depAssets model.DepAssets) error
	DeleteDepAssets(ctx context.Context, tx driver.Tx, dID string) error
	CreateDepDependencies(ctx context.Context, tx driver.Tx, dID string, dIDs []string) error
	DeleteDepDependencies(ctx context.Context, tx driver.Tx, dID string) error
	CreateDepContainers(ctx context.Context, tx driver.Tx, dID string, depContainers map[string]model.DepContainer) error
	DeleteDepContainers(ctx context.Context, tx driver.Tx, dID string) error
}

//type AuxDeploymentHandler interface {
//	List(ctx context.Context, dID string, filter model.AuxDepFilter, ctrInfo bool) ([]model.AuxDeployment, error)
//	Get(ctx context.Context, aID string, ctrInfo bool) (model.AuxDeployment, error)
//	Create(ctx context.Context, mod *module.Module, dep model.Deployment, auxReq model.AuxDepReq) (string, error)
//	Update(ctx context.Context, aID string, mod *module.Module, auxReq model.AuxDepReq) error
//	Delete(ctx context.Context, aID string) error
//	DeleteAll(ctx context.Context, dID string, filter model.AuxDepFilter) error
//	Start(ctx context.Context, aID string) error
//	StartAll(ctx context.Context, dID string, filter model.AuxDepFilter) error
//	Stop(ctx context.Context, aID string) error
//	StopAll(ctx context.Context, dID string, filter model.AuxDepFilter) error
//}
//
//type AuxDepStorageHandler interface {
//	BeginTransaction(ctx context.Context) (driver.Tx, error)
//	ListAuxDep(ctx context.Context, dID string, filter model.AuxDepFilter) ([]model.AuxDeployment, error)
//	CreateAuxDep(ctx context.Context, tx driver.Tx, auxDep model.AuxDepBase) (string, error)
//	CreateAuxDepCtr(ctx context.Context, tx driver.Tx, aID string, ctr model.AuxDepContainer) error
//	ReadAuxDep(ctx context.Context, aID string) (model.AuxDeployment, error)
//	UpdateAuxDep(ctx context.Context, tx driver.Tx, aID string, auxDep model.AuxDepBase) error
//	DeleteAuxDep(ctx context.Context, aID string) error
//	DeleteAuxDepCtr(ctx context.Context, tx driver.Tx, aID string) error
//}

type CfgValidationHandler interface {
	ValidateBase(cType string, cTypeOpt module.ConfigTypeOptions, dataType module.DataType) error
	ValidateTypeOptions(cType string, cTypeOpt module.ConfigTypeOptions) error
	ValidateValue(cType string, cTypeOpt module.ConfigTypeOptions, value any, isSlice bool, dataType module.DataType) error
	ValidateValInOpt(cOpt any, value any, isSlice bool, dataType module.DataType) error
}

type Validator func(params map[string]any) error

type StageItem interface {
	Module() *module.Module
	ModFile() string
	Dir() dir_fs.DirFS
	Indirect() bool
}

type Stage interface {
	Items() map[string]StageItem
	Get(mID string) (StageItem, bool)
	Remove() error
}

type ModRepo interface {
	Versions() []string
	Get(ver string) (dir_fs.DirFS, error)
	Remove() error
}
