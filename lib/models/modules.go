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
	"time"

	external_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models/external"
)

type Module struct {
	external_models.ModuleLibModule
	Source     string     `json:"source"`
	Channel    string     `json:"channel"`
	Added      time.Time  `json:"added"`
	Updated    time.Time  `json:"updated"`
	IsDeployed bool       `json:"is_deployed"`
	Deployment Deployment `json:"deployment"`
}

type ModuleReduced struct {
	Id          string            `json:"id"`
	Source      string            `json:"source"`
	Channel     string            `json:"channel"`
	Version     string            `json:"version"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	License     string            `json:"license"`
	Author      string            `json:"author"`
	IsDeployed  bool              `json:"is_deployed"`
	Deployment  DeploymentReduced `json:"deployment"`
}

type DeploymentReduced struct {
	Id            string    `json:"id"`
	ModuleSource  string    `json:"module_source"`
	ModuleChannel string    `json:"module_channel"`
	ModuleVersion string    `json:"module_version"`
	Enabled       bool      `json:"enabled"`
	Created       time.Time `json:"created"`
	Updated       time.Time `json:"updated"`
	State         int       `json:"state"`
}

type ModulesFilter struct {
	Ids               []string
	Name              string
	Tags              []string
	Author            string
	IsDeployed        int
	DeploymentEnabled int
	DeploymentState   int
}

type ModuleVariant struct {
	Source  string `json:"source"`
	Channel string `json:"channel"`
	Version string `json:"version"`
}

type ChangeRequestItem struct {
	Id      string `json:"id"`
	Source  string `json:"source"`
	Channel string `json:"channel"`
	Remove  bool   `json:"remove"`
	Update  bool   `json:"update"`
}

type ModulesChangeRequest struct {
	Install []ModuleAbbreviated    `json:"install"`
	Change  [][2]ModuleAbbreviated `json:"change"`
	Remove  []string               `json:"remove"`
	Created time.Time              `json:"created"`
}

type ModuleAbbreviated struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Desc string `json:"description"`
	ModuleVariant
}

type ChangeReportItem struct {
	Id     string `json:"id"`
	Action string `json:"action"`
}

type ChangeReportErrItem struct {
	ChangeReportItem
	Error string `json:"error"`
}

type ModulesChangeReport struct {
	Success []ChangeReportItem    `json:"success"`
	Failed  []ChangeReportErrItem `json:"failed"`
}

type ModulesChangeJobResult struct {
	JobResult
	ModulesChangeReport
}
