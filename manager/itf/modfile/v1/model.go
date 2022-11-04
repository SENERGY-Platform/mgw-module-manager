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
	"module-manager/manager/itf/module"
	"module-manager/manager/util"
)

type Port string

type Module struct {
	module.Base  `yaml:",inline"`
	Services     map[string]Service             `yaml:"services"`     // map depicting the services the module consists of (keys serve as unique identifiers and can be reused elsewhere in the modfile to reference a service)
	Volumes      map[string][]VolumeTarget      `yaml:"volumes"`      // map linking volumes to mount points (keys represent volume names)
	Dependencies map[module.ID]ModuleDependency `yaml:"dependencies"` // external modules required by the module (keys represent module IDs)
	Resources    map[string]Resource            `yaml:"resources"`    // host resources required by services (e.g. devices, sockets, ...)
	Secrets      map[string]Secret              `yaml:"secrets"`      // secrets required by services (e.g. certs, keys, ...)
	Configs      map[string]ConfigValue         `yaml:"configs"`      // configuration values required by services
	InputGroups  map[string]module.InputGroup   `yaml:"inputGroups"`  // map of groups for categorising user inputs (keys serve as unique identifiers and can be reused elsewhere in the modfile to reference a group)
}

type Service struct {
	module.ServiceBase `yaml:",inline"`
	Include            []BindMount                        `json:"include" yaml:"include"`              // files or dictionaries to be mounted from module repository
	Tmpfs              []TmpfsMount                       `json:"tmpfs" yaml:"tmpfs"`                  // temporary file systems (in memory) required by the service
	HttpEndpoints      []HttpEndpoint                     `json:"http_endpoints" yaml:"httpEndpoints"` // http endpoints of the service to be exposed via the api gateway
	PortMappings       []PortMapping                      `json:"port_mappings" yaml:"portMappings"`   // service ports to be published on the host
	Dependencies       map[string]ServiceDependencyTarget `json:"dependencies" yaml:"dependencies"`    // map depicting internal service dependencies (identifiers defined in Module.Services serve as keys)
}

type BindMount struct {
	MountPoint       string `json:"mount_point" yaml:"mountPoint"` // absolute path in container
	module.BindMount `yaml:",inline"`
}

type TmpfsMount struct {
	MountPoint        string `json:"mount_point" yaml:"mountPoint"` // absolute path in container
	module.TmpfsMount `yaml:",inline"`
}

type HttpEndpoint struct {
	Path                string `json:"path" yaml:"path"` // absolute path for the endpoint
	module.HttpEndpoint `yaml:",inline"`
}

type PortMapping struct {
	Name     *string           `json:"name" yaml:"name"`          // port name
	Port     Port              `json:"port" yaml:"port"`          // port number provided as integer / string or port range provided as string (e.g. 8080-8081)
	HostPort *Port             `json:"host_port" yaml:"hostPort"` // port number provided as integer / string or port range provided as string (e.g. 8080-8081), can be overridden during deployment to avoid collisions (arbitrary ports are used if nil)
	Protocol *cem_lib.PortType `json:"protocol" yaml:"protocol"`  // specify port protocol (defaults to tcp if nil)
}

type ServiceDependencyTarget struct {
	RefVar    string                 `json:"ref_var" yaml:"refVar"`      // environment variable to hold the addressable reference of the required service
	Condition module.SrvDepCondition `json:"condition" yaml:"condition"` // running state of the required service
}

type VolumeTarget struct {
	MountPoint string   `json:"mount_point" yaml:"mountPoint"` // absolute path in container
	Services   []string `json:"services" yaml:"services"`      // service identifiers as used in Module.Services to map the mount point to a number of services
}

type ModuleDependency struct {
	Version          util.SemVersionRange                `json:"version" yaml:"version"`                    // version of required module (e.g. =v1.0.2, >v1.0.2., >=v1.0.2, >v1.0.2;<v2.1.3, ...)
	RequiredServices map[string][]ModuleDependencyTarget `json:"required_services" yaml:"requiredServices"` // map linking required services to reference variables (identifiers as defined in Module.Services of the required module are used as keys)
}

type ModuleDependencyTarget struct {
	RefVar   string   `json:"ref_var" yaml:"refVar"`    // container environment variable to hold the addressable reference of the external service
	Services []string `json:"services" yaml:"services"` // service identifiers as used in Module.Services to map the reference variable to a number of services
}

type ResourceTargetBase struct {
	MountPoint string   `json:"mount_point" yaml:"mountPoint"` // absolute path in container
	Services   []string `json:"services" yaml:"services"`      // service identifiers as used in Module.Services to map the mount point to a number of services
}

type ResourceTarget struct {
	ResourceTargetBase `yaml:",inline"`
	ReadOnly           bool `json:"read_only" yaml:"readOnly"` // if true resource will be mounted as read only
}

type Resource struct {
	module.Resource `yaml:",inline"`
	Targets         []ResourceTarget `json:"targets" yaml:"targets"` // mount points for the resource
}

type Secret struct {
	module.Secret `yaml:",inline"`
	Targets       []ResourceTargetBase `json:"targets" yaml:"targets"` // mount points for the secret
}

type ConfigValue struct {
	module.ConfigValue `yaml:",inline"`
	Targets            []ConfigTarget `json:"targets" yaml:"targets"` // reference variables for the configuration value
}

type ConfigTarget struct {
	RefVar   string   `json:"ref_var" yaml:"refVar"`    // container environment variable to hold the configuration value
	Services []string `json:"services" yaml:"services"` // service identifiers as used in Module.Services to map the reference variable to a number of services
}
