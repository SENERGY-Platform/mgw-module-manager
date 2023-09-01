/*
 * Copyright 2023 InfAI (CC SES)
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

package model

const ServiceName = "mgw-module-manager"

const (
	HeaderRequestID = "X-Request-ID"
	HeaderApiVer    = "X-Api-Version"
	HeaderSrvName   = "X-Service"
)

const (
	ModulesPath           = "modules"
	ModUpdatesPath        = "updates"
	ModUptPreparePath     = "prepare"
	ModUptCancelPath      = "cancel"
	DeploymentsPath       = "deployments"
	DepTemplatePath       = "dep-template"
	DepUpdateTemplatePath = "upt-template"
	DepEnablePath         = "enable"
	DepDisablePath        = "disable"
	DepRestartPath        = "restart"
	DepHealthPath         = "health"
	JobsPath              = "jobs"
	JobsCancelPath        = "cancel"
)

const (
	Ascending  SortDirection = 0
	Descending SortDirection = 1
)

const (
	DepHealthy   HealthStatus = "healthy"
	DepUnhealthy HealthStatus = "unhealthy"
	DepTrans     HealthStatus = "transitioning"
)

const (
	JobPending   JobStatus = "pending"
	JobRunning   JobStatus = "running"
	JobCanceled  JobStatus = "canceled"
	JobCompleted JobStatus = "completed"
	JobError     JobStatus = "error"
	JobOK        JobStatus = "ok"
)

var JobStateMap = map[JobStatus]struct{}{
	JobPending:   {},
	JobRunning:   {},
	JobCanceled:  {},
	JobCompleted: {},
	JobError:     {},
	JobOK:        {},
}
