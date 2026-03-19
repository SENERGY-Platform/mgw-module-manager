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
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

const dirPerm = 0770

type extendedDeployment struct {
	models_handler_storage.Deployment
	UserData      userDataCollection
	Containers    map[string]models_handler_storage.DeploymentContainer
	Volumes       map[string]models_handler_storage.DeploymentVolume
	MergedConfigs map[string]models_handler_storage.Config
	MergedFiles   map[string][]byte
}

type defaultDataCollection struct {
	Configs map[string]models_handler_storage.Config
	Files   map[string][]byte
}
type userDataCollection struct {
	GlobalConfigs map[string]models_handler_storage.DeploymentGlobalConfig
	HostResources map[string]models_handler_storage.DeploymentHostResource
	Secrets       map[string]models_handler_storage.DeploymentSecret
	Configs       map[string]models_handler_storage.DeploymentUserConfig
	Files         map[string]models_handler_storage.DeploymentFile
	FileGroups    map[string]models_handler_storage.DeploymentFileGroup
}

type containerEnvironmentDataCollection struct {
	SecretMounts    map[string]models_external.SecretPathVariant
	FileMounts      map[string]string
	FileGroupMounts map[string][]fileGroupMount
	Configs         map[string]string
}

type fileGroupMount struct {
	FileName string
	Path     string
}

type cacheCollection struct {
	HostResources    map[string]models_external.HostResource        // {hostResourceId:HostResource}
	GlobalConfigs    map[string]models_handler_storage.GlobalConfig // {globalConfigId:GlobalConfig}
	SecretValues     map[string]models_external.SecretValueVariant  // {secretId+itemName:SecretValueVariant}
	DeploymentIds    map[string]string                              // {moduleId:deploymentId}
	ContainerAliases map[string]map[string]string                   // {moduleId:{serviceReference:containerAlias}}
}
