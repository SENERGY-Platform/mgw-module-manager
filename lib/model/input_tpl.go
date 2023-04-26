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
)

type DepTemplateBase struct {
	HostResources map[string]DepTemplateHostRes `json:"host_resources"` // {ref:ResourceInput}
	Secrets       map[string]DepTemplateSecret  `json:"secrets"`        // {ref:SecretInput}
	Configs       map[string]DepTemplateConfig  `json:"configs"`        // {ref:ConfigInput}
	InputGroups   map[string]module.InputGroup  `json:"input_groups"`   // {ref:InputGroup}
}

type DepTemplate struct {
	ModuleID string `json:"module_id"`
	DepTemplateBase
	Dependencies map[string]DepTemplateBase `json:"dependencies"`
}

type DepTemplateHostRes struct {
	module.Input
	module.HostResource
}

type DepTemplateSecret struct {
	module.Input
	module.Secret
}

type DepTemplateConfig struct {
	module.Input
	Default  any             `json:"default"`
	Options  any             `json:"options"`
	OptExt   bool            `json:"opt_ext"`
	Type     string          `json:"type"`
	TypeOpt  map[string]any  `json:"type_opt"`
	DataType module.DataType `json:"data_type"`
	IsList   bool            `json:"is_list"`
	Required bool            `json:"required"`
}
