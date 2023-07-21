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
	Dir      string    `json:"dir"`
	Enabled  bool      `json:"enabled"`
	Indirect bool      `json:"indirect"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

type DepAssets struct {
	HostResources map[string]string    `json:"host_resources"` // {ref:resourceID}
	Secrets       map[string]DepSecret `json:"secrets"`        // {ref:DepSecret}
	Configs       map[string]DepConfig `json:"configs"`        // {ref:DepConfig}
	RequiredDep   []string             `json:"required_dep"`   // deployments required by this deployment
	DepRequiring  []string             `json:"dep_requiring"`  // deployments requiring this deployment
}

type Deployment struct {
	DepMeta
	DepAssets
}

type DepSecret struct {
	ID       string             `json:"id"`
	Variants []DepSecretVariant `json:"variants"`
}

type DepSecretVariant struct {
	Item    *string `json:"item"`
	AsMount bool    `json:"as_mount"`
	AsEnv   bool    `json:"as_env"`
}

type DepConfig struct {
	Value    any             `json:"value"`
	DataType module.DataType `json:"data_type"`
	IsSlice  bool            `json:"is_slice"`
}

type DepInstance struct {
	ID         string      `json:"id"`
	Created    time.Time   `json:"created"`
	Containers []Container `json:"containers"`
}

type Instance struct {
	ID      string    `json:"id"`
	DepID   string    `json:"dep_id"`
	Created time.Time `json:"created"`
}

type Container struct {
	ID    string `json:"id"`
	Ref   string `json:"ref"`
	Order uint   `json:"order"`
}

type SortDirection = int

type CtrFilter struct {
	SortOrder SortDirection
}

type DepInput struct {
	Name           *string           `json:"name"`           // defaults to module name if nil
	HostResources  map[string]string `json:"host_resources"` // {ref:resourceID}
	Secrets        map[string]string `json:"secrets"`        // {ref:secretID}
	Configs        map[string]any    `json:"configs"`        // {ref:value}
	SecretRequests map[string]any    // {ref:value}
}

type DepCreateRequest struct {
	ModuleID string `json:"module_id"`
	DepInput
	Dependencies map[string]DepInput `json:"dependencies"`
}

type DepFilter struct {
	ModuleID string
	Name     string
	Indirect bool
}

type DepInstFilter struct {
	DepID string
}

type DepUpdateTemplate struct {
	Name string `json:"name"`
	InputTemplate
}

type HealthStatus = string

type DepHealthInfo struct {
	Status     HealthStatus    `json:"status"`
	Containers []CtrHealthInfo `json:"containers"`
}

type CtrHealthInfo struct {
	ID    string `json:"id"`
	Ref   string `json:"ref"`
	State string `json:"state"`
}
