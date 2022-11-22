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

package deployment

import (
	"module-manager/manager/itf/module"
)

type Base struct {
	Name          *string           `json:"name"` // module name if nil
	ModuleID      string            `json:"module_id"`
	ModuleVersion string            `json:"module_version"`
	Resources     map[string]string `json:"resources"` // {ref:resourceID}
	Secrets       map[string]string `json:"secrets"`   // {ref:secretID}
	Configs       map[string]any    `json:"configs"`   // {ref:value}
}

type Deployment struct {
	ID string `json:"id"`
	Base
}

// --------------------------------------------------

type UserInput struct {
	ResourceInputs map[string]ResourceInput     `json:"resource_inputs"` // {ref:ResourceInput}
	SecretInputs   map[string]ResourceInput     `json:"secret_inputs"`   // {ref:ResourceInput}
	ConfigInputs   map[string]ConfigInput       `json:"config_inputs"`   // {ref:ConfigInput}
	InputGroups    map[string]module.InputGroup `json:"input_groups"`    // {ref:InputGroup}
}

type ResourceInput struct {
	module.Input
	OptionsSrc string `json:"options_src"`
}

type ConfigInput struct {
	module.Input
	Options []any `json:"options"`
}
