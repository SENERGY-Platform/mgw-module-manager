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
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"io/fs"
)

type ModuleHandler interface {
	List(ctx context.Context, filter lib_model.ModFilter, dependencyInfo bool) (map[string]model.Module, error)
	Get(ctx context.Context, mID string, dependencyInfo bool) (model.Module, error)
	Add(ctx context.Context, mod *module.Module, modDir dir_fs.DirFS, modFile string) error
	Update(ctx context.Context, mod *module.Module, modDir dir_fs.DirFS, modFile string) error
	Delete(ctx context.Context, mID string, force bool) error
	GetTree(ctx context.Context, mID string) (map[string]model.Module, error)
	AppendModTree(ctx context.Context, tree map[string]model.Module) error
}

type ModFileHandler interface {
	GetModule(file fs.File) (*module.Module, error)
	GetModFile(dir dir_fs.DirFS) (fs.File, string, error)
}

type ModStorageHandler interface {
	BeginTransaction(ctx context.Context) (driver.Tx, error)
	ListMod(ctx context.Context, filter model.ModFilter, dependencyInfo bool) (map[string]model.Module, error)
	ReadMod(ctx context.Context, mID string, dependencyInfo bool) (model.Module, error)
	CreateMod(ctx context.Context, tx driver.Tx, mod model.Module) error
	UpdateMod(ctx context.Context, tx driver.Tx, mod model.Module) error
	DeleteMod(ctx context.Context, tx driver.Tx, mID string) error
	CreateModDependencies(ctx context.Context, tx driver.Tx, mID string, mIDs []string) error
	DeleteModDependencies(ctx context.Context, tx driver.Tx, mID string) error
}

type ModTransferHandler interface {
	Get(ctx context.Context, mID string) (ModRepo, error)
}

type ModStagingHandler interface {
	Prepare(ctx context.Context, modules map[string]*module.Module, mID, ver string) (Stage, error)
}

type ModUpdateHandler interface {
	Check(ctx context.Context, modules map[string]*module.Module) error
	List(ctx context.Context) map[string]lib_model.ModUpdate
	Get(ctx context.Context, mID string) (lib_model.ModUpdate, error)
	Remove(ctx context.Context, mID string) error
	Prepare(ctx context.Context, modules map[string]*module.Module, stage Stage, mID string) error
	GetPending(ctx context.Context, mID string) (Stage, map[string]struct{}, map[string]struct{}, map[string]struct{}, error)
	CancelPending(ctx context.Context, mID string) error
}

type DeploymentHandler interface {
	List(ctx context.Context, filter lib_model.DepFilter, dependencyInfo, assets, containers, containerInfo bool) (map[string]lib_model.Deployment, error)
	Get(ctx context.Context, dID string, dependencyInfo, assets, containers, containerInfo bool) (lib_model.Deployment, error)
	Create(ctx context.Context, mod *module.Module, depReq lib_model.DepInput, incl dir_fs.DirFS, indirect bool) (string, error)
	Delete(ctx context.Context, dID string, force bool) error
	DeleteAll(ctx context.Context, filter lib_model.DepFilter, force bool) error
	Update(ctx context.Context, dID string, mod *module.Module, depReq lib_model.DepInput, incl dir_fs.DirFS) error
	Start(ctx context.Context, dID string, dependencies bool) error
	StartAll(ctx context.Context, filter lib_model.DepFilter, dependencies bool) error
	Stop(ctx context.Context, dID string, force bool) error
	StopAll(ctx context.Context, filter lib_model.DepFilter, force bool) error
	Restart(ctx context.Context, id string) error
	RestartAll(ctx context.Context, filter lib_model.DepFilter) error
}

type DepStorageHandler interface {
	BeginTransaction(ctx context.Context) (driver.Tx, error)
	ListDep(ctx context.Context, filter lib_model.DepFilter, dependencyInfo, assets, containers bool) (map[string]lib_model.Deployment, error)
	ReadDep(ctx context.Context, dID string, dependencyInfo, assets, containers bool) (lib_model.Deployment, error)
	CreateDep(ctx context.Context, tx driver.Tx, depBase lib_model.DepBase) (string, error)
	UpdateDep(ctx context.Context, tx driver.Tx, depBase lib_model.DepBase) error
	DeleteDep(ctx context.Context, tx driver.Tx, dID string) error
	ReadDepTree(ctx context.Context, dID string, assets, containers bool) (map[string]lib_model.Deployment, error)
	AppendDepTree(ctx context.Context, tree map[string]lib_model.Deployment, assets, containers bool) error
	CreateDepAssets(ctx context.Context, tx driver.Tx, dID string, depAssets lib_model.DepAssets) error
	DeleteDepAssets(ctx context.Context, tx driver.Tx, dID string) error
	CreateDepDependencies(ctx context.Context, tx driver.Tx, dID string, dIDs []string) error
	DeleteDepDependencies(ctx context.Context, tx driver.Tx, dID string) error
	CreateDepContainers(ctx context.Context, tx driver.Tx, dID string, depContainers map[string]lib_model.DepContainer) error
	DeleteDepContainers(ctx context.Context, tx driver.Tx, dID string) error
}

type AuxDeploymentHandler interface {
	List(ctx context.Context, dID string, filter lib_model.AuxDepFilter, assets, containerInfo bool) (map[string]lib_model.AuxDeployment, error)
	Get(ctx context.Context, aID string, assets, containerInfo bool) (lib_model.AuxDeployment, error)
	Create(ctx context.Context, mod model.Module, dep lib_model.Deployment, requiredDep map[string]lib_model.Deployment, auxReq lib_model.AuxDepReq, updateImg bool) (string, error)
	Update(ctx context.Context, aID string, mod model.Module, dep lib_model.Deployment, requiredDep map[string]lib_model.Deployment, auxReq lib_model.AuxDepReq, updateImg bool) error
	Delete(ctx context.Context, aID string, force bool) error
	DeleteAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter, force bool) error
	Start(ctx context.Context, aID string) error
	StartAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter) error
	Stop(ctx context.Context, aID string) error
	StopAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter) error
	Restart(ctx context.Context, aID string) error
	RestartAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter) error
}

type AuxDepStorageHandler interface {
	BeginTransaction(ctx context.Context) (driver.Tx, error)
	ListAuxDep(ctx context.Context, dID string, filter lib_model.AuxDepFilter, assets bool) (map[string]lib_model.AuxDeployment, error)
	ReadAuxDep(ctx context.Context, aID string, assets bool) (lib_model.AuxDeployment, error)
	CreateAuxDep(ctx context.Context, tx driver.Tx, auxDep lib_model.AuxDepBase) (string, error)
	UpdateAuxDep(ctx context.Context, tx driver.Tx, auxDep lib_model.AuxDepBase) error
	DeleteAuxDep(ctx context.Context, tx driver.Tx, aID string) error
	CreateAuxDepContainer(ctx context.Context, tx driver.Tx, aID string, auxDepContainer lib_model.AuxDepContainer) error
	DeleteAuxDepContainer(ctx context.Context, tx driver.Tx, aID string) error
}

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
