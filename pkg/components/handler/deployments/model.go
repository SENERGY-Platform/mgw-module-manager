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

package handler_deployments

import (
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/config"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

const (
	containersStateStopped = iota + 1 // no running containers
	containersStateRunning            // all containers running
	containersStatePartial            // one or more containers not running or restarting
	containersStateBroken             // one or more containers missing
)

const dirPerm = 0770

type defaultDataCollection struct {
	Configs map[string]models_config.Config
	Files   map[string][]byte
}
type userDataCollection struct {
	GlobalConfigs map[string]models_handler_database.DeploymentGlobalConfig
	HostResources map[string]models_handler_database.DeploymentHostResource
	Secrets       map[string]models_handler_database.DeploymentSecret
	Configs       map[string]models_handler_database.DeploymentUserConfig
	Files         map[string]models_handler_database.DeploymentFile
	FileGroups    map[string]models_handler_database.DeploymentFileGroup
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

type cacheCollection struct {
	HostResources map[string]models_external.HostResource
	GlobalConfigs map[string]models_handler_database.GlobalConfig
	SecretValues  map[string]models_external.SecretValueVariant
	Deployments   map[string]deploymentsCacheItem
}

type deploymentsCacheItem struct {
	DeploymentId string
	Containers   map[string]containerCacheItem
}

type containerCacheItem struct {
	Name  string
	Alias string
}
