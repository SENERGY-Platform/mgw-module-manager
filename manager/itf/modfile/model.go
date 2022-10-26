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

type ByteFmt uint64

type Port string

type DataType string

type SrvDepCondition string

type ResourceType string

type Module struct {
	ID             ModuleID                      `json:"id" yaml:"id"`                          // url without schema (e.g. github.com/user/repo)
	Name           string                        `json:"name" yaml:"name"`                      // module name
	Description    string                        `json:"description" yaml:"description"`        // short text describing the module
	License        string                        `json:"license" yaml:"license"`                // module license name (e.g. Apache License 2.0)
	Author         string                        `json:"author" yaml:"author"`                  // module author
	Version        util.SemVersion               `json:"version" yaml:"version"`                // module version (must be prefixed with 'v' and adhere to the semantic versioning guidelines, see https://semver.org/ for details)
	Type           ModuleType                    `json:"type" yaml:"type"`                      // module type (e.g. device-connector specifies a module for integrating devices)
	DeploymentType DeploymentType                `json:"deployment_type" yaml:"deploymentType"` // specifies whether a module can only be deployed once or multiple times
	Services       map[string]Service            `json:"services" yaml:"services"`              // map depicting the services the module consists of (keys serve as unique identifiers and can be reused elsewhere in the modfile to reference a service.)
	Volumes        map[string][]VolumeTarget     `json:"volumes" yaml:"volumes"`                // map linking volumes to mount points (keys represent volume names)
	Dependencies   map[ModuleID]ModuleDependency `json:"dependencies" yaml:"dependencies"`      // external modules required by the module (keys represent module IDs)
	Resources      []Resource                    `json:"resources" yaml:"resources"`            // host resources required by services (e.g. devices, sockets, ...)
	Secrets        []Secret                      `json:"secrets" yaml:"secrets"`                // secrets required by services (e.g. certs, keys, ...)
	Configs        []ConfigValue                 `json:"configs" yaml:"configs"`                // configuration values required by services
}

type Service struct {
	Name          string                       `json:"name" yaml:"name"`                    // service name
	Image         string                       `json:"image" yaml:"image"`                  // container image (must be versioned via tag or digest, e.g. srv-image:v1.0.0)
	Include       []BindMount                  `json:"include" yaml:"include"`              // files or dictionaries to be mounted from module repository
	Tmpfs         []TmpfsMount                 `json:"tmpfs" yaml:"tmpfs"`                  // temporary file systems (in memory) required by the service
	HttpEndpoints []HttpEndpoint               `json:"http_endpoints" yaml:"httpEndpoints"` // http endpoints of the service to be exposed via the api gateway
	PortMappings  []PortMapping                `json:"port_mappings" yaml:"portMappings"`   // service ports to be published on the host
	Dependencies  map[string]ServiceDependency `json:"dependencies" yaml:"dependencies"`    // map depicting internal service dependencies (identifiers defined in Module.Services serve as keys)
	RunConfig     cem_lib.RunConfig            `json:"run_config" yaml:"runConfig"`         // configurations for running the service container (e.g. restart strategy, stop timeout, ...)
}

type BindMount struct {
	MountPoint string `json:"mount_point" yaml:"mountPoint"` // absolute path in container
	Source     string `json:"source" yaml:"source"`          // relative path in module repo
	ReadOnly   bool   `json:"read_only" yaml:"readOnly"`
}

type TmpfsMount struct {
	MountPoint string            `json:"mount_point" yaml:"mountPoint"` // absolute path in container
	Size       ByteFmt           `json:"size" yaml:"size"`              // tmpfs size in bytes provided as integer or in human-readable form (e.g. 64Mb)
	Mode       *cem_lib.FileMode `json:"mode" yaml:"mode"`              // linux file mode to be used for the tmpfs provided as string (e.g. 777, 0777)
}

type HttpEndpoint struct {
	Name   string  `json:"name" yaml:"name"`      // endpoint name
	Port   *int    `json:"port" yaml:"port"`      // port the service is listening on (defaults to 80 if nil)
	Path   string  `json:"path" yaml:"path"`      // absolute path for the endpoint
	GwPath *string `json:"gw_path" yaml:"gwPath"` // optional relative path to be used by the api gateway
}

