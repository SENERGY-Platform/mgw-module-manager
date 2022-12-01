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
	"io"
	"module-manager/manager/itf/deployment"
	"module-manager/manager/itf/module"
)

type ModuleHandler interface {
	List() ([]module.Module, error)
	Read(id string) (module.Module, error)
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

type DeploymentHandler interface {
	List() ([]deployment.Deployment, error)
	Read(id string) (deployment.Deployment, error)
	Add(b deployment.Base, m module.Module) error
	Start(id string) error
	Stop(id string) error
	Delete(id string) error
	Update(id string) error
	InputTemplate(m module.Module) deployment.InputTemplate
}

type DeploymentStorageHandler interface {
	List() ([]deployment.Deployment, error)
	Create(base deployment.Deployment) error
	Read(id string) (deployment.Deployment, error)
	Update(id string) error
	Delete(id string) error
}

type ModFileModule interface {
	Parse(confDefHandler ConfDefHandler) (module.Module, error)
}

type ConfDefHandler interface {
	Parse(cType string, cTypeOpt map[string]any, dType module.DataType) (map[string]any, error)
}
