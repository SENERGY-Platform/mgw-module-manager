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

package handler_aux_deployments

import (
	"time"
)

type Config struct {
	PathEscapeDepth           int           `json:"path_escape_depth" env_var:"PATH_ESCAPE_DEPTH"`
	JobPollInterval           time.Duration `json:"job_poll_interval" env_var:"JOB_POLL_INTERVAL"`
	HostWorkDirPath           string        `json:"host_work_dir_path" env_var:"DEPLOYMENTS_HANDLER_HOST_WORK_DIR_PATH"`
	HealthMonitorStartupDelay time.Duration `json:"health_monitor_startup_delay" env_var:"AUX_DEPLOYMENTS_HANDLER_HEALTH_MONITOR_STARTUP_DELAY"`
	HealthMonitorLoopDelay    time.Duration `json:"health_monitor_loop_delay" env_var:"AUX_DEPLOYMENTS_HANDLER_HEALTH_MONITOR_LOOP_DELAY"`
}

type Handler struct {
	databaseHandler              databaseHandler
	containerEngineWrapperClient containerEngineWrapperClient
	config                       Config
}

func New(databaseHandler databaseHandler, containerEngineWrapperClient containerEngineWrapperClient, config Config) *Handler {
	return &Handler{
		databaseHandler:              databaseHandler,
		containerEngineWrapperClient: containerEngineWrapperClient,
		config:                       config,
	}
}
