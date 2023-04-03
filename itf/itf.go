/*
 * Copyright 2022 InfAI (CC SES)
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

package itf

import (
	"context"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"io"
	"module-manager/model"
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
	Update(ctx context.Context, m *module.Module, name *string, hostRes map[string]string, secrets map[string]string, configs map[string]any) error
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string) error
}

type DepStorageHandler interface {
	List(ctx context.Context) ([]model.DepMeta, error)
	Create(ctx context.Context, dep *model.Deployment) (Transaction, string, error)
	Read(ctx context.Context, id string) (*model.Deployment, error)
	Update(ctx context.Context, dep *model.Deployment) (Transaction, error)
	Delete(ctx context.Context, id string) error
}

type Transaction interface {
	Commit() error
	Rollback() error
}

type Validator func(params map[string]any) error

type CfgValidationHandler interface {
	ValidateBase(cType string, cTypeOpt module.ConfigTypeOptions, dataType module.DataType) error
	ValidateTypeOptions(cType string, cTypeOpt module.ConfigTypeOptions) error
	ValidateValue(cType string, cTypeOpt module.ConfigTypeOptions, value any, isSlice bool, dataType module.DataType) error
	ValidateValInOpt(cOpt any, value any, isSlice bool, dataType module.DataType) error
}

type Api interface {
	AddModule(ctx context.Context, id string) error
	GetModules(ctx context.Context) ([]*module.Module, error)
	GetModule(ctx context.Context, id string) (*module.Module, error)
	DeleteModule(ctx context.Context, id string) error
	GetInputTemplate(ctx context.Context, id string) (model.InputTemplate, error)
	AddDeployment(ctx context.Context, dr model.DepRequest) (string, error)
	GetDeployments(ctx context.Context) ([]model.DepMeta, error)
	GetDeployment(ctx context.Context, id string) (*model.Deployment, error)
	StartDeployment(ctx context.Context, id string) error
	StopDeployment(ctx context.Context, id string) error
	UpdateDeployment(ctx context.Context, dr model.DepRequest)
	DeleteDeployment(ctx context.Context, id string) error
}
