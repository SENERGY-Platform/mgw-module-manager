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
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"io"
	"module-manager/model"
)

type ModuleHandler interface {
	List() ([]*module.Module, error)
	Read(id string) (*module.Module, error)
	Add(id string) error
	Delete(id string) error
	Update(id string) error
}

type ModuleStorageHandler interface {
	List() ([]string, error)
	Open(id string) (io.ReadCloser, error)
	Delete(id string) error
	CopyTo(id string, dstPath string) error
	CopyFrom(id string, srcPath string) error
}

type ModuleTransferHandler interface {
}

type DeploymentHandler interface {
	List() ([]model.Deployment, error)
	Read(id string) (model.Deployment, error)
	Add(b model.DeploymentBase, m *module.Module) (string, error)
	Start(id string) error
	Stop(id string) error
	Delete(id string) error
	Update(id string) error
	InputTemplate(m *module.Module) model.InputTemplate
}

type DeploymentStorageHandler interface {
	List() ([]model.Deployment, error)
	Create(base model.Deployment) error
	Read(id string) (model.Deployment, error)
	Update(id string) error
	Delete(id string) error
}

type Validator func(params map[string]any) error

type ConfigValidationHandler interface {
	ValidateBase(cType string, cTypeOpt module.ConfigTypeOptions, dataType module.DataType) error
	ValidateOptions(cType string, cTypeOpt module.ConfigTypeOptions) error
	ValidateValue(cType string, cTypeOpt module.ConfigTypeOptions, value any) error
}

type Api interface {
	AddModule(id string) error
	GetModules() ([]*module.Module, error)
	GetModule(id string) (*module.Module, error)
	DeleteModule(id string) error
	GetInputTemplate(id string) (model.InputTemplate, error)
	AddDeployment(dr model.DeploymentRequest) (string, error)
	GetDeployments() ([]model.Deployment, error)
	GetDeployment() (model.Deployment, error)
	StartDeployment(id string) error
	StopDeployment(id string) error
	UpdateDeployment(dr model.DeploymentRequest)
	DeleteDeployment(id string) error
}
