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

import (
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
)

type ContainerState = string

const (
	ContainerInitialized ContainerState = cew_model.InitState
	ContainerRunning     ContainerState = cew_model.RunningState
	ContainerPaused      ContainerState = cew_model.PausedState
	ContainerRestarting  ContainerState = cew_model.RestartingState
	ContainerRemoving    ContainerState = cew_model.RemovingState
	ContainerStopped     ContainerState = cew_model.StoppedState
	ContainerDead        ContainerState = cew_model.DeadState
)

type ContainerHealth = string

const (
	ContainerHealthy       ContainerHealth = cew_model.HealthyState
	ContainerUnhealthy     ContainerHealth = cew_model.UnhealthyState
	ContainerTransitioning ContainerHealth = cew_model.TransitionState
)
