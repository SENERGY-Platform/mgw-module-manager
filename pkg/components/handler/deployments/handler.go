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
	WorkDirPath     string        `json:"work_dir_path" env_var:"DEPLOYMENTS_HANDLER_WORK_DIR_PATH"`
	PathEscapeDepth int           `json:"path_escape_depth" env_var:"PATH_ESCAPE_DEPTH"`
	JobPollInterval time.Duration `json:"job_poll_interval" env_var:"DEPLOYMENTS_HANDLER_JOB_POLL_INTERVAL"`
	HostWorkDirPath string
	HostSecretsPath string
}

type Handler struct {
	storageHdl storageHandler
	cewClient  containerEngineWrapperClient
	hmClient   hostManagerClient
	smClient   secretManagerClient
	cmClient   coreManagerClient
	config     Config
	mu         sync.RWMutex
}

func New(
	storageHdl storageHandler,
	cewClient containerEngineWrapperClient,
	hmClient hostManagerClient,
	smClient secretManagerClient,
	cmClient coreManagerClient,
	config Config,
) *Handler {
	return &Handler{
		storageHdl: storageHdl,
		cewClient:  cewClient,
		hmClient:   hmClient,
		smClient:   smClient,
		cmClient:   cmClient,
		config:     config,
	}
}

func (h *Handler) Init() error {
	return os.MkdirAll(h.config.WorkDirPath, dirPerm)
}
