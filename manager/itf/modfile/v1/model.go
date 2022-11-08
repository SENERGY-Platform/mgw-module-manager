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

package v1

import (
	"github.com/SENERGY-Platform/mgw-container-engine-manager-lib/cem-lib"
)

type Port string

type ByteFmt uint64

type Module struct {
	ID             string                      `yaml:"id"`             // url without schema (e.g. github.com/user/repo)
	Name           string                      `yaml:"name"`           // module name
	Description    string                      `yaml:"description"`    // short text describing the module
	License        string                      `yaml:"license"`        // module license name (e.g. Apache License 2.0)
	Author         string                      `yaml:"author"`         // module author
	Version        string                      `yaml:"version"`        // module version (must be prefixed with 'v' and adhere to the semantic versioning guidelines, see https://semver.org/ for details)
	Type           string                      `yaml:"type"`           // module type (e.g. device-connector specifies a module for integrating devices)
	DeploymentType string                      `yaml:"deploymentType"` // specifies whether a module can only be deployed once or multiple times
	Services       map[string]Service          `yaml:"services"`       // map depicting the services the module consists of (keys serve as unique identifiers and can be reused elsewhere in the modfile to reference a service)
	Volumes        map[string][]VolumeTarget   `yaml:"volumes"`        // map linking volumes to mount points (keys represent volume names)
	Dependencies   map[string]ModuleDependency `yaml:"dependencies"`   // external modules required by the module (keys represent module IDs)
	Resources      map[string]Resource         `yaml:"resources"`      // host resources required by services (e.g. devices, sockets, ...)
	Secrets        map[string]Secret           `yaml:"secrets"`        // secrets required by services (e.g. certs, keys, ...)
	Configs        map[string]ConfigValue      `yaml:"configs"`        // configuration values required by services
	InputGroups    map[string]InputGroup       `yaml:"inputGroups"`    // map of groups for categorising user inputs (keys serve as unique identifiers and can be reused elsewhere in the modfile to reference a group)
}

type Service struct {
	Name          string                             `yaml:"name"`          // service name
	Image         string                             `yaml:"image"`         // container image (must be versioned via tag or digest, e.g. srv-image:v1.0.0)
	RunConfig     cem_lib.RunConfig                  `yaml:"runConfig"`     // configurations for running the service container (e.g. restart strategy, stop timeout, ...)
	Include       []BindMount                        `yaml:"include"`       // files or dictionaries to be mounted from module repository
	Tmpfs         []TmpfsMount                       `yaml:"tmpfs"`         // temporary file systems (in memory) required by the service
	HttpEndpoints []HttpEndpoint                     `yaml:"httpEndpoints"` // http endpoints of the service to be exposed via the api gateway
	PortMappings  []PortMapping                      `yaml:"portMappings"`  // service ports to be published on the host
	Dependencies  map[string]ServiceDependencyTarget `yaml:"dependencies"`  // map depicting internal service dependencies (identifiers defined in Module.Services serve as keys)
}

type BindMount struct {
	MountPoint string `yaml:"mountPoint"` // absolute path in container
	Source     string `yaml:"source"`     // relative path in module repo
	ReadOnly   bool   `yaml:"readOnly"`
}

type TmpfsMount struct {
	MountPoint string            `yaml:"mountPoint"` // absolute path in container
	Size       ByteFmt           `yaml:"size"`       // tmpfs size in bytes provided as integer or in human-readable form (e.g. 64Mb)
	Mode       *cem_lib.FileMode `yaml:"mode"`       // linux file mode to be used for the tmpfs provided as string (e.g. 777, 0777)
}

type HttpEndpoint struct {
	Name   string  `yaml:"name"`   // endpoint name
	Path   string  `yaml:"path"`   // absolute path for the endpoint
	Port   *int    `yaml:"port"`   // port the service is listening on (defaults to 80 if nil)
	GwPath *string `yaml:"gwPath"` // optional relative path to be used by the api gateway
}

type PortMapping struct {
	Name     *string           `yaml:"name"`     // port name
	Port     Port              `yaml:"port"`     // port number provided as integer / string or port range provided as string (e.g. 8080-8081)
	HostPort *Port             `yaml:"hostPort"` // port number provided as integer / string or port range provided as string (e.g. 8080-8081), can be overridden during deployment to avoid collisions (arbitrary ports are used if nil)
	Protocol *cem_lib.PortType `yaml:"protocol"` // specify port protocol (defaults to tcp if nil)
}

