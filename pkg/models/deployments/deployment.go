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

package deployments

import (
	"time"

	lib_models_service "github.com/SENERGY-Platform/mgw-module-manager/lib/models/service"
	models_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/configs"
)

const (
	StateHealthy = iota + 1
	StateUnhealthy
)

type Deployment struct {
	DeploymentBase
	Containers    map[string]Container
	Volumes       map[string]DeploymentVolume
	HostResources map[string]DeploymentHostResource
	Secrets       map[string]DeploymentSecret
	Configs       map[string]DeploymentUserConfig
	GlobalConfigs map[string]DeploymentGlobalConfig
	Files         map[string]DeploymentFile
	FileGroups    map[string]DeploymentFileGroup
	State         int // health state determined by container states
}

type DeploymentReduced struct {
	DeploymentBase
	Containers map[string]Container
	State      int // health state determined by container states
}

type Container struct {
	ContainerBase
	ImageId string // docker image id
	State   string // docker container state
	Health  string // docker container health
}

type DeploymentsFilterWithState struct {
	DeploymentsFilter
	State int
}

type DeploymentBase struct {
	Id            string
	ModuleId      string
	ModuleSource  string
	ModuleChannel string
	ModuleVersion string
	DirName       string
	FilesDirName  string
	Enabled       bool
	Created       time.Time
	Updated       time.Time
}

type ContainerBase struct {
	Name         string
	DeploymentId string
	Reference    string
	Alias        string
}

type DeploymentVolume struct {
	DeploymentId string
	Reference    string
	Name         string
}

type DeploymentHostResource struct {
	Id           string
	DeploymentId string
	Reference    string
}

type DeploymentSecret struct {
	Id           string
	DeploymentId string
	Reference    string
	Items        []lib_models_service.DeploymentSecretItem
}

type DeploymentUserConfig struct {
	Id           string
	DeploymentId string
	Reference    string
	models_configs.Value
}

type DeploymentGlobalConfig struct {
	Id           string
	DeploymentId string
	Reference    string
}

type DeploymentFile struct {
	DeploymentId string
	Reference    string
	Data         []byte
}

type DeploymentFileGroup struct {
	Id           string
	DeploymentId string
	Reference    string
	Files        []DeploymentFileGroupFile
}

type DeploymentFileGroupFile struct {
	Path   string
	Format int
	Data   []byte
}

type DeploymentsFilter struct {
	Ids       []string
	ModuleIds []string
	Enabled   int
}

type DeploymentsHostResourcesFilter struct {
	Ids           []string
	DeploymentIds []string
}

type DeploymentsSecretsFilter struct {
	Ids           []string
	DeploymentIds []string
	AsMount       int
	AsEnv         int
}

type DeploymentGlobalConfigsFilter struct {
	Ids           []string
	DeploymentIds []string
}

type UserInput struct {
	ModuleId      string
	HostResources map[string]string                        // {ref:resourceID}
	Secrets       map[string]string                        // {ref:secretID}
	Configs       map[string]models_configs.Value          // {ref:Config}
	GlobalConfigs map[string]string                        // {ref:configID}
	Files         map[string][]byte                        // {ref:data}
	FileGroups    map[string]map[string]FileGroupUserInput // {ref:{path:FileGroupUserInput}}
}

type FileGroupUserInput struct {
	Format int
	Data   []byte
}
