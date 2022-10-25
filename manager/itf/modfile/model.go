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
	"module-manager/manager/util"
)

type ModuleID string

type ModuleType string

type DeploymentType string

type Port string

type DataType string

type SrvDepCondition string

type ResourceType string

type Module struct {
	ID             ModuleID                      `json:"id" yaml:"id"`
	Name           string                        `json:"name" yaml:"name"`
	Description    string                        `json:"description" yaml:"description"`
	License        string                        `json:"license" yaml:"license"`
	Author         string                        `json:"author" yaml:"author"`
	Version        util.SemVersion               `json:"version" yaml:"version"`
	Type           ModuleType                    `json:"type" yaml:"type"`
	DeploymentType DeploymentType                `json:"deployment_type" yaml:"deploymentType"` // if MultipleDeployment the module can't be used as dependency
	Services       map[string]Service            `json:"services" yaml:"services"`
	Volumes        map[string][]VolumeTarget     `json:"volumes" yaml:"volumes"`
	Dependencies   map[ModuleID]ModuleDependency `json:"dependencies" yaml:"dependencies"`
	Resources      []Resource                    `json:"resources" yaml:"resources"`
	Secrets        []Secret                      `json:"secrets" yaml:"secrets"`
	Configs        []ConfigValue                 `json:"configs" yaml:"configs"`
}

type Service struct {
	Name          string                       `json:"name" yaml:"name"`
	Image         string                       `json:"image" yaml:"image"`
	Include       []BindMount                  `json:"include" yaml:"include"` // files or dirs from module repo
	Tmpfs         []TmpfsMount                 `json:"tmpfs" yaml:"tmpfs"`
	HttpEndpoints []HttpEndpoint               `json:"http_endpoints" yaml:"httpEndpoints"`
	PortMappings  []PortMapping                `json:"port_mappings" yaml:"portMappings"`
	Dependencies  map[string]ServiceDependency `json:"dependencies" yaml:"dependencies"`
	RunConfig     cem_lib.RunConfig            `json:"run_config" yaml:"runConfig"`
}

type BindMount struct {
	MountPoint string `json:"mount_point" yaml:"mountPoint"`
	Source     string `json:"source" yaml:"source"` // relative path in module dir | prevent mounting of Modfile | must exist
	ReadOnly   bool   `json:"read_only" yaml:"readOnly"`
}

type TmpfsMount struct {
	MountPoint string            `json:"mount_point" yaml:"mountPoint"`
	Size       int64             `json:"size" yaml:"size"`
	Mode       *cem_lib.FileMode `json:"mode" yaml:"mode"`
}

type HttpEndpoint struct {
	Name   string  `json:"name" yaml:"name"`
	Port   *int    `json:"port" yaml:"port"`
	Path   string  `json:"path" yaml:"path"`
	GwPath *string `json:"gw_path" yaml:"gwPath"`
}

type PortMapping struct {
	Name     *string           `json:"name" yaml:"name"`
	Port     Port              `json:"port" yaml:"port"`
	HostPort *Port             `json:"host_port" yaml:"hostPort"` // set by module-manager if empty or can be overridden by module-manager during deployment to avoid collisions
	Protocol *cem_lib.PortType `json:"protocol" yaml:"protocol"`
}

type ServiceDependency struct {
	Condition SrvDepCondition `json:"condition" yaml:"condition"`
	EnvVar    string          `json:"env_var" yaml:"envVar"` // container domain name provided by module-manager during deployment
}

type VolumeTarget struct {
	Service    *string `json:"service" yaml:"service"` // if empty use mount point for every service | allow for exceptions
	MountPoint string  `json:"mount_point" yaml:"mountPoint"`
}

type ModuleDependency struct {
	Version          util.SemVersionRange          `json:"version" yaml:"version"`
	RequiredServices map[string][]DependentService `json:"required_services" yaml:"requiredServices"`
}

type DependentService struct {
	Service *string `json:"service" yaml:"service"` // if empty use ref var for every service | allow for exceptions
	RefVar  string  `json:"ref_var" yaml:"refVar"`  // container domain name provided by module-manager during deployment
}

type ResourceBase struct {
	ID       *string          `json:"id" yaml:"id"`
	Tags     []string         `json:"tags" yaml:"tags"`
	Services []ResourceTarget `json:"services" yaml:"services"`
}

type ResourceTarget struct {
	Service    *string `json:"service" yaml:"service"` // if empty use mount point for every service | allow for exceptions
	MountPoint string  `json:"mount_point" yaml:"mountPoint"`
	ReadOnly   bool    `json:"read_only" yaml:"readOnly"`
}

type Resource struct {
	ResourceBase `yaml:",inline"`
	Type         string         `json:"type" yaml:"type"` // via type-map linking type to endpoint for ID | types: serial-port, uds-port, etc. | type map provided via module-manager config?
	UserInput    *UserInputBase `json:"user_input" yaml:"userInput"`
}

type Secret struct {
	ResourceBase `yaml:",inline"`
	UserInput    *UserInput `json:"user_input" yaml:"userInput"`
}

type ConfigValue struct {
	Value     any            `json:"value" yaml:"value"`     // nil or default value
	Options   []any          `json:"options" yaml:"options"` // possible values
	Type      DataType       `json:"type" yaml:"type"`
	Services  []ConfigTarget `json:"services" yaml:"services"`
	UserInput *UserInput     `json:"user_input" yaml:"userInput"`
}

type ConfigTarget struct {
	Service *string `json:"service" yaml:"service"` // if empty use ref var for every service | allow for exceptions
	RefVar  string  `json:"ref_var" yaml:"refVar"`
}

type UserInputBase struct {
	Name        string  `json:"name" yaml:"name"`
	Description *string `json:"description" yaml:"description"`
	Required    bool    `json:"required" yaml:"required"`
}

type UserInput struct {
	UserInputBase `yaml:",inline"`
	Type          string         `json:"type" yaml:"type"`
	Constraints   map[string]any `json:"constraints" yaml:"constraints"`
}
