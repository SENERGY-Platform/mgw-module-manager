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

package models

import (
	srv_info_hdl "github.com/SENERGY-Platform/go-service-base/srv-info-hdl"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
)

// ModuleBase provides the metadata of a module as declared in its modfile.
// It describes among others the services, volumes, dependencies, secrets, files and configs a module consists of and therefore what must be provided as user input to deploy it.
type ModuleBase = module_lib.Module

// ModuleFileBase provides the metadata of a file a module can consume.
// It states whether the file must be provided as user input to deploy the module.
type ModuleFileBase = module_lib.File

// ServiceInfo provides name, version, uptime and memory usage of the module manager service.
type ServiceInfo = srv_info_hdl.ServiceInfo
