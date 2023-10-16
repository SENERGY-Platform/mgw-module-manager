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
	"time"
)

type ModuleHandler interface {
	List(ctx context.Context, filter model.ModFilter) ([]model.Module, error)
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
	List(ctx context.Context, filter model.ModFilter) ([]model.Module, error)
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
	List(ctx context.Context, filter model.DepFilter) ([]model.DepBase, error)
	Get(ctx context.Context, dID string, assets, instance bool) (model.Deployment, error)
	Create(ctx context.Context, mod *module.Module, depReq model.DepInput, incl dir_fs.DirFS, indirect bool) (string, error)
	Delete(ctx context.Context, dID string, orphans bool) error
	Update(ctx context.Context, dID string, mod *module.Module, depReq model.DepInput, incl dir_fs.DirFS) error
	Enable(ctx context.Context, dID string, dependencies bool) error
	Disable(ctx context.Context, dID string, dependencies bool) error
	Start(ctx context.Context, dID string) error
	Stop(ctx context.Context, dID string) error
}

type DepStorageHandler interface {
	BeginTransaction(ctx context.Context) (driver.Tx, error)
	ListDep(ctx context.Context, filter model.DepFilter) ([]model.DepBase, error)
	CreateDep(ctx context.Context, tx driver.Tx, depBase model.DepBase) (string, error)
	CreateDepAssets(ctx context.Context, tx driver.Tx, dID string, depAssets model.DepAssets) error
	ReadDep(ctx context.Context, dID string, assets bool) (model.Deployment, error)
	UpdateDep(ctx context.Context, tx driver.Tx, depBase model.DepBase) error
	DeleteDep(ctx context.Context, dID string) error
	DeleteDepAssets(ctx context.Context, itf driver.Tx, dID string) error
	ListInst(ctx context.Context, filter model.DepInstFilter) ([]model.Instance, error)
	CreateInst(ctx context.Context, tx driver.Tx, dID string, timestamp time.Time) (string, error)
	ReadInst(ctx context.Context, iID string) (model.Instance, error)
	DeleteInst(ctx context.Context, iID string) error
	ListInstCtr(ctx context.Context, iID string, filter model.CtrFilter) ([]model.Container, error)
	CreateInstCtr(ctx context.Context, tx driver.Tx, iID string, ctr model.Container) error
}

type DepHealthHandler interface {
	List(ctx context.Context, instances map[string]model.DepInstance) (map[string]model.DepHealthInfo, error)
	Get(ctx context.Context, instance model.DepInstance) (model.DepHealthInfo, error)
}

type SubDeploymentHandler interface {
	List(ctx context.Context, filter model.SubDepFilter, ctrInfo bool) ([]model.SubDeployment, error)
	Get(ctx context.Context, id string, ctrInfo bool) (model.SubDeployment, error)
	Create(ctx context.Context, sdReq model.SubDepBase) (string, error)
	Update(ctx context.Context, id string, sdReq model.SubDepBase) error
	Delete(ctx context.Context, id string) error
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string) error
}

type CfgValidationHandler interface {
	ValidateBase(cType string, cTypeOpt module.ConfigTypeOptions, dataType module.DataType) error
	ValidateTypeOptions(cType string, cTypeOpt module.ConfigTypeOptions) error
	ValidateValue(cType string, cTypeOpt module.ConfigTypeOptions, value any, isSlice bool, dataType module.DataType) error
	ValidateValInOpt(cOpt any, value any, isSlice bool, dataType module.DataType) error
}

type JobHandler interface {
	List(filter model.JobFilter) []model.Job
	Get(id string) (model.Job, error)
	Create(desc string, tFunc func(context.Context, context.CancelFunc) error) (string, error)
	Cancel(id string) error
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
