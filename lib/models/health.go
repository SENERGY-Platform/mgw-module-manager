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

import "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"

// DeploymentsHealthInfo summarises the health of all enabled deployments.
// By default only unhealthy deployments are listed, which allows detecting problems by checking the length of Deployments.
type DeploymentsHealthInfo struct {
	Deployments             []DeploymentHealthInfo `json:"deployments"`               // health information per deployment, only unhealthy deployments unless DeploymentsHealthInfoFilter.IncludeHealthy is set
	TotalEnabledDeployments int                    `json:"total_enabled_deployments"` // number of enabled deployments that were checked, disabled deployments are never checked
}

// DeploymentHealthInfo provides the health of a single deployment and of the auxiliary deployments it owns.
type DeploymentHealthInfo struct {
	ModuleId                         string                          `json:"module_id"`                           // ID of the module the deployment is based on
	State                            constants.DeploymentState       `json:"state"`                               // health state derived from the states of all containers, values: 1 = healthy, 2 = unhealthy, 0 if it could not be determined
	Containers                       []DeploymentContainerHealthInfo `json:"containers"`                          // health information per container, only containers that are not ok unless DeploymentsHealthInfoFilter.IncludeHealthy is set
	TotalContainers                  int                             `json:"total_containers"`                    // number of containers the deployment consists of
	AuxiliaryDeployments             []AuxiliaryDeploymentHealthInfo `json:"auxiliary_deployments"`               // health information per auxiliary deployment, only auxiliary deployments that are not ok unless DeploymentsHealthInfoFilter.IncludeHealthy is set
	TotalEnabledAuxiliaryDeployments int                             `json:"total_enabled_auxiliary_deployments"` // number of enabled auxiliary deployments that were checked, disabled auxiliary deployments are never checked
	AuxiliaryDeploymentsState        constants.DeploymentState       `json:"auxiliary_deployments_state"`         // combined health state of all enabled auxiliary deployments, values: 1 = healthy, 2 = unhealthy, 0 if none were checked
}

// DeploymentContainerHealthInfo provides the health of a single container of a deployment.
type DeploymentContainerHealthInfo struct {
	Reference           string `json:"reference"` // reference of the module service the container is based on
	ContainerHealthInfo        // state and health of the container
}

// ContainerHealthInfo provides the state a container is reported to be in by the container engine.
// A container is considered ok if it is running and not unhealthy.
type ContainerHealthInfo struct {
	State  constants.ContainerState  `json:"state"`  // docker container state, values: initialized, running, paused, restarting, removing, stopped, dead
	Health constants.ContainerHealth `json:"health"` // docker container health, values: healthy, unhealthy, transitioning, empty if the container defines no health check
}

// AuxiliaryDeploymentHealthInfo provides the health of a single auxiliary deployment.
type AuxiliaryDeploymentHealthInfo struct {
	Id        string              `json:"id"`        // ID of the auxiliary deployment
	Reference string              `json:"reference"` // reference of the module auxiliary service the auxiliary deployment is based on
	Container ContainerHealthInfo `json:"container"` // state and health of the container of the auxiliary deployment
}

// DeploymentsHealthInfoFilter restricts which deployments are checked and how much detail is returned.
// All fields are optional, an empty field is not applied, exclusions take precedence over inclusions.
type DeploymentsHealthInfoFilter struct {
	ModuleIds               []string // only check deployments of these modules, all deployments are checked if empty
	ExclModuleIds           []string // do not check deployments of these modules
	AuxiliaryDeployments    bool     // include auxiliary deployments in the check
	AuxDeploymentsOfIds     []string // only check auxiliary deployments of deployments of these modules, all are checked if empty
	ExclAuxDeploymentsOfIds []string // do not check auxiliary deployments of deployments of these modules
	IncludeHealthy          bool     // also return deployments, containers and auxiliary deployments that are healthy instead of only those that are not
}
