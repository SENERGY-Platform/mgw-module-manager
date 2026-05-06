/*
 * Copyright 2026 InfAI (CC SES)
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

package attr_keys

import "github.com/SENERGY-Platform/go-service-base/struct-logger/attributes"

const (
	RequestId       = "request_id"
	StackTrace      = "stack_trace"
	Id              = "id"
	ModuleId        = "module_id"
	DeploymentId    = "deployment_id"
	AuxDeploymentId = "auxiliary_deployment_id"
	Filter          = "filter"
	Source          = "source"
	Channel         = "channel"
	DirName         = "dir_name"
	Signal          = "signal"
	Version         = "version"
	ConfigValues    = "config_values"
	Component       = "component"
	Description     = "description"
	Error           = attributes.ErrorKey
	Method          = attributes.MethodKey
	Path            = attributes.PathKey
)
