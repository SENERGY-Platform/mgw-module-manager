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

import (
	"time"

	models_config "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/config"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_database "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

type Module struct {
	models_external.ModuleLibModule
	Source     string     `json:"source"`
	Channel    string     `json:"channel"`
	Added      time.Time  `json:"added"`
	Updated    time.Time  `json:"updated"`
	IsDeployed bool       `json:"is_deployed"`
	Deployment Deployment `json:"deployment"`
}

type Deployment struct {
	Id            string                                  `json:"id"`
	ModuleSource  string                                  `json:"module_source"`
	ModuleChannel string                                  `json:"module_channel"`
	ModuleVersion string                                  `json:"module_version"`
	Enabled       bool                                    `json:"enabled"`
	Created       time.Time                               `json:"created"`
	Updated       time.Time                               `json:"updated"`
	Containers    map[string]Container                    `json:"containers"`
	Volumes       map[string]string                       `json:"volumes"`        // {reference:name}
	HostResources map[string]string                       `json:"host_resources"` // {reference:hostResourceId}
	Secrets       map[string]Secret                       `json:"secrets"`
	Configs       map[string]models_config.InterfaceValue `json:"configs"`
	GlobalConfigs map[string]string                       `json:"global_configs"` // {reference:globalConfigId}
	Files         map[string]string                       `json:"files"`          // {reference:data}
	FileGroups    map[string]FileGroup                    `json:"file_groups"`
	State         int                                     `json:"state"` // health state determined by container states
}

type Container struct {
	Name    string `json:"name"`
	Alias   string `json:"alias"`
	ImageId string `json:"image_id"` // docker image id
	State   string `json:"state"`    // docker container state
	Health  string `json:"health"`   // docker container health
}

type Secret struct {
	Id    string `json:"id"`
	Items []models_handler_database.DeploymentSecretItem
}

type FileGroup struct {
	Id    string          `json:"id"`
	Files []FileGroupFile `json:"files"`
}

type FileGroupFile struct {
	Path   string `json:"path"`
	Format int    `json:"format"`
	Data   string `json:"data"`
}

type ModuleReduced struct {
	Id          string            `json:"id"`
	Source      string            `json:"source"`
	Channel     string            `json:"channel"`
	Version     string            `json:"version"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	License     string            `json:"license"`
	Author      string            `json:"author"`
	IsDeployed  bool              `json:"is_deployed"`
	Deployment  DeploymentReduced `json:"deployment"`
}

type DeploymentReduced struct {
	Id            string    `json:"id"`
	ModuleSource  string    `json:"module_source"`
	ModuleChannel string    `json:"module_channel"`
	ModuleVersion string    `json:"module_version"`
	Enabled       bool      `json:"enabled"`
	Created       time.Time `json:"created"`
	Updated       time.Time `json:"updated"`
	State         int       `json:"state"`
}

type ModulesFilter struct {
	Ids               []string
	Name              string
	Tags              []string
	Author            string
	IsDeployed        int
	DeploymentEnabled int
	DeploymentState   int
}

type ModuleVariant struct {
	Source  string `json:"source"`
	Channel string `json:"channel"`
	Version string `json:"version"`
}

type ChangeRequestItem struct {
	Id      string `json:"id"`
	Source  string `json:"source"`
	Channel string `json:"channel"`
	Remove  bool   `json:"remove"`
	Update  bool   `json:"update"`
}

type ModulesChangeRequest struct {
	Install []ModuleAbbreviated    `json:"install"`
	Change  [][2]ModuleAbbreviated `json:"change"`
	Remove  []string               `json:"remove"`
	Created time.Time              `json:"created"`
}

type ModuleAbbreviated struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Desc string `json:"description"`
	ModuleVariant
}

const (
	ChangeActionInstall = "install"
	ChangeActionChange  = "change"
	ChangeActionRemove  = "remove"
)

type ChangeReportItem struct {
	Id     string `json:"id"`
	Action string `json:"action"`
}

type ChangeReportErrItem struct {
	ChangeReportItem
	Error string `json:"error"`
}

type ModulesChangeReport struct {
	Success []ChangeReportItem    `json:"success"`
	Failed  []ChangeReportErrItem `json:"failed"`
}

type JobResultModulesChange struct {
	JobResult
	ModulesChangeReport
}
