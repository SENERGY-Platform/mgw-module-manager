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
	Labels    map[string]string                `json:"labels"`  // {name:value}
	Configs   map[string]string                `json:"configs"` // {varName:value}
	Volumes   []AuxiliaryDeploymentVolumeMount `json:"volumes"`
	Container Container                        `json:"container"`
}

type AuxiliaryDeploymentReduced struct {
	AuxiliaryDeploymentBase
	Container Container `json:"container"`
}

type AuxiliaryDeploymentVolumeMount struct {
	Reference string `json:"reference"`
	MountPath string `json:"mount_path"`
}

type AuxiliaryDeploymentRunConfig struct {
	Command   []string `json:"command"`
	PseudoTTY bool     `json:"pseudo_tty"`
}

type AuxiliaryDeploymentVolume struct {
	Id           string `json:"id"`
	DeploymentId string `json:"deployment_id"`
	Reference    string `json:"reference"`
	Name         string `json:"name"`
}

type AuxiliaryDeploymentVolumeWithMounts struct {
	AuxiliaryDeploymentVolume
	MountedBy []string `json:"mounted_by"`
}

type AuxiliaryDeploymentInputBase struct {
	Reference string                            `json:"reference"`
	Name      string                            `json:"name"`
	Image     string                            `json:"image"`
	PullImage bool                              `json:"pull_image"`
	Labels    map[string]string                 `json:"labels"`  // {name:value}
	Configs   map[string]string                 `json:"configs"` // {varName:value}
	Volumes   map[string]string                 `json:"volumes"` // {mntPath:reference}
	RunConfig AuxiliaryDeploymentInputRunConfig `json:"run_config"`
	Enabled   bool                              `json:"enabled"`
	Recreate  bool                              `json:"recreate"` // recreate the auxiliary deployment if parent deployment gets updated
}

type AuxiliaryDeploymentInputRunConfig struct {
	Command   []string `json:"command"`
	PseudoTTY int      `json:"pseudo_tty"`
}

type AuxiliaryDeploymentUpdateInputBase struct {
	AuxiliaryDeploymentInputBase
	Incremental bool `json:"incremental"`
}

type AuxiliaryDeploymentResult struct {
	Id             string `json:"id"`
	ContainerAlias string `json:"container_alias"`
}

type AuxiliaryDeploymentBatchResult struct {
	Id string `json:"id"`
	ErrorResult
}

type AuxiliaryDeploymentVolumeResult struct {
	Reference string `json:"reference"`
	ErrorResult
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

type AuxiliaryDeploymentInput struct {
	DeploymentId string `json:"deployment_id"`
	AuxiliaryDeploymentInputBase
}

type AuxiliaryDeploymentUpdateInput struct {
	DeploymentId    string `json:"deployment_id"`
	AuxDeploymentId string `json:"auxiliary_deployment_id"`
	AuxiliaryDeploymentUpdateInputBase
}

type AuxiliaryDeploymentCreateJobResult struct {
	JobResult
	AuxiliaryDeploymentResult
}

type AuxiliaryDeploymentJobResult struct {
	JobResult
	Results       []AuxiliaryDeploymentBatchResult `json:"results"`
	ResultsErrNum int                              `json:"results_err_num"`
}

type AuxiliaryDeploymentRecreateResult struct {
	ErrorResult
	Results       []AuxiliaryDeploymentBatchResult `json:"results"`
	ResultsErrNum int                              `json:"results_err_num"`
}

type AuxiliaryDeploymentDeleteResult struct {
	ErrorResult
	Results             []AuxiliaryDeploymentBatchResult  `json:"results"`
	ResultsErrNum       int                               `json:"results_err_num"`
	VolumeResults       []AuxiliaryDeploymentVolumeResult `json:"volume_results"`
	VolumeResultsErrNum int                               `json:"volume_results_err_num"`
}
