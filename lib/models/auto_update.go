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

import "time"

// AutoUpdateSettings controls whether and when the gateway updates its installed modules and their deployments by itself.
// The window is evaluated on the wall clock of TimeZone and may cross midnight.
type AutoUpdateSettings struct {
	Enabled     bool   `json:"enabled"`      // true if modules are updated automatically, off by default
	Level       string `json:"level"`        // highest version change that is applied, values: patch, minor, major, default: patch
	WindowStart string `json:"window_start"` // start of the daily time window in which an update run may start, format HH:MM, inclusive, default: 02:00
	WindowEnd   string `json:"window_end"`   // end of the daily time window, format HH:MM, exclusive, must differ from WindowStart, default: 05:00
	TimeZone    string `json:"time_zone"`    // IANA time zone name the window is evaluated in, e.g. UTC or America/New_York, default: Europe/Berlin
}

// AutoUpdateStatus combines the auto-update settings with the outcome of the last update run.
type AutoUpdateStatus struct {
	AutoUpdateSettings                // current settings
	LastRun            *AutoUpdateRun `json:"last_run"` // outcome of the last update run, null if no run has completed yet
}

// AutoUpdateRun is the outcome of one auto-update run.
type AutoUpdateRun struct {
	Start       time.Time                 `json:"start"`   // point in time at which the run started
	End         time.Time                 `json:"end"`     // point in time at which the run ended
	Updated     []AutoUpdateUpdatedModule `json:"updated"` // modules that were updated together with their deployment, and deployments that were updated to their installed module
	Skipped     []AutoUpdateSkippedModule `json:"skipped"` // modules with a newer version and deployments lagging their installed module that were left unchanged
	Failed      []AutoUpdateFailedModule  `json:"failed"`  // modules whose update or deployment update failed
	ErrorResult                           // set if the run was aborted by an error that does not belong to a single module
}

// AutoUpdateUpdatedModule is a module the auto-update changed, including modules changed as dependency of another module,
// or a deployment it updated to the version of its installed module, e.g. after an earlier run was interrupted.
type AutoUpdateUpdatedModule struct {
	Id   string `json:"id"`   // ID of the module
	From string `json:"from"` // version installed before the run, for a deployment updated on its own the version the deployment was created for
	To   string `json:"to"`   // version installed by the run, for a deployment updated on its own the installed version
}

// AutoUpdateSkippedModule is an installed module with a newer version in its repository channel, or a deployment created
// for an older version of its installed module, that the auto-update did not change.
type AutoUpdateSkippedModule struct {
	Id          string             `json:"id"`           // ID of the module
	Version     string             `json:"version"`      // installed version, for a lagging deployment the version the deployment was created for
	NextVersion string             `json:"next_version"` // version available in the repository channel the module was installed from, for a lagging deployment the installed version
	Reason      string             `json:"reason"`       // why the module was not updated, values: level_exceeded, dependency_level_exceeded, new_dependency, needs_input, invalid_version, module_error, deployment_error, dependency_not_deployed, dependent_range_violated, deployment_variant_differs
	Missing     []CarryOverMissing `json:"missing"`      // items of the affected deployments that need user input, only set if Reason is needs_input
}

// AutoUpdateFailedModule is a module whose update was attempted and did not complete.
type AutoUpdateFailedModule struct {
	Id    string `json:"id"`    // ID of the module
	Error string `json:"error"` // reason the update failed
}
