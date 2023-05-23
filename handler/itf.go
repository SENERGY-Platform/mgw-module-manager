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
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
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
	GetIncl(ctx context.Context, mID string) (dir_fs.DirFS, error)
	Add(ctx context.Context, mod *module.Module, modDir dir_fs.DirFS, modFile string, indirect bool) error
	Delete(ctx context.Context, mID string) error
	Update(ctx context.Context, mID string) error
}

type ModFileHandler interface {
	GetModule(file fs.File) (*module.Module, error)
	GetModFile(dir dir_fs.DirFS) (fs.File, string, error)
}

type ModStorageHandler interface {
	List(ctx context.Context, filter model.ModFilter) ([]model.Module, error)
	Get(ctx context.Context, mID string) (model.Module, error)
	GetDir(ctx context.Context, mID string) (model.Module, dir_fs.DirFS, error)
	Add(ctx context.Context, mod model.Module, modDir dir_fs.DirFS, modFile string) error
	Delete(ctx context.Context, mID string) error
}

type ModTransferHandler interface {
	ListVersions(ctx context.Context, mID string) ([]string, error)
	Get(ctx context.Context, mID, ver string) (dir_fs.DirFS, error)
}

type ModStagingHandler interface {
	Prepare(ctx context.Context, modules map[string]*module.Module, mID, ver string, updateReq bool) (Stage, error)
}

type DeploymentHandler interface {
	List(ctx context.Context, filter model.DepFilter) ([]model.DepMeta, error)
	Get(ctx context.Context, dID string) (*model.Deployment, error)
	Create(ctx context.Context, mod *module.Module, depReq model.DepRequestBase, incl dir_fs.DirFS, indirect bool) (string, error)
	Delete(ctx context.Context, dID string, orphans bool) error
	Update(ctx context.Context, mod *module.Module, dep *model.Deployment, depReq model.DepRequestBase) error
	Start(ctx context.Context, dID string) error
	Stop(ctx context.Context, dID string, dependencies bool) error
}

type DepStorageHandler interface {
	BeginTransaction(ctx context.Context) (driver.Tx, error)
	ListDep(ctx context.Context, filter model.DepFilter) ([]model.DepMeta, error)
	CreateDep(ctx context.Context, tx driver.Tx, mID, name string, indirect bool, timestamp time.Time) (string, error)
	CreateDepConfigs(ctx context.Context, tx driver.Tx, mConfigs module.Configs, dConfigs map[string]any, dID string) error
	CreateDepHostRes(ctx context.Context, tx driver.Tx, hostResources map[string]string, dID string) error
	CreateDepSecrets(ctx context.Context, tx driver.Tx, secrets map[string]string, dID string) error
	CreateDepReq(ctx context.Context, tx driver.Tx, depReq []string, dID string) error
	ReadDep(ctx context.Context, dID string) (*model.Deployment, error)
	UpdateDep(ctx context.Context, dID, name string, stopped, indirect bool, timestamp time.Time) error
	DeleteDep(ctx context.Context, dID string) error
	DeleteDepConfigs(ctx context.Context, tx driver.Tx, dID string) error
	DeleteDepHostRes(ctx context.Context, tx driver.Tx, dID string) error
	DeleteDepSecrets(ctx context.Context, tx driver.Tx, dID string) error
	ListInst(ctx context.Context, filter model.DepInstFilter) ([]model.Instance, error)
	CreateInst(ctx context.Context, tx driver.Tx, dID string, timestamp time.Time) (string, error)
	ReadInst(ctx context.Context, iID string) (model.Instance, error)
	DeleteInst(ctx context.Context, iID string) error
	ListInstCtr(ctx context.Context, iID string, filter model.CtrFilter) ([]model.Container, error)
	CreateInstCtr(ctx context.Context, tx driver.Tx, iID, cID, sRef string, order uint) error
}

type CewJobHandler interface {
	AwaitJob(ctx context.Context, jID string) (cew_model.Job, error)
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
	Item(mID string) StageItem
	Items() map[string]StageItem
	Remove() error
}
