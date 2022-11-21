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
	Name          *string `json:"name"`
	ModuleID      string  `json:"module_id"`
	ModuleVersion string  `json:"module_version"`
}

type Deployment struct {
	ID string `json:"id"`
	Base
	Resources map[string]Resource `json:"resources"`
	Secrets   map[string]Resource `json:"secrets"`
	Configs   map[string]any      `json:"configs"`
}

type Resource struct {
	ID  string `json:"id"`
	Src string `json:"src"`
}

// --------------------------------------------------

type Template struct {
	Base
	ResourceInputs map[string]ResourceInput     `json:"resource_inputs"`
	SecretInputs   map[string]ResourceInput     `json:"secret_inputs"`
	ConfigInputs   map[string]ConfigInput       `json:"config_inputs"`
	InputGroups    map[string]module.InputGroup `json:"input_groups"`
}

type Input struct {
	Value any `json:"value"`
	module.Input
}

type ResourceInput struct {
	Input
	OptionsSrc string `json:"options_src"`
}

type ConfigInput struct {
	Input
	Options []any `json:"options"`
}
