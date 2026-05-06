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

import pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"

const (
	containersStateStopped   = iota + 1 // no running containers
	containersStateRunning              // all containers running
	containersStatePartial              // one or more containers not running or restarting
	containersStateUnhealthy            // all containers running but one or more is unhealthy
	containersStateBroken               // one or more containers missing
)

const dirPerm = 0770

type defaultDataCollection struct {
	Configs map[string]pkg_models.Value
	Files   map[string][]byte
}
type userDataCollection struct {
	GlobalConfigs map[string]pkg_models.DeploymentGlobalConfig
	HostResources map[string]pkg_models.DeploymentHostResource
	Secrets       map[string]pkg_models.DeploymentSecret
	Configs       map[string]pkg_models.DeploymentUserConfig
	Files         map[string]pkg_models.DeploymentFile
	FileGroups    map[string]pkg_models.DeploymentFileGroup
}

type bindMountDataCollection struct {
	Secrets    map[string]pkg_models.SecretPathVariant
	Files      map[string]string
	FileGroups map[string][]fileGroupMount
}

type fileGroupMount struct {
	FileName string
	Path     string
}

type cacheCollection struct {
	HostResources map[string]pkg_models.HostResource
	GlobalConfigs map[string]pkg_models.Config
	SecretValues  map[string]pkg_models.SecretValueVariant
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
