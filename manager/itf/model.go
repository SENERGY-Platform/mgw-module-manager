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
	"io/fs"
	"module-manager/manager/util/set"
	"time"
)

// Module ---------------------------------------------------------------------------------------

type ModuleType = string

type DeploymentType = string

type Module struct {
	ID             string                      `json:"id"`
	Name           string                      `json:"name"`
	Description    string                      `json:"description"`
	Tags           set.Set[string]             `json:"tags"`
	License        string                      `json:"license"`
	Author         string                      `json:"author"`
	Version        string                      `json:"version"`
	Type           ModuleType                  `json:"type"`
	DeploymentType DeploymentType              `json:"deployment_type"`
	Services       map[string]*Service         `json:"services"`     // {ref:Service}
	Volumes        set.Set[string]             `json:"volumes"`      // {volName}
	Dependencies   map[string]ModuleDependency `json:"dependencies"` // {moduleID:ModuleDependency}
	Resources      map[string]set.Set[string]  `json:"resources"`    // {ref:{tag}}
	Secrets        map[string]Secret           `json:"secrets"`      // {ref:Secret}
	Configs        Configs                     `json:"configs"`      // {ref:ConfigValue}
	Inputs         Inputs                      `json:"inputs"`
}

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
	SrvReferences        map[string]string                   `json:"srv_references"`        // {refVar:ref}
	HttpEndpoints        map[string]HttpEndpoint             `json:"http_endpoints"`        // {externalPath:HttpEndpoint}
	Dependencies         map[string]ServiceDependencyTarget  `json:"dependencies"`          // {refVar:ServiceDependencyTarget}
	ExternalDependencies map[string]ExternalDependencyTarget `json:"external_dependencies"` // {refVar:ExternalDependencyTarget}
	PortMappings         PortMappings                        `json:"port_mappings"`
}

type RunConfig struct {
	MaxRetries     *int           `json:"max_retries"` // '5' if nil
	RemoveAfterRun bool           `json:"remove_after_run"`
	StopTimeout    *time.Duration `json:"stop_timeout"`
	StopSignal     *string        `json:"stop_signal"`
	PseudoTTY      bool           `json:"pseudo_tty"`
}

type BindMount struct {
	Source   string `json:"source"`
	ReadOnly bool   `json:"read_only"`
}

type TmpfsMount struct {
	Size uint64       `json:"size"`
	Mode *fs.FileMode `json:"mode"` // '0770' if nil
}

type HttpEndpoint struct {
	Name string `json:"name"`
	Port *int   `json:"port"` // '80' if nil
	Path string `json:"path"` // internal path
}

type PortMappings map[string]portMapping

type PortProtocol = string

type portMapping struct {
	Name     *string       `json:"name"`
	Port     []uint        `json:"port"`      // {n} || {s, e}
	HostPort []uint        `json:"host_port"` // {n} || {s, e}
	Protocol *PortProtocol `json:"protocol"`  // 'tcp' if nil
}

type ServiceCondition = string

type ServiceDependencyTarget struct {
	Service   string           `json:"service"`
	Condition ServiceCondition `json:"condition"`
}

type ExternalDependencyTarget struct {
	ID      string `json:"id"`
	Service string `json:"service"`
}

type ModuleDependency struct {
	Version          string          `json:"version"`
	RequiredServices set.Set[string] `json:"required_services"` // {ref}
}

type ResourceTarget struct {
	Ref      string `json:"ref"`
	ReadOnly bool   `json:"read_only"`
}

type Secret struct {
	Type string          `json:"type"` // types are defined by secret-manager
	Tags set.Set[string] `json:"tags"`
}

type Configs map[string]configValue

type configValue struct {
	Default   any               `json:"default"`
	Options   any               `json:"options"`
	OptExt    bool              `json:"opt_ext"`
	Type      string            `json:"type"`
	TypeOpt   ConfigTypeOptions `json:"type_opt"`
	DataType  DataType          `json:"data_type"`
	IsSlice   bool              `json:"is_slice"`
	Delimiter *string           `json:"delimiter"` // ';' if nil (only applies if IsSlice == true)
}

type ConfigTypeOptions map[string]configTypeOption

type DataType uint

type configTypeOption struct {
	Value    any      `json:"value"`
	DataType DataType `json:"data_type"`
}

type Input struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Required    bool    `json:"required"`
	Group       *string `json:"group"`
}

type InputGroup struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Group       *string `json:"group"`
}

type Inputs struct {
	Resources map[string]Input      `json:"resources"` // {ref:Input}
	Secrets   map[string]Input      `json:"secrets"`   // {ref:Input}
	Configs   map[string]Input      `json:"configs"`   // {ref:Input}
	Groups    map[string]InputGroup `json:"groups"`    // {ref:InputGroup}
}

// Config Definition ----------------------------------------------------------------------------

type ConfigDefinition struct {
	DataType   set.Set[DataType]                 `json:"data_type"`
	Options    map[string]ConfigDefinitionOption `json:"options"`
	Validators []ConfigDefinitionValidator       `json:"validators"`
}

type ConfigDefinitionOption struct {
	DataType set.Set[DataType] `json:"data_type"`
	Inherit  bool              `json:"inherit"`
	Required bool              `json:"required"`
}

type ConfigDefinitionValidator struct {
	Name      string                                    `json:"name"`
	Parameter map[string]ConfigDefinitionValidatorParam `json:"parameter"`
}

type ConfigDefinitionValidatorParam struct {
	Value any     `json:"value"`
	Ref   *string `json:"ref"`
}

// Deployment -----------------------------------------------------------------------------------

type DeploymentBase struct {
	Name      *string           `json:"name"` // module name if nil
	ModuleID  string            `json:"module_id"`
	Resources map[string]string `json:"resources"` // {ref:resourceID}
	Secrets   map[string]string `json:"secrets"`   // {ref:secretID}
	Configs   map[string]any    `json:"configs"`   // {ref:value}
}

type Deployment struct {
	ID string `json:"id"`
	DeploymentBase
	Containers set.Set[string]
}

type DeploymentsPostRequest struct {
	DeploymentBase
	SecretRequests map[string]any // {ref:value}
}

// Input Template -------------------------------------------------------------------------------

type InputTemplate struct {
	Resources   map[string]InputTemplateResource `json:"resources"`    // {ref:ResourceInput}
	Secrets     map[string]InputTemplateResource `json:"secrets"`      // {ref:SecretInput}
	Configs     map[string]InputTemplateConfig   `json:"configs"`      // {ref:ConfigInput}
	InputGroups map[string]InputGroup            `json:"input_groups"` // {ref:InputGroup}
}

type InputTemplateResource struct {
	Input
	Resource
}

type InputTemplateConfig struct {
	Input
	Default  any            `json:"default"`
	Options  any            `json:"options"`
	OptExt   bool           `json:"opt_ext"`
	Type     string         `json:"type"`
	TypeOpt  map[string]any `json:"type_opt"`
	DataType DataType       `json:"data_type"`
	IsList   bool           `json:"is_list"`
}
