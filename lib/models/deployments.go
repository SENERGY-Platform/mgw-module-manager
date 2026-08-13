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

// Deployment is an installed module that has been configured and deployed, at most one deployment exists per module.
// It combines the user input the deployment was created with and the runtime information of the containers it consists of.
type Deployment struct {
	Id            string                         `json:"id"`             // unique ID of the deployment
	ModuleSource  string                         `json:"module_source"`  // source of the repository the deployed module version was installed from
	ModuleChannel string                         `json:"module_channel"` // channel of the repository the deployed module version was installed from
	ModuleVersion string                         `json:"module_version"` // version of the module the deployment was created for, can differ from the installed version until the deployment is updated
	Enabled       bool                           `json:"enabled"`        // true if the containers of the deployment are meant to be running, disabled deployments are not started and not health checked
	Created       time.Time                      `json:"created"`        // point in time at which the deployment was created
	Updated       time.Time                      `json:"updated"`        // point in time at which the deployment was last updated
	Containers    map[string]Container           `json:"containers"`     // containers of the deployment by module service reference
	Volumes       map[string]string              `json:"volumes"`        // {reference:name} docker volume names by module volume reference
	HostResources map[string]string              `json:"host_resources"` // {reference:hostResourceId} host resources set as user input by module host resource reference
	Secrets       map[string]DeploymentSecret    `json:"secrets"`        // secrets set as user input by module secret reference
	Configs       map[string]InterfaceValue      `json:"configs"`        // config values set as user input by module config reference
	GlobalConfigs map[string]string              `json:"global_configs"` // {reference:globalConfigId} global configs set as user input by module config reference
	Files         map[string]string              `json:"files"`          // {reference:data} file contents as base64 encoded strings by module file reference
	FileGroups    map[string]DeploymentFileGroup `json:"file_groups"`    // files set as user input by module file group reference
	State         int                            `json:"state"`          // health state determined by container states, values: 1 = healthy, 2 = unhealthy, 0 if the deployment is disabled or the state could not be determined
	ErrorResult                                  // set if the runtime information could not be retrieved, the deployment is then returned without container states
}

// DeploymentReduced is a Deployment without user input and container details.
// It is used where only the identity, the enabled flag and the health state of a deployment are of interest.
type DeploymentReduced struct {
	Id            string    `json:"id"`             // unique ID of the deployment
	ModuleSource  string    `json:"module_source"`  // source of the repository the deployed module version was installed from
	ModuleChannel string    `json:"module_channel"` // channel of the repository the deployed module version was installed from
	ModuleVersion string    `json:"module_version"` // version of the module the deployment was created for, can differ from the installed version until the deployment is updated
	Enabled       bool      `json:"enabled"`        // true if the containers of the deployment are meant to be running, disabled deployments are not started and not health checked
	Created       time.Time `json:"created"`        // point in time at which the deployment was created
	Updated       time.Time `json:"updated"`        // point in time at which the deployment was last updated
	State         int       `json:"state"`          // health state determined by container states, values: 1 = healthy, 2 = unhealthy, 0 if the deployment is disabled or the state could not be determined
	ErrorResult             // set if the runtime information could not be retrieved, the deployment is then returned without a state
}

// Container provides the identity and runtime state of a container managed by the module manager.
type Container struct {
	Name    string                    `json:"name"`     // name of the container as known to the container engine, generated by the module manager
	Alias   string                    `json:"alias"`    // network alias under which the container is reachable by other containers of the deployment
	ImageId string                    `json:"image_id"` // docker image id, empty if the container could not be found
	State   constants.ContainerState  `json:"state"`    // docker container state, values: initialized, running, paused, restarting, removing, stopped, dead
	Health  constants.ContainerHealth `json:"health"`   // docker container health, values: healthy, unhealthy, transitioning, empty if the container defines no health check
}

// DeploymentSecret is a secret of the secret service that was set as user input for a module secret reference.
type DeploymentSecret struct {
	Id    string                 `json:"id"` // ID of the secret in the secret service
	Items []DeploymentSecretItem // items of the secret and how they are provided to the containers
}

