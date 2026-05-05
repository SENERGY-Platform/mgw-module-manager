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

package models_handler_deployments

import (
	models_config "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/config"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_handler_database "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

const (
	StateHealthy = iota + 1
	StateUnhealthy
)

type Deployment struct {
	models_handler_database.Deployment
	Containers    map[string]Container
	Volumes       map[string]models_handler_database.DeploymentVolume
	HostResources map[string]models_handler_database.DeploymentHostResource
	Secrets       map[string]models_handler_database.DeploymentSecret
	Configs       map[string]models_handler_database.DeploymentUserConfig
	GlobalConfigs map[string]models_handler_database.DeploymentGlobalConfig
	Files         map[string]models_handler_database.DeploymentFile
	FileGroups    map[string]models_handler_database.DeploymentFileGroup
	State         int // health state determined by container states
}

type DeploymentReduced struct {
	models_handler_database.Deployment
	Containers map[string]Container
	State      int // health state determined by container states
}

type Container struct {
	models_handler_database.DeploymentContainer
	ImageId string // docker image id
	State   string // docker container state
	Health  string // docker container health
}

type DeploymentsFilter struct {
	models_handler_database.DeploymentsFilter
	State int
}

type UserInput struct {
	ModuleId      string
	HostResources map[string]string                        // {ref:resourceID}
	Secrets       map[string]string                        // {ref:secretID}
	Configs       map[string]models_config.Value           // {ref:Config}
	GlobalConfigs map[string]string                        // {ref:configID}
	Files         map[string][]byte                        // {ref:data}
	FileGroups    map[string]map[string]FileGroupUserInput // {ref:{path:FileGroupUserInput}}
}

type FileGroupUserInput struct {
	Format int
	Data   []byte
}

type Result struct {
	ModuleId string `json:"module_id"`
	Id       string `json:"id"`
	models_error.ErrorResult
}
