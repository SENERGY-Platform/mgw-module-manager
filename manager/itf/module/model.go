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
	"io/fs"
	"reflect"
	"time"
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
	Services       map[string]*Service         `json:"services"`     // {ref:Service}
	Volumes        Set[string]                 `json:"volumes"`      // {volName}
	Dependencies   map[string]ModuleDependency `json:"dependencies"` // {moduleID:ModuleDependency}
	Resources      map[string]Resource         `json:"resources"`    // {ref:Resource}
	Secrets        map[string]Resource         `json:"secrets"`      // {ref:Resource}
	Configs        Configs                     `json:"configs"`      // {ref:ConfigValue}
	UserInput      UserInput                   `json:"user_input"`
}

type Set[T comparable] map[T]struct{}

type Service struct {
	Name                 string                              `json:"name"`
	Image                string                              `json:"image"`
	RunConfig            RunConfig                           `json:"run_config"`
	Include              map[string]BindMount                `json:"include"`               // {mntPoint:BindMount}
	Tmpfs                map[string]TmpfsMount               `json:"tmpfs"`                 // {mntPoint:TmpfsMount}
	Volumes              map[string]string                   `json:"volumes"`               // {mntPoint:volName}
	Resources            map[string]ResourceTarget           `json:"resources"`             // {mntPoint:ResourceTarget}
	Secrets              map[string]string                   `json:"secrets"`               // {mntPoint:ref}
	Configs              map[string]string                   `json:"configs"`               // {refVar:ref}
	HttpEndpoints        map[string]HttpEndpoint             `json:"http_endpoints"`        // {path:HttpEndpoint}
	Dependencies         map[string]ServiceDependencyTarget  `json:"dependencies"`          // {refVar:ServiceDependencyTarget}
	ExternalDependencies map[string]ExternalDependencyTarget `json:"external_dependencies"` // {refVar:ExternalDependencyTarget}
	PortMappings         PortMappings                        `json:"port_mappings"`
}

type RunConfig struct {
	RestartStrategy string         `json:"restart_strategy"`
	Retries         *int           `json:"retries"`
	RemoveAfterRun  bool           `json:"remove_after_run"`
	StopTimeout     *time.Duration `json:"stop_timeout"`
	StopSignal      *string        `json:"stop_signal"`
	PseudoTTY       bool           `json:"pseudo_tty"`
}

type BindMount struct {
	Source   string `json:"source"`
	ReadOnly bool   `json:"read_only"`
}

type TmpfsMount struct {
	Size uint64       `json:"size"`
	Mode *fs.FileMode `json:"mode"` // 0770 if nil
}

type HttpEndpoint struct {
	Name   string  `json:"name"`
	Port   *int    `json:"port"` // 80 if nil
	GwPath *string `json:"gw_path"`
}

type PortMappings map[string]portMapping

type portMapping struct {
	Name     *string `json:"name"`
	Port     []uint  `json:"port"`      // {n} || {s, e}
	HostPort []uint  `json:"host_port"` // {n} || {s, e}
	Protocol *string `json:"protocol"`  // tcp if nil
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
	Version          string      `json:"version"`
	RequiredServices Set[string] `json:"required_services"` // {ref}
}

type ResourceTarget struct {
	Ref      string `json:"ref"`
	ReadOnly bool   `json:"read_only"`
}

type Resource struct {
	Type string      `json:"type"` // resource type as defined by external services managing resources (e.g. serial-device, certificate, credentials ...)
	Tags Set[string] `json:"tags"`
}

type Configs map[string]configValue

type SliceOpt struct {
	dataType  reflect.Kind
	delimiter *string // ; if nil
}

type configValue struct {
	Default  any  `json:"default"`
	Options  any  `json:"options"`
	OptExt   bool `json:"opt_ext"`
	dataType reflect.Kind
	sliceOpt *SliceOpt
}

type InputBase struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Required    bool    `json:"required"`
	Group       *string `json:"group"`
}

type Input struct {
	InputBase
	Type        string         `json:"type"` // type of the input (e.g. text, number, user+password, drop-down ...)
	Constraints map[string]any `json:"constraints"`
}

type InputGroup struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Group       *string `json:"group"`
}

type UserInput struct {
	Resources map[string]InputBase  `json:"resources"` // {ref:InputBase}
	Secrets   map[string]Input      `json:"secrets"`   // {ref:Input}
	Configs   map[string]Input      `json:"configs"`   // {ref:Input}
	Groups    map[string]InputGroup `json:"groups"`    // {ref:InputGroup}
}
