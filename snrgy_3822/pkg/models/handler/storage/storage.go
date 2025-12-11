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

package storage

import (
	"time"
)

type Module struct {
	Id      string
	DirName string
	Source  string
	Channel string
	Added   time.Time
	Updated time.Time
}

type ModulesFilter struct {
	Ids     []string
	Source  string
	Channel string
}

type Deployment struct {
	Id      string
	Module  DeploymentModule
	Name    string
	DirName string
	Enabled bool
	Created time.Time
	Updated time.Time
}

type DeploymentModule struct {
	Id      string
	Source  string
	Channel string
	Version string
}

type DeploymentContainer struct {
	Id           string
	DeploymentId string
	Reference    string
	Alias        string
	Order        int
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
	Name    string
	AsMount bool
	AsEnv   bool
}

type DeploymentConfig struct {
	DeploymentId string
	Reference    string
	String       string
	StringSlice  []string
	Int64        int64
	Int64Slice   []int64
	Float64      float64
	Float64Slice []float64
	Bool         bool
	BoolSlice    []bool
	DataType     int
	IsSlice      bool
}

const (
	StringType = iota
	Int64Type
	Float64Type
	BoolType
)

type DeploymentsFilter struct {
	Ids       []string
	ModuleIds []string
	Name      string
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
