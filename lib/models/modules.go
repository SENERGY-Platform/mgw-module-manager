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
)

// Module is an installed module together with the origin it was installed from and its deployment, if one exists.
// The embedded ModuleBase states what has to be provided as DeploymentUserInput to deploy the module.
type Module struct {
	ModuleBase                        // module metadata as declared in the modfile
	Source      string                `json:"source"`      // source of the repository the module was installed from
	Channel     string                `json:"channel"`     // channel of the repository the module was installed from
	Added       time.Time             `json:"added"`       // point in time at which the module was installed
	Updated     time.Time             `json:"updated"`     // point in time at which the module was last updated
	Files       map[string]ModuleFile `json:"files"`       // files the module consumes by reference, overrides ModuleBase.Files to also expose the default data
	IsDeployed  bool                  `json:"is_deployed"` // true if a deployment exists for the module, Deployment is only populated if true
	Deployment  Deployment            `json:"deployment"`  // deployment of the module, zero value if IsDeployed is false
	ErrorResult                       // set if the deployment of the module could not be retrieved completely
}

// ModuleFile is a file a module consumes, extended by the default content shipped with the module.
type ModuleFile struct {
	ModuleFileBase        // file metadata as declared in the modfile
	DefaultData    string `json:"default_data"` // default file content as a base64 encoded string, used if no content is provided as user input
}

// ModuleReduced is an installed module without the modfile metadata.
// It is used for listings where the full module definition is not needed.
type ModuleReduced struct {
	Id          string            `json:"id"`          // unique ID of the module
	Source      string            `json:"source"`      // source of the repository the module was installed from
	Channel     string            `json:"channel"`     // channel of the repository the module was installed from
	Version     string            `json:"version"`     // installed version of the module
	Name        string            `json:"name"`        // human readable name of the module
	Description string            `json:"description"` // human readable description of the module
	Tags        []string          `json:"tags"`        // tags the module is categorised by, usable to filter modules
	License     string            `json:"license"`     // license the module is published under
	Author      string            `json:"author"`      // author of the module
	IsDeployed  bool              `json:"is_deployed"` // true if a deployment exists for the module, Deployment is only populated if true
	Deployment  DeploymentReduced `json:"deployment"`  // deployment of the module, zero value if IsDeployed is false
	ErrorResult                   // set if the deployment of the module could not be retrieved completely
}

// ModulesFilter restricts the installed modules that are returned.
// All fields are optional, an empty or zero field is not applied, set fields are combined conjunctively.
type ModulesFilter struct {
	Ids               []string // only return modules with one of these IDs
	Name              string   // only return modules whose name contains this string, matched case insensitively
	Tags              []string // only return modules carrying one of these tags, currently not evaluated
	Author            string   // only return modules of this author, matched exactly
	IsDeployed        int      // values: 1 = only deployed modules, -1 = only modules without a deployment, 0 = do not filter by deployment existence
	DeploymentEnabled int      // values: 1 = only modules with an enabled deployment, -1 = only modules with a disabled deployment, 0 = do not filter by enabled state, only applied to deployed modules
	DeploymentState   int      // only return modules whose deployment has this health state, values: 1 = healthy, 2 = unhealthy, 0 = do not filter by deployment state
}

// ModuleVariant identifies a specific version of a module within a specific repository channel.
// The same module can be provided by several repositories and channels, each of these combinations is a variant.
type ModuleVariant struct {
	Source  string `json:"source"`  // source of the repository providing the variant
	Channel string `json:"channel"` // channel of the repository providing the variant
	Version string `json:"version"` // version of the module the variant provides
}

// ChangeRequestItem requests a change for a single module, it is the user input for creating a modules change request.
// Exactly one intent must be expressed, either Remove, Update or a variant given by Source and Channel, setting both Remove and Update is rejected.
type ChangeRequestItem struct {
	Id      string `json:"id"`      // ID of the module to change
	Source  string `json:"source"`  // install or switch to the variant provided by the repository with this source, ignored if Remove or Update is set
	Channel string `json:"channel"` // install or switch to the variant provided by this repository channel, ignored if Remove or Update is set
	Remove  bool   `json:"remove"`  // remove the module, fails if a deployment exists for it, must not be combined with Update
	Update  bool   `json:"update"`  // update the module within its current source and channel, must not be combined with Remove
}

// ModulesChangeRequest is the pending plan of module installations, changes and removals including the dependencies they pull in.
// It is created for review and has to be executed explicitly, only one change request exists at a time.
type ModulesChangeRequest struct {
	Install []ModuleAbbreviated    `json:"install"` // modules that will be newly installed, including dependencies of requested modules
	Change  [][2]ModuleAbbreviated `json:"change"`  // modules that will be changed, item format: [currentVariant, nextVariant]
	Remove  []string               `json:"remove"`  // IDs of the modules that will be removed
	Created time.Time              `json:"created"` // point in time at which the change request was created
}

// ModuleAbbreviated identifies a module variant together with the information needed to present it for review.
type ModuleAbbreviated struct {
	Id            string `json:"id"`          // unique ID of the module
	Name          string `json:"name"`        // human readable name of the module
	Desc          string `json:"description"` // human readable description of the module
	ModuleVariant        // repository source, channel and version of the variant
}

// ChangeReportItem reports the action that was executed for a single module of a change request.
type ChangeReportItem struct {
	Id     string `json:"id"`     // ID of the module the action was executed for
	Action string `json:"action"` // executed action, values: install, change, remove
}

// ChangeReportErrItem reports a module of a change request whose action failed.
type ChangeReportErrItem struct {
	ChangeReportItem        // module and action that failed
	Error            string `json:"error"` // reason the action failed
}

// ModulesChangeReport lists the outcome of executing a modules change request per module.
type ModulesChangeReport struct {
	Success []ChangeReportItem    `json:"success"` // modules whose action was executed successfully
	Failed  []ChangeReportErrItem `json:"failed"`  // modules whose action failed, the remaining actions of the change request are still executed
}

// ModulesChangeJobResult is the result of a job created by executing a modules change request.
type ModulesChangeJobResult struct {
	JobResult
	ModulesChangeReport // outcome per module, empty if the job was aborted
}
