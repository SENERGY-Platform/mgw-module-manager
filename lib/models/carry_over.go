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

// DeploymentCarryOver reports whether the setup of the deployment of a module can be reused for the module without user input.
type DeploymentCarryOver struct {
	ModuleId    string               `json:"module_id"` // ID of the module the check was performed for
	Input       *DeploymentUserInput `json:"input"`     // user input built from the current setup that can be passed to the deployment update as is, null if the setup cannot be carried over
	Missing     []CarryOverMissing   `json:"missing"`   // items the current setup does not answer, empty if Input is set
	ErrorResult                      // set if the module or its deployment could not be read, Input is then null and Missing empty
}

// CarryOverMissing names a module item the current setup does not answer and that therefore needs user input.
type CarryOverMissing struct {
	Kind   string `json:"kind"`   // kind of the item, values: config, secret, host_resource, file, file_group
	Ref    string `json:"ref"`    // reference of the item in the module
	Reason string `json:"reason"` // why the setup does not answer the item, values: not_declared, not_set, empty_value, type_mismatch, invalid_value, missing_path, global_config_not_found, secret_not_found, secret_type_unknown
}
