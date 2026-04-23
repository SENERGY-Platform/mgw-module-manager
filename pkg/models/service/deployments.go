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

package models_service

type UserInput struct {
	ModuleId      string                                   `json:"module_id"`
	HostResources map[string]string                        `json:"host_resources"` // {ref:resourceID}
	Secrets       map[string]string                        `json:"secrets"`        // {ref:secretID}
	Configs       map[string]any                           `json:"configs"`        // {ref:value}
	GlobalConfigs map[string]string                        `json:"global_configs"` // {ref:configID}
	Files         map[string]string                        `json:"files"`          // {ref:data}
	FileGroups    map[string]map[string]FileGroupUserInput `json:"file_groups"`    // {ref:{path:FileGroupUserInput}}
}

type FileGroupUserInput struct {
	Format int    `json:"format"`
	Data   string `json:"data"`
}
