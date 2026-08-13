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

// RepoModule is a module offered by the configured repositories, aggregated over all repositories and channels providing it.
// It is the basis for deciding which variant of a module to install via a ChangeRequestItem.
// Name, Desc and Version are taken from the highest priority channel of one of the repository variants, use RepositoryVariants to address a specific variant.
type RepoModule struct {
	Id                 string                 `json:"id"`                  // unique ID of the module
	Name               string                 `json:"name"`                // human readable name of the module
	Desc               string                 `json:"description"`         // human readable description of the module
	Version            string                 `json:"version"`             // version of the module offered by the repositories
	RepositoryVariants []RepoModuleVariant    `json:"repository_variants"` // variants providing the module, sorted by repository priority in descending order
	IsInstalled        bool                   `json:"is_installed"`        // true if the module is installed, InstalledVariant is only populated if true
	InstalledVariant   InstalledModuleVariant `json:"installed_variant"`   // variant the module is installed from, zero value if IsInstalled is false
}

// InstalledModuleVariant is the variant a module is installed from together with the version it can be updated to.
type InstalledModuleVariant struct {
	ModuleVariant        // repository source, channel and version the module is installed from
	NextVersion   string `json:"next_version"` // version the module can be updated to within its source and channel, empty if no update is available
}

// RepoModulesFilter restricts the repository modules that are returned.
// All fields are optional, an empty or false field is not applied, set fields are combined conjunctively.
type RepoModulesFilter struct {
	Ids             []string                       `json:"ids"`              // only return modules with one of these IDs
	Name            string                         `json:"name"`             // only return modules whose name contains this string, matched case insensitively
	Repositories    []RepoModuleRepositoriesFilter `json:"repositories"`     // only return modules offered by these repositories and channels
	Installed       bool                           `json:"installed"`        // only return modules that are installed
	UpdateAvailable bool                           `json:"update_available"` // only return installed modules for which an update is available
}

// RepoModuleRepositoriesFilter restricts the repository modules to a single repository and optionally to some of its channels.
type RepoModuleRepositoriesFilter struct {
	Source   string   `json:"source"`   // source of the repository to restrict to
	Channels []string `json:"channels"` // channels of the repository to restrict to, all channels are used if empty
}

// RepoModuleVariant lists the channels of a single repository that offer a module.
type RepoModuleVariant struct {
	Source   string                     `json:"source"`   // source of the repository offering the module
	Priority int                        `json:"priority"` // priority of the repository, a higher value takes precedence when a module is offered by several repositories
	Channels []RepoModuleVariantChannel `json:"channels"` // channels of the repository offering the module, sorted by channel priority in descending order
}

// RepoModuleVariantChannel is a single channel of a repository offering a module.
type RepoModuleVariantChannel struct {
	Name     string `json:"name"`     // name of the channel
	Priority int    `json:"priority"` // priority of the channel, a higher value takes precedence when a module is offered by several channels of the same repository
	Version  string `json:"version"`  // version of the module the channel offers
}

// Repository is a configured source of modules.
type Repository struct {
	Type     string              // type of the repository, determines how it is accessed, values: github.com, host-dir
	Source   string              // source identifying the repository, must be unique across all repositories
	Priority int                 // priority of the repository, a higher value takes precedence when a module is offered by several repositories, must be unique across all repositories
	Channels []RepositoryChannel // channels the repository is split into
}

// RepositoryChannel is a channel of a repository, for example a release stage such as stable or testing.
type RepositoryChannel struct {
	Name     string // name of the channel, unique per repository
	Priority int    // priority of the channel, a higher value takes precedence when a module is offered by several channels of the same repository
}

// RepositoriesRefreshFilter restricts which repositories are refreshed.
// All fields are optional, an empty field is not applied, all repositories are refreshed if no field is set.
type RepositoriesRefreshFilter struct {
	Types   []string // only refresh repositories of these types, values: github.com, host-dir
	Sources []string // only refresh repositories with these sources
}

// RepositoryJobResult is the result of a job created by refreshing repositories.
type RepositoryJobResult struct {
	JobResult
	Results       []RepositoryResult // result per repository the job was started for
	ResultsErrNum int                `json:"results_err_num"` // number of entries in Results that failed
}

// RepositoryResult reports the outcome of refreshing a single repository.
type RepositoryResult struct {
	Type          string                         `json:"type"`           // type of the repository, values: github.com, host-dir
	Source        string                         `json:"source"`         // source of the repository
	Refresh       bool                           `json:"refresh"`        // true if the repository was refreshed, false if it did not match the filter and was skipped
	ChannelErrors []RepositoryChannelErrorResult `json:"channel_errors"` // errors that occurred for individual channels while the repository itself was refreshed
	ErrorResult                                  // set if the repository could not be refreshed at all
}

// RepositoryChannelErrorResult reports an error that occurred while refreshing a single channel of a repository.
type RepositoryChannelErrorResult struct {
	Channel     string `json:"channel"` // name of the channel that failed
	ErrorResult        // reason the channel could not be refreshed
}
