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

package external

import (
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
)

const (
	CewInitState       = cew_model.InitState
	CewRunningState    = cew_model.RunningState
	CewPausedState     = cew_model.PausedState
	CewRestartingState = cew_model.RestartingState
	CewRemovingState   = cew_model.RemovingState
	CewStoppedState    = cew_model.StoppedState
	CewDeadState       = cew_model.DeadState
	CewHealthyState    = cew_model.HealthyState
	CewUnhealthyState  = cew_model.UnhealthyState
	CewTransitionState = cew_model.TransitionState
)

type ModuleLibModule = module_lib.Module