// DeploymentSecretItem states how a single item of a secret is passed to the containers of a deployment.
type DeploymentSecretItem struct {
	Name    string `json:"name"`     // name of the item within the secret, empty if the secret is used as a whole
	AsMount bool   `json:"as_mount"` // true if the item is mounted into the containers as a file
	AsEnv   bool   `json:"as_env"`   // true if the item is passed to the containers as an environment variable
}

// DeploymentFileGroup holds the files that were set as user input for a module file group reference.
// In contrast to module files, the number and paths of the files in a group are chosen by the user.
type DeploymentFileGroup struct {
	Id    string                    `json:"id"`    // unique ID of the file group, generated from the deployment ID and the module file group reference
	Files []DeploymentFileGroupFile `json:"files"` // files of the group
}

// DeploymentFileGroupFile is a single file of a DeploymentFileGroup.
type DeploymentFileGroupFile struct {
	Path   string `json:"path"`   // path of the file relative to the base path the module mounts the file group at
	Format string `json:"format"` // format of the file content, not interpreted by the module manager, content is defined by the module consuming the file group
	Data   string `json:"data"`   // file content as a base64 encoded string
}

// DeploymentUserInput provides the values a module requires to be deployed or updated.
// Which references must be provided is declared by the module, see ModuleBase.
type DeploymentUserInput struct {
	ModuleId      string                                             `json:"module_id"`      // ID of the module to deploy or update
	HostResources map[string]string                                  `json:"host_resources"` // {ref:resourceID} host resource IDs by module host resource reference
	Secrets       map[string]string                                  `json:"secrets"`        // {ref:secretID} secret service IDs by module secret reference
	Configs       map[string]interface{}                             `json:"configs"`        // {ref:value} config values by module config reference, must match the type declared by the module config, references not set fall back to the module default
	GlobalConfigs map[string]string                                  `json:"global_configs"` // {ref:configID} global config IDs by module config reference
	Files         map[string]string                                  `json:"files"`          // {ref:data} file contents as base64 encoded strings by module file reference, references not set fall back to the default data of the module file
	FileGroups    map[string]map[string]DeploymentFileGroupUserInput `json:"file_groups"`    // {ref:{path:FileGroupUserInput}} files by module file group reference and file path
}

// DeploymentFileGroupUserInput is the user input for a single file of a file group.
type DeploymentFileGroupUserInput struct {
	Format string `json:"format"` // format of the file content, not interpreted by the module manager, content is defined by the module consuming the file group
	Data   string `json:"data"`   // file content as a base64 encoded string
}

// DeploymentJobResult is the result of a job created by deploying modules.
type DeploymentJobResult struct {
	JobResult
	Results       []DeploymentResult `json:"results"`         // result per module the job was started for
	ResultsErrNum int                `json:"results_err_num"` // number of entries in Results that failed
}

// DeploymentUpdateJobResult is the result of a job created by updating deployments.
type DeploymentUpdateJobResult struct {
	JobResult
	Results       []DeploymentUpdateResult `json:"results"`         // result per module the job was started for
	ResultsErrNum int                      `json:"results_err_num"` // number of entries in Results that failed
}

// DeploymentUpdateResult reports the outcome of updating a single deployment and recreating the auxiliary deployments it owns.
type DeploymentUpdateResult struct {
	DeploymentResult
	AuxiliaryDeployments AuxiliaryDeploymentRecreateResult `json:"auxiliary_deployments"` // outcome of recreating the auxiliary deployments of the deployment
}

// DeploymentDeleteJobResult is the result of a job created by deleting deployments.
type DeploymentDeleteJobResult struct {
	JobResult
	Results       []DeploymentDeleteResult `json:"results"`         // result per module the job was started for
	ResultsErrNum int                      `json:"results_err_num"` // number of entries in Results that failed
}

// DeploymentDeleteResult reports the outcome of deleting a single deployment and the auxiliary deployments it owns.
type DeploymentDeleteResult struct {
	DeploymentResult
	AuxiliaryDeployments AuxiliaryDeploymentDeleteResult `json:"auxiliary_deployments"` // outcome of deleting the auxiliary deployments of the deployment
}

// DeploymentResult reports the outcome of an operation on the deployment of a single module.
type DeploymentResult struct {
	ModuleId    string `json:"module_id"` // ID of the module the operation was performed for
	Id          string `json:"id"`        // ID of the deployment, empty if the operation failed before the deployment was created
	ErrorResult        // set if the operation failed for this module
}
