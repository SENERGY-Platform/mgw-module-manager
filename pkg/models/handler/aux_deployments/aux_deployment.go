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

package models_handler_aux_deployments

import (
	"time"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

type AuxiliaryDeploymentBase struct {
	Id           string
	DeploymentId string
	Reference    string
	Name         string
	Image        string
	Created      time.Time
	Updated      time.Time
	Enabled      bool
	RunConfig    models_handler_database.AuxiliaryDeploymentRunConfig
}

type AuxiliaryDeployment struct {
	AuxiliaryDeploymentBase
	Labels    map[string]string // {name:value}
	Configs   map[string]string // {varName:value}
	Volumes   []Volume
	Container Container
}

type AuxiliaryDeploymentReduced struct {
	AuxiliaryDeploymentBase
	Container Container
}

type Volume struct {
	Reference string
	MountPath string
}

type Container struct {
	Name    string
	Alias   string
	ImageId string // docker image id
	State   string // docker container state
	Health  string // docker container health
}

type AuxiliaryDeploymentsFilter struct {
	models_handler_database.AuxiliaryDeploymentsFilter
	State string // docker container state
}

type ServiceInput struct {
	Reference string
	Name      string
	Image     string
	PullImage bool
	Labels    map[string]string // {name:value}
	Configs   map[string]string // {varName:value}
	Volumes   map[string]string // {mntPath:volumeName}
	RunConfig RunConfig
	Enabled   bool
}

type UpdateServiceInput struct {
	ServiceInput
	Incremental bool
}

type RunConfig struct {
	Command   []string
	PseudoTTY int
}
