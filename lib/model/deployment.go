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

package model

import (
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"time"
)

type DepMeta struct {
	ID       string    `json:"id"`
	ModuleID string    `json:"module_id"`
	Name     string    `json:"name"`
	Stopped  bool      `json:"stopped"`
	Indirect bool      `json:"indirect"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

type Deployment struct {
	DepMeta
	HostResources map[string]string    `json:"host_resources"` // {ref:resourceID}
	Secrets       map[string]string    `json:"secrets"`        // {ref:secretID}
	Configs       map[string]DepConfig `json:"configs"`        // {ref:value}
	RequiredDep   []string             `json:"required_dep"`   // deployments required by this deployment
	DepRequiring  []string             `json:"dep_requiring"`  // deployments requiring this deployment
}

type DepConfig struct {
	Value    any             `json:"value"`
	DataType module.DataType `json:"data_type"`
	IsSlice  bool            `json:"is_slice"`
}

type DepInstanceMeta struct {
	ID      string    `json:"id"`
	DepID   string    `json:"dep_id"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

type DepInstance struct {
	DepInstanceMeta
	Containers map[string]string `json:"containers"`
}

type DepRequestBase struct {
	Name           *string           `json:"name"`           // defaults to module name if nil
	HostResources  map[string]string `json:"host_resources"` // {ref:resourceID}
	Secrets        map[string]string `json:"secrets"`        // {ref:secretID}
	Configs        map[string]any    `json:"configs"`        // {ref:value}
	SecretRequests map[string]any    // {ref:value}
}

type DepRequest struct {
	ModuleID string `json:"module_id"`
	DepRequestBase
	Dependencies map[string]DepRequestBase `json:"dependencies"`
}

type DepCtrlCmd = string

type DepCtrlRequest struct {
	Cmd DepCtrlCmd `json:"cmd"`
}

type DepFilter struct {
	ModuleID string
	Name     string
	Indirect bool
}

type DepInstFilter struct {
	DepID string
}
