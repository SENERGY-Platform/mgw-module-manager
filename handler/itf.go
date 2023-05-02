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
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"time"
)

type ModuleHandler interface {
	List(ctx context.Context, filter model.ModFilter) ([]model.ModuleMeta, error)
	Get(ctx context.Context, mID string) (*module.Module, error)
	GetWithDep(ctx context.Context, mID string) (*module.Module, map[string]*module.Module, error)
	Add(ctx context.Context, mr model.ModRequest) error
	Delete(ctx context.Context, mID string) error
	Update(ctx context.Context, mID string) error
	CreateInclDir(ctx context.Context, mID, iID string) (string, error)
	DeleteInclDir(ctx context.Context, iID string) error
}

type ModFileHandler interface {
	GetModule(dir util.DirFS) (*module.Module, error)
}

type ModStorageHandler interface {
	List(ctx context.Context, filter model.ModFilter) ([]model.ModuleMeta, error)
	Get(ctx context.Context, mID string) (*module.Module, error)
	Add(ctx context.Context, dir util.DirFS, mID string) error
	Delete(ctx context.Context, mID string) error
	GetInclDir(ctx context.Context, iID string) (util.DirFS, error)
	MakeInclDir(ctx context.Context, mID, iID string) (util.DirFS, error)
	RemoveInclDir(ctx context.Context, iID string) error
}

type ModTransferHandler interface {
	ListVersions(ctx context.Context, mID string) ([]string, error)
	Get(ctx context.Context, mID, ver string) (util.DirFS, error)
}

type DeploymentHandler interface {
	List(ctx context.Context, filter model.DepFilter) ([]model.DepMeta, error)
	Get(ctx context.Context, dID string) (*model.Deployment, error)
	GetTemplate(ctx context.Context, mID string) (*model.DepTemplate, error)
	Create(ctx context.Context, dr model.DepRequest) (string, error)
	Delete(ctx context.Context, dID string, orphans bool) error
	Update(ctx context.Context, dID string, drb model.DepRequestBase) error
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
	ListInst(ctx context.Context, filter model.DepInstFilter) ([]model.DepInstanceMeta, error)
	CreateInst(ctx context.Context, tx driver.Tx, dID string, timestamp time.Time) (string, error)
	ReadInst(ctx context.Context, iID string) (*model.DepInstance, error)
	UpdateInst(ctx context.Context, iID string, timestamp time.Time) error
	DeleteInst(ctx context.Context, iID string) error
	CreateInstCtr(ctx context.Context, tx driver.Tx, iID, cID, sRef string) error
	DeleteInstCtr(ctx context.Context, cID string) error
}

type Validator func(params map[string]any) error

type CfgValidationHandler interface {
	ValidateBase(cType string, cTypeOpt module.ConfigTypeOptions, dataType module.DataType) error
	ValidateTypeOptions(cType string, cTypeOpt module.ConfigTypeOptions) error
	ValidateValue(cType string, cTypeOpt module.ConfigTypeOptions, value any, isSlice bool, dataType module.DataType) error
	ValidateValInOpt(cOpt any, value any, isSlice bool, dataType module.DataType) error
}
