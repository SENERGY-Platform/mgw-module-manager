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
	"module-manager/manager/util"
)

type ID string

type ModuleType string

type DeploymentType string

type ByteFmt uint64

type DataType string

type SrvDepCondition string

type Base struct {
	ID             ID              `json:"id" yaml:"id"`                          // url without schema (e.g. github.com/user/repo)
	Name           string          `json:"name" yaml:"name"`                      // module name
	Description    string          `json:"description" yaml:"description"`        // short text describing the module
	License        string          `json:"license" yaml:"license"`                // module license name (e.g. Apache License 2.0)
	Author         string          `json:"author" yaml:"author"`                  // module author
	Version        util.SemVersion `json:"version" yaml:"version"`                // module version (must be prefixed with 'v' and adhere to the semantic versioning guidelines, see https://semver.org/ for details)
	Type           ModuleType      `json:"type" yaml:"type"`                      // module type (e.g. device-connector specifies a module for integrating devices)
	DeploymentType DeploymentType  `json:"deployment_type" yaml:"deploymentType"` // specifies whether a module can only be deployed once or multiple times
}

type Module struct {
	Base
	Services     map[string]*Service     `json:"services"`     // {srvName:Service}
	Volumes      []string                `json:"volumes"`      // {volName}
	Dependencies map[ID]ModuleDependency `json:"dependencies"` // {moduleID:ModuleDependency}
	Resources    map[string]Resource     `json:"resources"`    // {ref:Resource}
	Secrets      map[string]Secret       `json:"secrets"`      // {ref:Secret}
	Configs      map[string]ConfigValue  `json:"configs"`      // {ref:ConfigValue}
	InputGroups  map[string]InputGroup   `json:"input_groups"` // {ref:InputGroup}
}

type ServiceBase struct {
	Name      string            `json:"name" yaml:"name"`            // service name
	Image     string            `json:"image" yaml:"image"`          // container image (must be versioned via tag or digest, e.g. srv-image:v1.0.0)
	RunConfig cem_lib.RunConfig `json:"run_config" yaml:"runConfig"` // configurations for running the service container (e.g. restart strategy, stop timeout, ...)
}

type Service struct {
	ServiceBase
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
	Source   string `json:"source" yaml:"source"` // relative path in module repo
	ReadOnly bool   `json:"read_only" yaml:"readOnly"`
}

type TmpfsMount struct {
	Size ByteFmt           `json:"size" yaml:"size"` // tmpfs size in bytes provided as integer or in human-readable form (e.g. 64Mb)
	Mode *cem_lib.FileMode `json:"mode" yaml:"mode"` // linux file mode to be used for the tmpfs provided as string (e.g. 777, 0777)
}

type HttpEndpoint struct {
	Name   string  `json:"name" yaml:"name"`      // endpoint name
	Port   *int    `json:"port" yaml:"port"`      // port the service is listening on (defaults to 80 if nil)
	GwPath *string `json:"gw_path" yaml:"gwPath"` // optional relative path to be used by the api gateway
}

type PortMapping struct {
	Name     *string           `json:"name"`
	Port     []int             `json:"port"`
	HostPort []int             `json:"host_port"`
	Protocol *cem_lib.PortType `json:"protocol"`
}

type ServiceDependencyTarget struct {
	Service   string          `json:"service"`
	Condition SrvDepCondition `json:"condition"`
}

type ExternalDependencyTarget struct {
	ID      ID     `json:"id"`
	Service string `json:"service"`
}

type ModuleDependency struct {
	Version          util.SemVersionRange `json:"version"`
	RequiredServices []string             `json:"required_services"` // {srvName}
}

type ResourceTarget struct {
	Reference string `json:"reference"`
	ReadOnly  bool   `json:"read_only"`
}

type ResourceBase struct {
	Type string   `json:"type" yaml:"type"` // resource type as defined by external services managing resources (e.g. serial-device, certificate, ...)
	Tags []string `json:"tags" yaml:"tags"` // tags for aiding resource identification (e.g. a vendor), unique type and tag combinations can be used to select resources without requiring user interaction
}

type Resource struct {
	ResourceBase `yaml:",inline"`
	UserInput    *UserInputBase `json:"user_input" yaml:"userInput"` // definitions for user input via gui (if nil the type and tag combination must yield a single resource)
}

type Secret struct {
	ResourceBase `yaml:",inline"`
	UserInput    *UserInput `json:"user_input" yaml:"userInput"` // definitions for user input via gui (if nil the type and tag combination must yield a single secret)
}

type ConfigValue struct {
	Value     any        `json:"value" yaml:"value"`          // default configuration value or nil
	Options   []any      `json:"options" yaml:"options"`      // list of possible configuration values
	Type      DataType   `json:"type" yaml:"type"`            // data type of the configuration value
	UserInput *UserInput `json:"user_input" yaml:"userInput"` // definitions for user input via gui (if nil a default value must be set)
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
