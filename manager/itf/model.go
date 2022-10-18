/*
 * Copyright 2022 InfAI (CC SES)
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

package itf

import (
	"github.com/SENERGY-Platform/mgw-container-engine-manager-lib/cem-lib"
	"io/fs"
	"module-manager/manager/util"
)

type ModuleID string

type ModuleType string

type DeploymentType string

type DataType string

type SrvDepCondition string

type ResourceType string

// Modfile ------------------------------------->

type Module struct {
	ID             ModuleID           `json:"id"`
	Type           ModuleType         `json:"type"`
	Version        util.SemVersion    `json:"version"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	License        string             `json:"license"`
	Services       []Service          `json:"services"`
	Dependencies   []ModuleDependency `json:"dependencies"`
	DeploymentType DeploymentType     `json:"deployment_type"` // if MultipleDeployment the module can't be used as dependency
	UserInputs     *UserInputs        `json:"user_inputs"`
}

type Service struct {
	Name         string              `json:"name"`
	Image        string              `json:"image"`
	Include      []BindMount         `json:"include"` // files or dirs from module repo
	VolumeMounts []VolumeMount       `json:"volume_mounts"`
	TmpfsMounts  []TmpfsMount        `json:"tmpfs_mounts"`
	HttpApis     []HttpApi           `json:"http_apis"`
	PortBindings []PortBinding       `json:"port_bindings"`
	RunConfig    cem_lib.RunConfig   `json:"run_config"`
	Dependencies []ServiceDependency `json:"dependencies"`
	Environment  []EnvVar            `json:"environment"`
	Resources    []Resource          `json:"resources"`
}

type BindMount struct {
	MountPoint string `json:"mount_point"`
	Source     string `json:"source"` // relative path in module dir | prevent mounting of Modfile | must exist
	ReadOnly   bool   `json:"read_only"`
}

type VolumeMount struct {
	MountPoint string  `json:"mount_point"`
	Name       *string `json:"name"` // prefixed by module-manager
}

type TmpfsMount struct {
	MountPoint string      `json:"mount_point"`
	Size       int64       `json:"size"`
	Mode       fs.FileMode `json:"mode"`
}

type ModuleDependency struct {
	ModuleID      ModuleID             `json:"module_id"`
	ModuleVersion util.SemVersionRange `json:"module_version"`
	Services      []RequiredService    `json:"services"`
}

type RequiredService struct {
	Name       string             `json:"name"`
	RequiredBy []DependentService `json:"required_by"`
}

type DependentService struct {
	Name   string `json:"name"`
	EnvVar string `json:"env_var"` // container domain name provided by module-manager during deployment
}

type ServiceDependency struct {
	Name      string          `json:"name"`
	Condition SrvDepCondition `json:"condition"`
	EnvVar    string          `json:"env_var"` // container domain name provided by module-manager during deployment
}

type PortBinding struct {
	Name       *string          `json:"name"`
	Port       int              `json:"port"`
	TargetPort int              `json:"target_port"` // can be overridden by module-manager during deployment to avoid collisions
	Protocol   cem_lib.PortType `json:"protocol"`
}

type HttpApi struct {
	Name *string `json:"name"`
	Port int     `json:"port"`
	Path string  `json:"path"`
}

type EnvVar struct {
	Name     string  `json:"name"`
	Value    *string `json:"value"`
	InputRef *string `json:"input_ref"`
}

type Resource struct {
	ID         *string `json:"id"` // either set to known resource or set by user during deployment
	Name       *string `json:"name"`
	MountPoint string  `json:"mount_point"`
	ReadOnly   bool    `json:"read_only"`
	Meta       Meta    `json:"meta"`
	InputRef   *string `json:"input_ref"`
}

type Meta struct {
	Type string   `json:"type"` // via type map linking type to endpoint for ID | types: host-resource, secret-resource, ... | type map provided via service config
	Tags []string `json:"tags"`
}

type InputGroup struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	GroupRef    *string `json:"group_ref"`
}

type Input struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Value       InputValue `json:"value"` // populate with default on GET
	Meta        *Meta      `json:"meta"`  // populate on GET
	GroupRef    *string    `json:"group_ref"`
}

type InputValue struct {
	Type DataType `json:"type"`
	Data any      `json:"value"` // populate with default on GET
}

type UserInputs struct {
	InputGroups    map[string]InputGroup `json:"input_groups"`
	EnvVarInputs   map[string]Input      `json:"env_var_inputs"`
	ResourceInputs map[string]Input      `json:"resource_inputs"`
	SecretInputs   map[string]Input      `json:"secret_inputs"`
}

// <------------------------------------- Modfile

// Stored in Deployment DB --------------------->

type Deployment struct {
	ID            string // generated by module-manager during deployment
	Name          string // provided by user
	ModuleID      ModuleID
	ModuleVersion util.SemVersion
	InstanceIDs   []string
}

type SubDeployment struct {
	ID           string // generated by module-manager during deployment
	DeploymentID string // parent deployment id
	InstanceID   string
}

type Instance struct {
	ID          string
	ContainerID string
	DomainName  string
	ServiceName string
	Config      UserConfig
}

// <--------------------- Stored in Deployment DB
