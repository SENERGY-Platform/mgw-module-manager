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

	sb_config_types "github.com/SENERGY-Platform/go-service-base/config-hdl/types"
)

type Config struct {
	WorkDirPath                string                   `json:"work_dir_path" env_var:"DEPLOYMENTS_HANDLER_WORK_DIR_PATH"`
	PathEscapeDepth            int                      `json:"path_escape_depth" env_var:"PATH_ESCAPE_DEPTH"`
	JobPollInterval            sb_config_types.Duration `json:"job_poll_interval" env_var:"JOB_POLL_INTERVAL"`
	HostWorkDirPath            string                   `json:"host_work_dir_path" env_var:"DEPLOYMENTS_HANDLER_HOST_WORK_DIR_PATH"`
	HostSecretsPath            string                   `json:"host_secrets_path" env_var:"HOST_SECRETS_PATH"`
	RuntimeMonitorStartupDelay sb_config_types.Duration `json:"runtime_monitor_startup_delay" env_var:"DEPLOYMENTS_HANDLER_RUNTIME_MONITOR_STARTUP_DELAY"`
	RuntimeMonitorLoopDelay    sb_config_types.Duration `json:"runtime_monitor_loop_delay" env_var:"DEPLOYMENTS_HANDLER_RUNTIME_MONITOR_LOOP_DELAY"`
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
	return os.MkdirAll(h.config.WorkDirPath, dirPerm)
}
