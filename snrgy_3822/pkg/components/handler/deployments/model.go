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

package deployments

import (
	"io/fs"

	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

const dirPerm = 0770

type deploymentWrapper struct {
	models_handler_storage.Deployment
	Configs          map[string]models_handler_storage.DeploymentUserConfig
	GlobalConfigs    map[string]models_handler_storage.DeploymentGlobalConfig
	HostResources    map[string]models_handler_storage.DeploymentHostResource
	Secrets          map[string]models_handler_storage.DeploymentSecret
	Files            map[string]models_handler_storage.DeploymentFile
	FileGroups       map[string]models_handler_storage.DeploymentFileGroup
	Containers       map[string]containerWrapper // {ref:containerWrapper}
	Volumes          map[string]models_handler_storage.DeploymentVolume
	Module           models_external.Module
	ModuleFileSystem fs.FS
	Error            error
}

type containerWrapper struct {
	models_handler_storage.DeploymentContainer
	Name string
}

type cacheWrapper struct {
	ExternalDependencies map[string]map[string]models_handler_storage.DeploymentContainer // {moduleId:{reference:DeploymentContainer}}
	HostResources        map[string]models_external.HostResource                          // {hostResourceId:HostResource}
	GlobalConfigs        map[string]models_handler_storage.GlobalConfig                   // {globalConfigId:GlobalConfig}
	SecretValues         map[string]models_external.SecretValueVariant                    // {secretId+itemName:SecretValueVariant}
}