type PortMapping struct {
	Name     *string           `json:"name" yaml:"name"`          // port name
	Port     Port              `json:"port" yaml:"port"`          // port number provided as integer / string or port range provided as string (e.g. 8080-8081)
	HostPort *Port             `json:"host_port" yaml:"hostPort"` // port number provided as integer / string or port range provided as string (e.g. 8080-8081), can be overridden during deployment to avoid collisions (arbitrary ports are used if nil)
	Protocol *cem_lib.PortType `json:"protocol" yaml:"protocol"`  // specify port protocol (defaults to tcp if nil)
}

type ServiceDependency struct {
	Condition SrvDepCondition `json:"condition" yaml:"condition"` // running state of the required service
	RefVar    string          `json:"ref_var" yaml:"refVar"`      // environment variable to hold the addressable reference of the required service
}

type VolumeTarget struct {
	Service    *string `json:"service" yaml:"service"`        // service identifier as used in Module.Services to map the mount point to a specific service (if nil the mount point is used for all services, combinations are possible)
	MountPoint string  `json:"mount_point" yaml:"mountPoint"` // absolute path in container
}

type ModuleDependency struct {
	Version          util.SemVersionRange          `json:"version" yaml:"version"`                    // version of required module (e.g. =v1.0.2, >v1.0.2., >=v1.0.2, >v1.0.2;<v2.1.3, ...)
	RequiredServices map[string][]DependentService `json:"required_services" yaml:"requiredServices"` // map linking required services to reference variables (identifiers as defined in Module.Services of the required module are used as keys)
}

type DependentService struct {
	Service *string `json:"service" yaml:"service"` // service identifier as used in Module.Services to map the reference variable to a specific service (if nil the reference variable is used for all services, combinations are possible)
	RefVar  string  `json:"ref_var" yaml:"refVar"`  // container environment variable to hold the addressable reference of the external service
}

type ResourceBase struct {
	Type string   `json:"type" yaml:"type"` // resource type as defined by external services managing resources (e.g. serial-device, certificate, ...)
	Tags []string `json:"tags" yaml:"tags"` // tags for aiding resource identification (e.g. a specific vendor), unique type and tag combinations can be used to select resources without requiring user interaction
}

type ResourceTargetBase struct {
	Service    *string `json:"service" yaml:"service"`        // service identifier as used in Module.Services to map the mount point to a specific service (if nil the mount point is used for all services, combinations are possible)
	MountPoint string  `json:"mount_point" yaml:"mountPoint"` // absolute path in container
}

type ResourceTarget struct {
	ResourceTargetBase `yaml:",inline"`
	ReadOnly           bool `json:"read_only" yaml:"readOnly"` // if true resource will be mounted as read only
}

type Resource struct {
	ResourceBase `yaml:",inline"`
	Services     []ResourceTarget `json:"services" yaml:"services"`    // mount points for the resource
	UserInput    *UserInputBase   `json:"user_input" yaml:"userInput"` // definitions for user input via gui (if nil the type and tag combination must yield a single resource)
}

type Secret struct {
	ResourceBase `yaml:",inline"`
	Services     []ResourceTargetBase `json:"services" yaml:"services"`    // mount points for the secret
	UserInput    *UserInput           `json:"user_input" yaml:"userInput"` // definitions for user input via gui (if nil the type and tag combination must yield a single secret)
}

type ConfigValue struct {
	Value     any            `json:"value" yaml:"value"`          // default configuration value or nil
	Options   []any          `json:"options" yaml:"options"`      // list of possible configuration values
	Type      DataType       `json:"type" yaml:"type"`            // data type of the configuration value
	Services  []ConfigTarget `json:"services" yaml:"services"`    // reference variables for the configuration value
	UserInput *UserInput     `json:"user_input" yaml:"userInput"` // definitions for user input via gui (if nil a default value must be set)
}

type ConfigTarget struct {
	Service *string `json:"service" yaml:"service"` // service identifier as used in Module.Services to map the reference variable to a specific service (if nil the reference variable is used for all services, combinations are possible)
	RefVar  string  `json:"ref_var" yaml:"refVar"`  // container environment variable to hold the configuration value
}

type UserInputBase struct {
	Name        string  `json:"name" yaml:"name"`               // input name (e.g. used as a label for input field)
	Description *string `json:"description" yaml:"description"` // short text describing the input
	Required    bool    `json:"required" yaml:"required"`       // if true a user interaction is required
}

type UserInput struct {
	UserInputBase `yaml:",inline"`
	Type          string         `json:"type" yaml:"type"`               // type of the input (e.g. text, number, password, drop-down ...)
	Constraints   map[string]any `json:"constraints" yaml:"constraints"` // constraints supported or required by the input type
}
