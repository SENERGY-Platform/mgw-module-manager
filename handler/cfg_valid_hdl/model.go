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

package cfg_valid_hdl

import (
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-lib/util"
)

type ConfigDefinition struct {
	DataType   util.Set[module.DataType]         `json:"data_type"`
	Options    map[string]ConfigDefinitionOption `json:"options"`
	Validators []ConfigDefinitionValidator       `json:"validators"`
}

type ConfigDefinitionOption struct {
	DataType util.Set[module.DataType] `json:"data_type"`
	Inherit  bool                      `json:"inherit"`
	Required bool                      `json:"required"`
}

type ConfigDefinitionValidator struct {
	Name      string                                    `json:"name"`
	Parameter map[string]ConfigDefinitionValidatorParam `json:"parameter"`
}

type ConfigDefinitionValidatorParam struct {
	Value any     `json:"value"`
	Ref   *string `json:"ref"`
}
