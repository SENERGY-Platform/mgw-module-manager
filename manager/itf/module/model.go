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

package module

import (
	"github.com/SENERGY-Platform/mgw-container-engine-manager-lib/cem-lib"
)

type Module struct {
	ID             string                      `json:"id"`
	Name           string                      `json:"name"`
	Description    string                      `json:"description"`
	License        string                      `json:"license"`
	Author         string                      `json:"author"`
	Version        string                      `json:"version"`
	Type           string                      `json:"type"`
	DeploymentType string                      `json:"deployment_type"`
	Services       map[string]*Service         `json:"services"`     // {srvName:Service}
	Volumes        []string                    `json:"volumes"`      // {volName}
	Dependencies   map[string]ModuleDependency `json:"dependencies"` // {moduleID:ModuleDependency}
	Resources      map[string]Resource         `json:"resources"`    // {ref:Resource}
	Secrets        map[string]Secret           `json:"secrets"`      // {ref:Secret}
	Configs        map[string]ConfigValue      `json:"configs"`      // {ref:ConfigValue}
	InputGroups    map[string]InputGroup       `json:"input_groups"` // {ref:InputGroup}
}

type Service struct {
	Name                 string                              `json:"name"`
	Image                string                              `json:"image"`
	RunConfig            cem_lib.RunConfig                   `json:"run_config"`
	Include              map[string]BindMount                `json:"include"`               // {mntPoint:BindMount}
	Tmpfs                map[string]TmpfsMount               `json:"tmpfs"`                 // {mntPoint:TmpfsMount}
	Volumes              map[string]string                   `json:"volumes"`               // {mntPoint:volName}
	Resources            map[string]ResourceTarget           `json:"resources"`             // {mntPoint:ResourceTarget}
	Secrets              map[string]string                   `json:"secrets"`               // {mntPoint:ref}
	Configs              map[string]string                   `json:"configs"`               // {refVar:ref}
	HttpEndpoints        map[string]HttpEndpoint             `json:"http_endpoints"`        // {path:HttpEndpoint}
	Dependencies         map[string]ServiceDependencyTarget  `json:"dependencies"`          // {refVar:ServiceDependencyTarget}
	ExternalDependencies map[string]ExternalDependencyTarget `json:"external_dependencies"` // {refVar:ExternalDependencyTarget}
	PortMappings         []PortMapping                       `json:"port_mappings"`
}

type BindMount struct {
	Source   string `json:"source"`
	ReadOnly bool   `json:"read_only"`
}

type TmpfsMount struct {
	Size uint64            `json:"size"`
	Mode *cem_lib.FileMode `json:"mode"`
}

type HttpEndpoint struct {
	Name   string  `json:"name"`
	Port   *int    `json:"port"`
	GwPath *string `json:"gw_path"`
}

type PortMapping struct {
	Name     *string           `json:"name"`
	Port     []int             `json:"port"`
	HostPort []int             `json:"host_port"`
	Protocol *cem_lib.PortType `json:"protocol"`
}

type ServiceDependencyTarget struct {
	Service   string `json:"service"`
	Condition string `json:"condition"`
}

type ExternalDependencyTarget struct {
	ID      string `json:"id"`
	Service string `json:"service"`
}

type ModuleDependency struct {
	Version          string   `json:"version"`
	RequiredServices []string `json:"required_services"` // {srvName}
}

type ResourceTarget struct {
	Reference string `json:"reference"`
	ReadOnly  bool   `json:"read_only"`
}

type ResourceBase struct {
	Type string   `json:"type"`
	Tags []string `json:"tags"`
}

type Resource struct {
	ResourceBase
	UserInput *UserInputBase `json:"user_input"`
}

type Secret struct {
	ResourceBase
	UserInput *UserInput `json:"user_input"`
}

type ConfigValue struct {
	Value     any        `json:"value"`
	Options   []any      `json:"options"`
	Type      string     `json:"type"`
	UserInput *UserInput `json:"user_input"`
}

type UserInputBase struct {
	Name        string  `json:"name" yaml:"name"`               // input name (e.g. used as a label for input field)
	Description *string `json:"description" yaml:"description"` // short text describing the input
	Required    bool    `json:"required" yaml:"required"`       // if true a user interaction is required
	Group       *string `json:"group" yaml:"group"`             // group identifier as used in Module.InputGroups to assign the user input to an input group
}

type UserInput struct {
	UserInputBase `yaml:",inline"`
	Type          string         `json:"type" yaml:"type"`               // type of the input (e.g. text, number, password, drop-down ...)
	Constraints   map[string]any `json:"constraints" yaml:"constraints"` // constraints supported or required by the input type
}

type InputGroup struct {
	Name        string  `json:"name" yaml:"name"`               // input group name
	Description *string `json:"description" yaml:"description"` // short text describing the input group
	Group       *string `json:"group" yaml:"group"`             // group identifier as used in Module.InputGroups to assign the input group to a parent group
}
