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

package modfile

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

type Module struct {
	ID             ModuleID           `json:"id"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	License        string             `json:"license"`
	Author         string             `json:"author"`
	Version        util.SemVersion    `json:"version"`
	Type           ModuleType         `json:"type"`
	DeploymentType DeploymentType     `json:"deployment_type"` // if MultipleDeployment the module can't be used as dependency
	Services       []Service          `json:"services"`
	Volumes        []Volume           `json:"volumes"`
	Dependencies   []ModuleDependency `json:"dependencies"`
	Resources      []Resource         `json:"resources"`
	Secrets        []Secret           `json:"secrets"`
	Configs        []ConfigValue      `json:"configs"`
}

type Service struct {
	Name         string              `json:"name"`
	Image        string              `json:"image"`
	Include      []BindMount         `json:"include"` // files or dirs from module repo
	TmpfsMounts  []TmpfsMount        `json:"tmpfs_mounts"`
	HttpApis     []HttpApi           `json:"http_apis"`
	PortBindings []PortBinding       `json:"port_bindings"`
	Dependencies []ServiceDependency `json:"dependencies"`
	RunConfig    cem_lib.RunConfig   `json:"run_config"`
}

type BindMount struct {
	MountPoint string `json:"mount_point"`
	Source     string `json:"source"` // relative path in module dir | prevent mounting of Modfile | must exist
	ReadOnly   bool   `json:"read_only"`
}

type TmpfsMount struct {
	MountPoint string      `json:"mount_point"`
	Size       int64       `json:"size"`
	Mode       fs.FileMode `json:"mode"`
}

type HttpApi struct {
	Name *string `json:"name"`
	Port int     `json:"port"`
	Path string  `json:"path"`
}

type PortBinding struct {
	Name       *string          `json:"name"`
	Port       int              `json:"port"`
	TargetPort int              `json:"target_port"` // can be overridden by module-manager during deployment to avoid collisions
	Protocol   cem_lib.PortType `json:"protocol"`
}

type ServiceDependency struct {
	Name      string          `json:"name"`
	Condition SrvDepCondition `json:"condition"`
	EnvVar    string          `json:"env_var"` // container domain name provided by module-manager during deployment
}

type Volume struct {
	Name     *string        `json:"name"`
	Services []VolumeTarget `json:"services"`
}

type VolumeTarget struct {
	Name       string `json:"name"`
	MountPoint string `json:"mount_point"`
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

type Resource struct {
	ID        *string          `json:"id"`   // nil or known ID
	Type      string           `json:"type"` // via type-map linking type to endpoint for ID | types: host-resource, secret-resource, ... | type map provided via module-manager config?
	Tags      []string         `json:"tags"`
	Services  []ResourceTarget `json:"services"`
	UserInput *UserInput       `json:"user_input"`
}

type ResourceTarget struct {
	Name       string `json:"name"`
	MountPoint string `json:"mount_point"`
	ReadOnly   bool   `json:"read_only"`
}

type ConfigValue struct {
	Value     any            `json:"value"`   // nil or default value
	Options   []any          `json:"options"` // possible values
	Type      DataType       `json:"type"`
	Services  []ConfigTarget `json:"services"`
	UserInput *UserInput     `json:"user_input"`
}

type ConfigTarget struct {
	Name   string `json:"name"`
	EnvVar string `json:"env_var"`
}

type UserInput struct {
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	Type        string         `json:"type"` // https://developer.mozilla.org/en-US/docs/Web/HTML/Element/input#input_types
	Constraints map[string]any `json:"constraints"`
	Required    bool           `json:"required"`
}
