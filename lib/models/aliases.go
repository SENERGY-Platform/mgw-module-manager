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
	srv_info_hdl "github.com/SENERGY-Platform/go-service-base/srv-info-hdl"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
)

type ContainerState = string
type ContainerHealth = string

const (
	CewInitState       ContainerState = cew_model.InitState
	CewRunningState    ContainerState = cew_model.RunningState
	CewPausedState     ContainerState = cew_model.PausedState
	CewRestartingState ContainerState = cew_model.RestartingState
	CewRemovingState   ContainerState = cew_model.RemovingState
	CewStoppedState    ContainerState = cew_model.StoppedState
	CewDeadState       ContainerState = cew_model.DeadState
)

const (
	CewHealthyState    ContainerHealth = cew_model.HealthyState
	CewUnhealthyState  ContainerHealth = cew_model.UnhealthyState
	CewTransitionState ContainerHealth = cew_model.TransitionState
)

type ModuleBase = module_lib.Module

type ServiceInfo = srv_info_hdl.ServiceInfo
