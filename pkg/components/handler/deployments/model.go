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

type bindMountDataCollection struct {
	Secrets    map[string]models_external.SecretPathVariant
	Files      map[string]string
	FileGroups map[string][]fileGroupMount
}

type fileGroupMount struct {
	FileName string
	Path     string
}

type deploymentsCacheItem struct {
	DeploymentId     string
	ContainerAliases map[string]string
}
