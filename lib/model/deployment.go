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

type DepBase struct {
	ID       string    `json:"id"`
	Module   DepModule `json:"module"`
	Name     string    `json:"name"`
	Dir      string    `json:"dir"`
	Enabled  bool      `json:"enabled"`
	Indirect bool      `json:"indirect"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

type DepModule struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type DepAssets struct {
	HostResources map[string]string    `json:"host_resources"` // {ref:resourceID}
	Secrets       map[string]DepSecret `json:"secrets"`        // {ref:DepSecret}
	Configs       map[string]DepConfig `json:"configs"`        // {ref:DepConfig}
}

type Deployment struct {
	DepBase
	RequiredDep  []string `json:"required_dep"`  // deployments required by this deployment
	DepRequiring []string `json:"dep_requiring"` // deployments requiring this deployment
	DepAssets
	Containers map[string]DepContainer `json:"containers"` // {ref:DepContainer}
	State      *HealthState            `json:"state"`
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

type DepContainer struct {
	ID     string         `json:"id"`
	SrvRef string         `json:"srv_ref"`
	Alias  string         `json:"alias"`
	Order  uint           `json:"order"`
	Info   *ContainerInfo `json:"info"`
}

type ContainerInfo struct {
	ImageID string `json:"image_id"` // docker image id
	State   string `json:"state"`    // docker container state
}

type HealthState = string

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
	IDs      []string
	ModuleID string
	Name     string
	Enabled  ToggleVal
	Indirect ToggleVal
}

type DepUpdateTemplate struct {
	Name string `json:"name"`
	InputTemplate
}
