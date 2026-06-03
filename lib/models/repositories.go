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

type RepoModule struct {
	Id                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Desc               string                 `json:"description"`
	Version            string                 `json:"version"`
	RepositoryVariants []RepoModuleVariant    `json:"repository_variants"`
	IsInstalled        bool                   `json:"is_installed"`
	InstalledVariant   InstalledModuleVariant `json:"installed_variant"`
}

type InstalledModuleVariant struct {
	ModuleVariant
	NextVersion string `json:"next_version"`
}

type RepoModulesFilter struct {
	Ids             []string                       `json:"ids"`
	Name            string                         `json:"name"`
	Repositories    []RepoModuleRepositoriesFilter `json:"repositories"`
	Installed       bool                           `json:"installed"`
	UpdateAvailable bool                           `json:"update_available"`
}

type RepoModuleRepositoriesFilter struct {
	Source   string   `json:"source"`
	Channels []string `json:"channels"`
}

type RepoModuleVariant struct {
	Source   string                     `json:"source"`
	Priority int                        `json:"priority"`
	Channels []RepoModuleVariantChannel `json:"channels"`
}

type RepoModuleVariantChannel struct {
	Name     string `json:"name"`
	Priority int    `json:"priority"`
	Version  string `json:"version"`
}

type Repository struct {
	Type     string
	Source   string
	Priority int
	Channels []RepositoryChannel
}

type RepositoryChannel struct {
	Name     string
	Priority int
}

type RepositoryJobResult struct {
	JobResult
	Results       []RepositoryResult
	ResultsErrNum int `json:"results_err_num"`
}

type RepositoryResult struct {
	Type          string                         `json:"type"`
	Source        string                         `json:"source"`
	ChannelErrors []RepositoryChannelErrorResult `json:"channel_errors"`
	ErrorResult
}

type RepositoryChannelErrorResult struct {
	Channel string `json:"channel"`
	ErrorResult
}
