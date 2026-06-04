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

package aux_deployments

import (
	"sync"
	"time"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/mutex_map"
)

type Config struct {
	PathEscapeDepth            int
	JobPollInterval            time.Duration
	HostDeploymentsPath        string
	RuntimeMonitorStartupDelay time.Duration
	RuntimeMonitorLoopDelay    time.Duration
}

type Handler struct {
	databaseHandler              databaseHandler
	containerEngineWrapperClient containerEngineWrapperClient
	config                       Config
	mutexes                      *mutex_map.RWMutexMap
	runtimeMonitorJobs           map[string]struct{}
	runtimeMonitorJobsMu         sync.RWMutex
}

func New(databaseHandler databaseHandler, containerEngineWrapperClient containerEngineWrapperClient, config Config) *Handler {
	return &Handler{
		databaseHandler:              databaseHandler,
		containerEngineWrapperClient: containerEngineWrapperClient,
		config:                       config,
		mutexes:                      mutex_map.New(),
	}
}
