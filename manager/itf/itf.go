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
)

type ModuleHandler interface {
	List() ([]Module, error)
	Read(id string) (Module, error)
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
	List() ([]Deployment, error)
	Read(id string) (Deployment, error)
	Add(b DeploymentBase, m Module) error
	Start(id string) error
	Stop(id string) error
	Delete(id string) error
	Update(id string) error
	InputTemplate(m Module) InputTemplate
}

type DeploymentStorageHandler interface {
	List() ([]Deployment, error)
	Create(base Deployment) error
	Read(id string) (Deployment, error)
	Update(id string) error
	Delete(id string) error
}

type Validator func(params map[string]any) error