type ServiceDependencyTarget struct {
	RefVar    string `yaml:"refVar"`    // environment variable to hold the addressable reference of the required service
	Condition string `yaml:"condition"` // running state of the required service
}

type VolumeTarget struct {
	MountPoint string   `yaml:"mountPoint"` // absolute path in container
	Services   []string `yaml:"services"`   // service identifiers as used in Module.Services to map the mount point to a number of services
}

type ModuleDependency struct {
	Version          string                              `yaml:"version"`          // version of required module (e.g. =v1.0.2, >v1.0.2., >=v1.0.2, >v1.0.2;<v2.1.3, ...)
	RequiredServices map[string][]ModuleDependencyTarget `yaml:"requiredServices"` // map linking required services to reference variables (identifiers as defined in Module.Services of the required module are used as keys)
}

type ModuleDependencyTarget struct {
	RefVar   string   `yaml:"refVar"`   // container environment variable to hold the addressable reference of the external service
	Services []string `yaml:"services"` // service identifiers as used in Module.Services to map the reference variable to a number of services
}

type ResourceTargetBase struct {
	MountPoint string   `yaml:"mountPoint"` // absolute path in container
	Services   []string `yaml:"services"`   // service identifiers as used in Module.Services to map the mount point to a number of services
}

type ResourceTarget struct {
	ResourceTargetBase `yaml:",inline"`
	ReadOnly           bool `yaml:"readOnly"` // if true resource will be mounted as read only
}

type Resource struct {
	Type      string           `yaml:"type"`      // resource type as defined by external services managing resources (e.g. serial-device, certificate, ...)
	Tags      []string         `yaml:"tags"`      // tags for aiding resource identification (e.g. a vendor), unique type and tag combinations can be used to select resources without requiring user interaction
	UserInput *UserInputBase   `yaml:"userInput"` // definitions for user input via gui (if nil the type and tag combination must yield a single resource)
	Targets   []ResourceTarget `yaml:"targets"`   // mount points for the resource
}

type Secret struct {
	Type      string               `yaml:"type"`      // resource type as defined by external services managing resources (e.g. serial-device, certificate, ...)
	Tags      []string             `yaml:"tags"`      // tags for aiding resource identification (e.g. a vendor), unique type and tag combinations can be used to select resources without requiring user interaction
	UserInput *UserInput           `yaml:"userInput"` // definitions for user input via gui (if nil the type and tag combination must yield a single secret)
	Targets   []ResourceTargetBase `yaml:"targets"`   // mount points for the secret
}

type ConfigValue struct {
	Value     any            `yaml:"value"`     // default configuration value or nil
	Options   []any          `yaml:"options"`   // list of possible configuration values
	Type      string         `yaml:"type"`      // data type of the configuration value
	UserInput *UserInput     `yaml:"userInput"` // definitions for user input via gui (if nil a default value must be set)
	Targets   []ConfigTarget `yaml:"targets"`   // reference variables for the configuration value
}

type ConfigTarget struct {
	RefVar   string   `yaml:"refVar"`   // container environment variable to hold the configuration value
	Services []string `yaml:"services"` // service identifiers as used in Module.Services to map the reference variable to a number of services
}

type UserInputBase struct {
	Name        string  `yaml:"name"`        // input name (e.g. used as a label for input field)
	Description *string `yaml:"description"` // short text describing the input
	Required    bool    `yaml:"required"`    // if true a user interaction is required
	Group       *string `yaml:"group"`       // group identifier as used in Module.InputGroups to assign the user input to an input group
}

type UserInput struct {
	UserInputBase `yaml:",inline"`
	Type          string         `yaml:"type"`        // type of the input (e.g. text, number, password, drop-down ...)
	Constraints   map[string]any `yaml:"constraints"` // constraints supported or required by the input type
}

type InputGroup struct {
	Name        string  `yaml:"name"`        // input group name
	Description *string `yaml:"description"` // short text describing the input group
	Group       *string `yaml:"group"`       // group identifier as used in Module.InputGroups to assign the input group to a parent group
}
