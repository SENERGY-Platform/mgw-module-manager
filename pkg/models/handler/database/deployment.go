/*
 * Copyright 2025 InfAI (CC SES)
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

package models_handler_database

import (
	"time"
)

type Deployment struct {
	Id            string
	ModuleId      string
	ModuleSource  string
	ModuleChannel string
	ModuleVersion string
	DirName       string
	FilesDirName  string
	Enabled       bool
	Created       time.Time
	Updated       time.Time
}

type DeploymentContainer struct {
	Name         string
	DeploymentId string
	Reference    string
	Alias        string
}

type DeploymentVolume struct {
	DeploymentId string
	Reference    string
	Name         string
}

type DeploymentHostResource struct {
	Id           string
	DeploymentId string
	Reference    string
}

type DeploymentSecret struct {
	Id           string
	DeploymentId string
	Reference    string
	Items        []DeploymentSecretItem
}

type DeploymentSecretItem struct {
	Name    string `json:"name"`
	AsMount bool   `json:"as_mount"`
	AsEnv   bool   `json:"as_env"`
}

type DeploymentUserConfig struct {
	Config
	DeploymentId string
	Reference    string
}

type DeploymentGlobalConfig struct {
	Id           string
	DeploymentId string
	Reference    string
}

type DeploymentFile struct {
	DeploymentId string
	Reference    string
	Data         []byte
}

type DeploymentFileGroup struct {
	Id           string
	DeploymentId string
	Reference    string
	Files        []DeploymentFileGroupFile
}

type DeploymentFileGroupFile struct {
	Path   string
	Format int
	Data   []byte
}

type DeploymentsFilter struct {
	Ids       []string
	ModuleIds []string
	Enabled   int
}

type DeploymentsHostResourcesFilter struct {
	Ids           []string
	DeploymentIds []string
}

type DeploymentsSecretsFilter struct {
	Ids           []string
	DeploymentIds []string
	AsMount       int
	AsEnv         int
}

type DeploymentGlobalConfigsFilter struct {
	Ids           []string
	DeploymentIds []string
}
