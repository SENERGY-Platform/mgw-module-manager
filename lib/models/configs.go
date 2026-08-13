/*
 * Copyright 2026 InfAI (CC SES)
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

package models

// GlobalConfigInput is the user input for creating or updating a global config.
// Global configs hold values that can be referenced by the configs of multiple deployments instead of setting the value per deployment.
type GlobalConfigInput struct {
	Name           string `json:"name"` // human readable name of the global config, does not have to be unique
	InterfaceValue        // value of the global config together with its type information
}

// GlobalConfig is a stored config value that deployments can reference via DeploymentUserInput.GlobalConfigs.
type GlobalConfig struct {
	Id             string `json:"id"`   // unique ID, used to reference the global config in deployment user inputs
	Name           string `json:"name"` // human readable name of the global config, does not have to be unique
	InterfaceValue        // value of the global config together with its type information
}

// InterfaceValue is a typed config value.
// DataType and IsSlice describe how Value is to be interpreted, they must match the type declared by the module config the value is used for.
type InterfaceValue struct {
	DataType int         `json:"data_type"` // type of Value, values: 1 = string, 2 = int64, 3 = float64, 4 = bool
	IsSlice  bool        `json:"is_slice"`  // true if Value is a list of DataType instead of a single item
	Value    interface{} `json:"value"`     // the value itself, a single item or a list of items as stated by DataType and IsSlice
}
