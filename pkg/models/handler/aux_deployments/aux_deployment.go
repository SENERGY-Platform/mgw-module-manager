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

	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_handler_database "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

type AuxiliaryDeploymentBase struct {
	Id           string                                               `json:"id"`
	DeploymentId string                                               `json:"deployment_id"`
	Reference    string                                               `json:"reference"`
	Name         string                                               `json:"name"`
	Image        string                                               `json:"image"`
	Created      time.Time                                            `json:"created"`
	Updated      time.Time                                            `json:"updated"`
	Enabled      bool                                                 `json:"enabled"`
	Recreate     bool                                                 `json:"recreate"`
	RunConfig    models_handler_database.AuxiliaryDeploymentRunConfig `json:"run_config"`
}

type AuxiliaryDeployment struct {
	AuxiliaryDeploymentBase
	Labels    map[string]string `json:"labels"`  // {name:value}
	Configs   map[string]string `json:"configs"` // {varName:value}
	Volumes   []Volume          `json:"volumes"`
	Container Container         `json:"container"`
}

type AuxiliaryDeploymentReduced struct {
	AuxiliaryDeploymentBase
	Container Container `json:"container"`
}

type Volume struct {
	Reference string `json:"reference"`
	MountPath string `json:"mount_path"`
}

type Container struct {
	Name    string `json:"name"`
	Alias   string `json:"alias"`
	ImageId string `json:"image_id"` // docker image id
	State   string `json:"state"`    // docker container state
	Health  string `json:"health"`   // docker container health
}

type AuxiliaryDeploymentsFilter struct {
	models_handler_database.AuxiliaryDeploymentsFilter
	State string // docker container state
}

type ServiceInput struct {
	Reference string                `json:"reference"`
	Name      string                `json:"name"`
	Image     string                `json:"image"`
	PullImage bool                  `json:"pull_image"`
	Labels    map[string]string     `json:"labels"`  // {name:value}
	Configs   map[string]string     `json:"configs"` // {varName:value}
	Volumes   map[string]string     `json:"volumes"` // {mntPath:reference}
	RunConfig ServiceInputRunConfig `json:"run_config"`
	Enabled   bool                  `json:"enabled"`
	Recreate  bool                  `json:"recreate"` // recreate the auxiliary deployment if parent deployment gets updated
}

type UpdateServiceInput struct {
	ServiceInput
	Incremental bool `json:"incremental"`
}

type ServiceInputRunConfig struct {
	Command   []string `json:"command"`
	PseudoTTY int      `json:"pseudo_tty"`
}

type Result struct {
	Id             string `json:"id"`
	ContainerAlias string `json:"container_alias"`
}

type BatchResult struct {
	Id string `json:"id"`
	models_error.ErrorResult
}

type VolumeResult struct {
	Reference string `json:"reference"`
	models_error.ErrorResult
}
