/*
 * Copyright 2025 InfAI (CC SES)
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
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

const (
	StateHealthy      = "healthy"
	StateUnhealthy    = "unhealthy"
	StateNotAvailable = "not-available"
)

type Deployment struct {
	models_handler_storage.Deployment
	Containers    map[string]Container
	Volumes       map[string]models_handler_storage.DeploymentVolume
	HostResources map[string]models_handler_storage.DeploymentHostResource
	Secrets       map[string]models_handler_storage.DeploymentSecret
	Configs       map[string]models_handler_storage.DeploymentUserConfig
	GlobalConfigs map[string]models_handler_storage.DeploymentGlobalConfig
	Files         map[string]models_handler_storage.DeploymentFile
	FileGroups    map[string]models_handler_storage.DeploymentFileGroup
	State         string // health state determined by container states
}

type DeploymentReduced struct {
	models_handler_storage.Deployment
	Containers map[string]Container
	State      string // health state determined by container states
}

type Container struct {
	models_handler_storage.DeploymentContainer
	ImageId string // docker image id
	State   string // docker container state
}

type DeploymentsFilter struct {
	models_handler_storage.DeploymentsFilter
	State string
}

type UserInput struct {
	ModuleId      string
	Name          string                                   // defaults to module name if empty
	HostResources map[string]string                        // {ref:resourceID}
	Secrets       map[string]string                        // {ref:secretID}
	Configs       map[string]any                           // {ref:value}
	GlobalConfigs map[string]string                        // {ref:configID}
	Files         map[string][]byte                        // {ref:data}
	FileGroups    map[string]map[string]FileGroupUserInput // {ref:{path:FileGroupUserInput}}
}

type FileGroupUserInput struct {
	Format int
	Data   []byte
}
