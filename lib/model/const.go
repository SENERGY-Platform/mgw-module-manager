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

const (
	HeaderRequestID = "X-Request-ID"
	HeaderApiVer    = "X-Api-Version"
	HeaderSrvName   = "X-Service"
	DepIdHeaderKey  = "X-MGW-DID"
)

const (
	ModulesPath                = "modules"
	ModUpdatesPath             = "updates"
	ModUptPreparePath          = "prepare"
	ModUptCancelPath           = "cancel"
	DeploymentsPath            = "deployments"
	DepTemplatePath            = "dep-template"
	DepUpdateTemplatePath      = "upt-template"
	DepBatchPath               = "deployments-batch"
	DepStartPath               = "start"
	DepStopPath                = "stop"
	DepRestartPath             = "restart"
	DepDeletePath              = "delete"
	AuxDeploymentsPath         = "aux-deployments"
	AuxDepBatchPath            = "aux-deployments-batch"
	JobsPath                   = "jobs"
	JobsCancelPath             = "cancel"
	DepAdvertisementsPath      = "dep-advertisements"
	DepAdvertisementsBatchPath = "dep-advertisements-batch"
	DiscoveryPath              = "discovery"
	SrvInfoPath                = "info"
	RestrictedPath             = "restricted"
)

const (
	DepHealthy   HealthState = "healthy"
	DepUnhealthy HealthState = "unhealthy"
	DepTrans     HealthState = "transitioning"
)

type ToggleVal = int8

const (
	No  ToggleVal = -1
	Yes ToggleVal = 1
)
