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

package models_external

import (
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	cm_model "github.com/SENERGY-Platform/mgw-core-manager/lib/model"
	job_hdl_lib "github.com/SENERGY-Platform/mgw-go-service-base/job-hdl/lib"
	hm_model "github.com/SENERGY-Platform/mgw-host-manager/lib/model"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	sm_model "github.com/SENERGY-Platform/mgw-secret-manager/pkg/api_model"
)

type ModuleLibModule = module_lib.Module
type ModuleLibService = module_lib.Service
type ModuleLibConfigs = module_lib.Configs
type ModuleLibConfigValue = module_lib.ConfigValue
type ModuleLibHttpEndpoint = module_lib.HttpEndpoint
type ModuleLibSrvRefTarget = module_lib.SrvRefTarget
type ModuleLibRunConfig = module_lib.RunConfig
type ModuleLibPort = module_lib.Port
type ModuleLibHostResTarget = module_lib.HostResTarget
type ModuleLibSecretTarget = module_lib.SecretTarget
type ModuleLibTmpfsMount = module_lib.TmpfsMount
type ModuleLibBindMount = module_lib.BindMount
type ModuleLibExtDependencyTarget = module_lib.ExtDependencyTarget
type ModuleLibHostResource = module_lib.HostResource
type ModuleLibFile = module_lib.File

const (
	ModuleLibBoolType    = module_lib.BoolType
	ModuleLibInt64Type   = module_lib.Int64Type
	ModuleLibFloat64Type = module_lib.Float64Type
	ModuleLibStringType  = module_lib.StringType
)

type Container = cew_model.Container
type ContainersFilter = cew_model.ContainerFilter
type Volume = cew_model.Volume
type VolumesFilter = cew_model.VolumeFilter
type Image = cew_model.Image
type ImagesFilter = cew_model.ImageFilter
type CEWNotFoundErr = cew_model.NotFoundError
type CewRunConfig = cew_model.RunConfig
type CewContainerNetwork = cew_model.ContainerNet
type CewPort = cew_model.Port
type CewPortBinding = cew_model.PortBinding
type CewMount = cew_model.Mount
type CewDevice = cew_model.Device

const (
	CewRunningState         = cew_model.RunningState
	CewUnhealthyState       = cew_model.UnhealthyState
	CewRestartStrategyNever = cew_model.RestartNever
	CewMountTypeVolume      = cew_model.VolumeMount
	CewMountTypeBind        = cew_model.BindMount
	CewMountTypeTmpfs       = cew_model.TmpfsMount
)

type Job = job_hdl_lib.Job

type HostResource = hm_model.HostResource

const (
	HostResourceTypeApp    = hm_model.Application
	HostResourceTypeDevice = hm_model.SerialDevice
)

type SecretVariantRequest = sm_model.SecretVariantRequest
type SecretPathVariant = sm_model.SecretPathVariant
type SecretValueVariant = sm_model.SecretValueVariant

type CmEndpointBase = cm_model.EndpointBase
type CmEndpointFiler = cm_model.EndpointFilter
type CmProxyConfig = cm_model.ProxyConfig
type CmStringSub = cm_model.StringSub
