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

package deployments

import (
	"os"
	"sync"
	"time"
)

type Config struct {
	WorkdirPath                string
	PathEscapeDepth            int
	JobPollInterval            time.Duration
	HostWorkdirPath            string
	HostSecretsPath            string
	RuntimeMonitorStartupDelay time.Duration
	RuntimeMonitorLoopDelay    time.Duration
}

type Handler struct {
	databaseHandler              databaseHandler
	containerEngineWrapperClient containerEngineWrapperClient
	hostManagerClient            hostManagerClient
	secretManagerClient          secretManagerClient
	coreManagerClient            coreManagerClient
	config                       Config
	mu                           sync.RWMutex
	runtimeMonitorJobs           map[string]struct{}
	runtimeMonitorJobsMu         sync.RWMutex
}

func New(
	databaseHandler databaseHandler,
	containerEngineWrapperClient containerEngineWrapperClient,
	hostManagerClient hostManagerClient,
	secretManagerClient secretManagerClient,
	coreManagerClient coreManagerClient,
	config Config,
) *Handler {
	return &Handler{
		databaseHandler:              databaseHandler,
		containerEngineWrapperClient: containerEngineWrapperClient,
		hostManagerClient:            hostManagerClient,
		secretManagerClient:          secretManagerClient,
		coreManagerClient:            coreManagerClient,
		config:                       config,
		runtimeMonitorJobs:           make(map[string]struct{}),
	}
}

func (h *Handler) CreateWorkDir() error {
	return os.MkdirAll(h.config.WorkdirPath, dirPerm)
}
