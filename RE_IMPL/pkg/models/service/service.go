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

package service

import "time"

type RepoModule struct {
	Id           string                  `json:"id"`
	Name         string                  `json:"name"`
	Desc         string                  `json:"description"`
	Version      string                  `json:"version"`
	Repositories []Repository            `json:"repositories"`
	Installed    *InstalledModuleVariant `json:"installed"`
}

type InstalledModuleVariant struct {
	ModuleVariant
	NextVersion string `json:"next_version"`
}

type RepoModulesFilter struct {
	Ids             []string           `json:"ids"`
	Name            string             `json:"name"`
	Repositories    []RepositoryFilter `json:"repositories"`
	Installed       bool               `json:"installed"`
	UpdateAvailable bool               `json:"update_available"`
}

type RepositoryFilter struct {
	Source   string   `json:"source"`
	Channels []string `json:"channels"`
}

type Repository struct {
	Source   string    `json:"source"`
	Priority int       `json:"priority"`
	Channels []Channel `json:"channels"`
}

type Channel struct {
	Name     string `json:"name"`
	Priority int    `json:"priority"`
	Version  string `json:"version"`
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

const (
	ChangeActionInstall = "install"
	ChangeActionChange  = "change"
	ChangeActionRemove  = "remove"
)

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
	Created time.Time             `json:"created"`
}

type ModulesFailedReport struct {
	ModuleAbbreviated
	Error string `json:"error"`
}
