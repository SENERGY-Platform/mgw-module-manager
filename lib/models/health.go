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

type DeploymentsHealthInfo struct {
	Deployments             []DeploymentHealthInfo `json:"deployments"`
	TotalEnabledDeployments int                    `json:"total_enabled_deployments"`
}

type DeploymentHealthInfo struct {
	ModuleId                         string                          `json:"module_id"`
	State                            DeploymentState                 `json:"state"`
	Containers                       []DeploymentContainerHealthInfo `json:"containers"`
	TotalContainers                  int                             `json:"total_containers"`
	AuxiliaryDeployments             []AuxiliaryDeploymentHealthInfo `json:"auxiliary_deployments"`
	TotalEnabledAuxiliaryDeployments int                             `json:"total_enabled_auxiliary_deployments"`
	AuxiliaryDeploymentsState        DeploymentState                 `json:"auxiliary_deployments_state"`
}

type DeploymentContainerHealthInfo struct {
	Reference string `json:"reference"`
	ContainerHealthInfo
}

type ContainerHealthInfo struct {
	State  ContainerState  `json:"state"`  // docker container state
	Health ContainerHealth `json:"health"` // docker container health
}

type AuxiliaryDeploymentHealthInfo struct {
	Id        string              `json:"id"`
	Reference string              `json:"reference"`
	Container ContainerHealthInfo `json:"container"`
}

type DeploymentsHealthInfoFilter struct {
	ModuleIds               []string
	ExclModuleIds           []string
	AuxiliaryDeployments    bool
	AuxDeploymentsOfIds     []string
	ExclAuxDeploymentsOfIds []string
	IncludeHealthy          bool
}
