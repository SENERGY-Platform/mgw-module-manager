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

type DeploymentBase struct {
	ModuleID  string            `json:"module_id"`
	Name      *string           `json:"name"`      // defaults to module name if nil
	Resources map[string]string `json:"resources"` // {ref:resourceID}
	Secrets   map[string]string `json:"secrets"`   // {ref:secretID}
	Configs   map[string]any    `json:"configs"`   // {ref:value}
}

type Deployment struct {
	ID string `json:"id"`
	DeploymentBase
	Containers map[string]struct{}
}

type DeploymentRequest struct {
	DeploymentBase
	SecretRequests map[string]any // {ref:value}
}
