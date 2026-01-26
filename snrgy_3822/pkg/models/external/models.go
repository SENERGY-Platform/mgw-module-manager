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

package external

import (
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	job_hdl_lib "github.com/SENERGY-Platform/mgw-go-service-base/job-hdl/lib"
	hm_model "github.com/SENERGY-Platform/mgw-host-manager/lib/model"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	sm_model "github.com/SENERGY-Platform/mgw-secret-manager/pkg/api_model"
)

type Module = module_lib.Module
type ModuleService = module_lib.Service
type ModuleConfigs = module_lib.Configs
type ModuleConfig = module_lib.ConfigValue
type ModuleConfigTypeOptions = module_lib.ConfigTypeOptions
type ModuleHostResource = module_lib.HostResource
type ModuleSecret = module_lib.Secret

const (
	ModuleConfigBoolType    = module_lib.BoolType
	ModuleConfigInt64Type   = module_lib.Int64Type
	ModuleConfigFloat64Type = module_lib.Float64Type
	ModuleConfigStringType  = module_lib.StringType
)

type Container = cew_model.Container
type ContainersFilter = cew_model.ContainerFilter
type Volume = cew_model.Volume
type VolumesFilter = cew_model.VolumeFilter
type Image = cew_model.Image
type ImagesFilter = cew_model.ImageFilter
type CEWNotFoundErr = cew_model.NotFoundError

const CewRunningState = cew_model.RunningState

type Job = job_hdl_lib.Job

type HostResource = hm_model.HostResource

type SecretVariantRequest = sm_model.SecretVariantRequest
type SecretPathVariant = sm_model.SecretPathVariant
type SecretValueVariant = sm_model.SecretValueVariant
