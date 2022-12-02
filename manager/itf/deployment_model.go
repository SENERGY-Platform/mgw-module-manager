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

package itf

type DeploymentBase struct {
	Name      *string           `json:"name"` // module name if nil
	ModuleID  string            `json:"module_id"`
	Resources map[string]string `json:"resources"` // {ref:resourceID}
	Secrets   map[string]string `json:"secrets"`   // {ref:secretID}
	Configs   map[string]any    `json:"configs"`   // {ref:value}
}

type Deployment struct {
	ID string `json:"id"`
	DeploymentBase
	Containers Set[string]
}

// --------------------------------------------------

type InputTemplate struct {
	Resources   map[string]InputTemplateResource `json:"resources"`    // {ref:ResourceInput}
	Secrets     map[string]InputTemplateResource `json:"secrets"`      // {ref:SecretInput}
	Configs     map[string]InputTemplateConfig   `json:"configs"`      // {ref:ConfigInput}
	InputGroups map[string]InputGroup            `json:"input_groups"` // {ref:InputGroup}
}

type InputTemplateResource struct {
	Input
	Resource
}

type InputTemplateConfig struct {
	Input
	Default  any            `json:"default"`
	Options  any            `json:"options"`
	OptExt   bool           `json:"opt_ext"`
	Type     string         `json:"type"`
	TypeOpt  map[string]any `json:"type_opt"`
	DataType DataType       `json:"data_type"`
	IsSlice  bool           `json:"is_slice"`
}

// --------------------------------------------------

type DeploymentsPostRequest struct {
	DeploymentBase
	SecretRequests map[string]any // {ref:value}
}
