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
	"io"
)

type ModuleHandler interface {
	List(ctx context.Context) ([]*module.Module, error)
	Get(ctx context.Context, id string) (*module.Module, error)
	Add(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, id string) error
	InputTemplate(ctx context.Context, id string) (model.InputTemplate, error)
}

type ModStorageHandler interface {
	List(ctx context.Context) ([]string, error)
	Open(ctx context.Context, id string) (io.ReadCloser, error)
	Delete(ctx context.Context, id string) error
	CopyTo(ctx context.Context, id string, dstPath string) error
	CopyFrom(ctx context.Context, id string, srcPath string) error
}

type ModTransferHandler interface {
}

type DeploymentHandler interface {
	List(ctx context.Context) ([]model.DepMeta, error)
	Get(ctx context.Context, id string) (*model.Deployment, error)
	Create(ctx context.Context, m *module.Module, name *string, hostRes map[string]string, secrets map[string]string, configs map[string]any) (string, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, m *module.Module, id string, name *string, hostRes map[string]string, secrets map[string]string, configs map[string]any) error
	Deploy(ctx context.Context, m *module.Module, mPath string, d *model.Deployment) error
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string) error
}

type DepStorageHandler interface {
	BeginTransaction(ctx context.Context) (driver.Tx, error)
	ListDep(ctx context.Context, filter model.DepFilter) ([]model.DepMeta, error)
	CreateDep(ctx context.Context, tx driver.Tx, dep *model.Deployment) (string, error)
	ReadDep(ctx context.Context, id string) (*model.Deployment, error)
	UpdateDep(ctx context.Context, tx driver.Tx, dep *model.Deployment) error
	DeleteDep(ctx context.Context, id string) error
	ListInst(ctx context.Context, filter model.DepInstFilter) ([]model.DepInstanceMeta, error)
	CreateInst(ctx context.Context, tx driver.Tx, inst *model.DepInstanceMeta) (string, error)
	ReadInst(ctx context.Context, id string) (*model.DepInstance, error)
	UpdateInst(ctx context.Context, tx driver.Tx, inst *model.DepInstanceMeta) error
	DeleteInst(ctx context.Context, id string) error
	CreateInstCtr(ctx context.Context, tx driver.Tx, iId, sRef string) (string, error)
	DeleteInstCtr(ctx context.Context, cId string) error
}

type Validator func(params map[string]any) error

type CfgValidationHandler interface {
	ValidateBase(cType string, cTypeOpt module.ConfigTypeOptions, dataType module.DataType) error
	ValidateTypeOptions(cType string, cTypeOpt module.ConfigTypeOptions) error
	ValidateValue(cType string, cTypeOpt module.ConfigTypeOptions, value any, isSlice bool, dataType module.DataType) error
	ValidateValInOpt(cOpt any, value any, isSlice bool, dataType module.DataType) error
}
