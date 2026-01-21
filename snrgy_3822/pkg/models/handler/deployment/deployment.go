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
	Containers    []Container
	HostResources []models_handler_storage.DeploymentHostResource
	Secrets       []models_handler_storage.DeploymentSecret
	Configs       []models_handler_storage.DeploymentConfig
	State         string // health state determined by container states
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
	ModuleId      string            `json:"module_id"`
	Name          string            `json:"name"`           // defaults to module name if empty
	HostResources map[string]string `json:"host_resources"` // {ref:resourceID}
	Secrets       map[string]string `json:"secrets"`        // {ref:secretID}
	Configs       map[string]any    `json:"configs"`        // {ref:value}
	GlobalConfigs map[string]string `json:"global_configs"` // {ref:configID}
}
