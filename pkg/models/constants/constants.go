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

package constants

const (
	EnvVariableDeploymentId    = "MGW_DID"
	EnvVariableAuxDeploymentId = "MGW_AID"
	EnvVariableCoreId          = "MGW_CID"
)

const (
	DeploymentAbbreviation    = "dep"
	AuxDeploymentAbbreviation = "aux-dep"
)

const (
	LabelCoreId                       = "mgw_cid"
	LabelManagerId                    = "mgw_mid"
	LabelDeploymentId                 = "mgw_did"
	LabelAuxDeploymentId              = "mgw_aid"
	LabelAuxDeploymentReference       = "mgw_aref"
	LabelAuxDeploymentVolumeId        = "mgw_advid"
	LabelVolumeReference              = "mgw_vref"
	LabelVolumeType                   = "mgw_vt"
	LabelServiceReference             = "mgw_sref"
	LabelHttpEndpointServiceReference = "srv_ref"
	LabelHttpEndpointModuleId         = "mod_id"
)

const (
	HeaderRequestId = "X-Request-Id"
	HeaderApiVer    = "X-Api-Version"
	HeaderSrvName   = "X-Service-Name"
	HeaderAuth      = "Authorization"
)

const (
	ValueDataTypeString = iota
	ValueDataTypeInt64
	ValueDataTypeFloat64
	ValueDataTypeBool
)

const (
	DeploymentStateHealthy = iota + 1
	DeploymentStateUnhealthy
)
