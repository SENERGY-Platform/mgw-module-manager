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
	"module-manager/manager/itf/modfile"
	"module-manager/manager/util"
)

type Base struct {
	Name          *string          `json:"name"`
	ModuleID      modfile.ModuleID `json:"module_id"`
	ModuleVersion util.SemVersion  `json:"module_version"`
}

type Deployment struct {
	ID string `json:"id"`
	Base
	Resources []Resource `json:"resources"`
	Secrets   []Resource `json:"secrets"`
	Configs   []Value    `json:"configs"`
}

type Resource struct {
	Ref string  `json:"ref"`
	ID  *string `json:"id"`
}

type Value struct {
	Ref   string `json:"ref"`
	Value any    `json:"value"`
}

// --------------------------------------------------

type Template struct {
	Base
	Resources []ResourceInput `json:"resources"`
	Secrets   []SecretInput   `json:"secrets"`
	Configs   []ConfigInput   `json:"configs"`
}

type ResourceInput struct {
	Resource
	UserInput  modfile.UserInputBase `json:"user_input"`
	OptionsSrc string                `json:"options_src"`
}

type SecretInput struct {
	Resource
	Value      any               `json:"value"`      // for input if secret does not exist
	UserInput  modfile.UserInput `json:"user_input"` // for input if secret does not exist
	OptionsSrc string            `json:"options_src"`
}

type ConfigInput struct {
	Value
	UserInput modfile.UserInput `json:"user_input"`
	Options   []any             `json:"options"`
}
