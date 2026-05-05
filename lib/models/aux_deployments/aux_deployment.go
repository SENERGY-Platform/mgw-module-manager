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

package aux_deployments

import (
	"time"
)

type AuxiliaryDeploymentBase struct {
	Id           string                       `json:"id"`
	DeploymentId string                       `json:"deployment_id"`
	Reference    string                       `json:"reference"`
	Name         string                       `json:"name"`
	Image        string                       `json:"image"`
	Created      time.Time                    `json:"created"`
	Updated      time.Time                    `json:"updated"`
	Enabled      bool                         `json:"enabled"`
	Recreate     bool                         `json:"recreate"`
	RunConfig    AuxiliaryDeploymentRunConfig `json:"run_config"`
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

type AuxiliaryDeploymentRunConfig struct {
	Command   []string `json:"command"`
	PseudoTTY bool     `json:"pseudo_tty"`
}

type AuxiliaryDeploymentsFilterWithState struct {
	AuxiliaryDeploymentsFilter
	State string // docker container state
}

type AuxiliaryDeploymentsFilter struct {
	Ids      []string
	Labels   map[string]string
	Image    string
	Enabled  int
	Recreate int
}
