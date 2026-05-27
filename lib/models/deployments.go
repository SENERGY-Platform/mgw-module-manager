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

import (
	"time"

	"github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
)

type Deployment struct {
	Id            string                         `json:"id"`
	ModuleSource  string                         `json:"module_source"`
	ModuleChannel string                         `json:"module_channel"`
	ModuleVersion string                         `json:"module_version"`
	Enabled       bool                           `json:"enabled"`
	Created       time.Time                      `json:"created"`
	Updated       time.Time                      `json:"updated"`
	Containers    map[string]Container           `json:"containers"`
	Volumes       map[string]string              `json:"volumes"`        // {reference:name}
	HostResources map[string]string              `json:"host_resources"` // {reference:hostResourceId}
	Secrets       map[string]DeploymentSecret    `json:"secrets"`
	Configs       map[string]InterfaceValue      `json:"configs"`
	GlobalConfigs map[string]string              `json:"global_configs"` // {reference:globalConfigId}
	Files         map[string]string              `json:"files"`          // {reference:data}
	FileGroups    map[string]DeploymentFileGroup `json:"file_groups"`
	State         int                            `json:"state"` // health state determined by container states
}

type Container struct {
	Name    string                    `json:"name"`
	Alias   string                    `json:"alias"`
	ImageId string                    `json:"image_id"` // docker image id
	State   constants.ContainerState  `json:"state"`    // docker container state
	Health  constants.ContainerHealth `json:"health"`   // docker container health
}

type DeploymentSecret struct {
	Id    string `json:"id"`
	Items []DeploymentSecretItem
}

type DeploymentSecretItem struct {
	Name    string `json:"name"`
	AsMount bool   `json:"as_mount"`
	AsEnv   bool   `json:"as_env"`
}

type DeploymentFileGroup struct {
	Id    string                    `json:"id"`
	Files []DeploymentFileGroupFile `json:"files"`
}

type DeploymentFileGroupFile struct {
	Path   string `json:"path"`
	Format int    `json:"format"`
	Data   string `json:"data"`
}

type DeploymentUserInput struct {
	ModuleId      string                                             `json:"module_id"`
	HostResources map[string]string                                  `json:"host_resources"` // {ref:resourceID}
	Secrets       map[string]string                                  `json:"secrets"`        // {ref:secretID}
	Configs       map[string]interface{}                             `json:"configs"`        // {ref:value}
	GlobalConfigs map[string]string                                  `json:"global_configs"` // {ref:configID}
	Files         map[string]string                                  `json:"files"`          // {ref:data}
	FileGroups    map[string]map[string]DeploymentFileGroupUserInput `json:"file_groups"`    // {ref:{path:FileGroupUserInput}}
}

type DeploymentFileGroupUserInput struct {
	Format int    `json:"format"`
	Data   string `json:"data"`
}

type DeploymentJobResult struct {
	JobResult
	Results       []DeploymentResult `json:"results"`
	ResultsErrNum int                `json:"results_err_num"`
}

type DeploymentUpdateJobResult struct {
	JobResult
	Results       []DeploymentUpdateResult `json:"results"`
	ResultsErrNum int                      `json:"results_err_num"`
}

type DeploymentUpdateResult struct {
	DeploymentResult
	AuxiliaryDeployments AuxiliaryDeploymentRecreateResult `json:"auxiliary_deployments"`
}

type DeploymentDeleteResult struct {
	DeploymentResult
	AuxiliaryDeployments AuxiliaryDeploymentDeleteResult `json:"auxiliary_deployments"`
}

type DeploymentResult struct {
	ModuleId string `json:"module_id"`
	Id       string `json:"id"`
	ErrorResult
}
