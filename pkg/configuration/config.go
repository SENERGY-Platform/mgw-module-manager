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

package configuration

import (
	"encoding/base64"
	"encoding/json"
	"time"

	sb_config_hdl "github.com/SENERGY-Platform/go-service-base/config-hdl"
	sb_config_types "github.com/SENERGY-Platform/go-service-base/config-hdl/types"
	struct_logger "github.com/SENERGY-Platform/go-service-base/struct-logger"
	helper_sql_db "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/sql_db"
)

type MgwCoreConfig struct {
	CewBaseUrl string                   `json:"cew_base_url" env_var:"MGW_CEW_BASE_URL"`
	CmBaseUrl  string                   `json:"cm_base_url" env_var:"MGW_CM_BASE_URL"`
	HmBaseUrl  string                   `json:"hm_base_url" env_var:"MGW_HM_BASE_URL"`
	SmBaseUrl  string                   `json:"sm_base_url" env_var:"MGW_SM_BASE_URL"`
	Timeout    sb_config_types.Duration `json:"timeout" env_var:"MGW_HTTP_TIMEOUT"`
}

type SqlConfig = helper_sql_db.Config
type DatabaseConfig struct {
	Address               string                   `json:"address" env_var:"DATABASE_ADDRESS"`
	Database              string                   `json:"database" env_var:"DATABASE_NAME"`
	User                  string                   `json:"user" env_var:"DATABASE_USER"`
	Password              sb_config_types.Secret   `json:"password" env_var:"DATABASE_PASSWORD"`
	Timeout               sb_config_types.Duration `json:"timeout" env_var:"DATABASE_TIMEOUT"`
	MaxOpenConnections    int                      `json:"max_open_connections" env_var:"DATABASE_MAX_OPEN_CONNECTIONS"`
	MaxIdleConnections    int                      `json:"max_idle_connections" env_var:"DATABASE_MAX_IDLE_CONNECTIONS"`
	ConnectionMaxLifetime sb_config_types.Duration `json:"connection_max_lifetime" env_var:"DATABASE_CONNECTION_MAX_LIFETIME"`
}

type ModulesHandlerConfig struct {
	WorkdirPath string `json:"workdir_path" env_var:"MODULES_HANDLER_WORKDIR_PATH"`
}

type DeploymentsHandlerConfig struct {
	WorkdirPath                string                   `json:"workdir_path" env_var:"DEPLOYMENTS_HANDLER_WORKDIR_PATH"`
	RuntimeMonitorStartupDelay sb_config_types.Duration `json:"runtime_monitor_startup_delay" env_var:"DEPLOYMENTS_HANDLER_RUNTIME_MONITOR_STARTUP_DELAY"`
	RuntimeMonitorLoopDelay    sb_config_types.Duration `json:"runtime_monitor_loop_delay" env_var:"DEPLOYMENTS_HANDLER_RUNTIME_MONITOR_LOOP_DELAY"`
}

type AuxDeploymentsHandlerConfig struct {
	RuntimeMonitorStartupDelay sb_config_types.Duration `json:"runtime_monitor_startup_delay" env_var:"AUX_DEPLOYMENTS_HANDLER_RUNTIME_MONITOR_STARTUP_DELAY"`
	RuntimeMonitorLoopDelay    sb_config_types.Duration `json:"runtime_monitor_loop_delay" env_var:"AUX_DEPLOYMENTS_HANDLER_RUNTIME_MONITOR_LOOP_DELAY"`
}

type HostDirRepositoryHandlerConfig struct {
	WorkdirPath string `json:"workdir_path" env_var:"HOST_DIR_HANDLER_WORKDIR_PATH"`
	Priority    int    `json:"priority" env_var:"HOST_DIR_HANDLER_PRIORITY"`
}

type GitHubRepositoriesHandlerConfig struct {
	BaseUrl     string                   `json:"base_url" env_var:"GITHUB_HANDLER_BASE_URL"`
	WorkdirPath string                   `json:"workdir_path" env_var:"GITHUB_HANDLER_WORKDIR_PATH"`
	Timeout     sb_config_types.Duration `json:"timeout" env_var:"GITHUB_HANDLER_HTTP_TIMEOUT"`
}

type JobsHandlerConfig struct {
	MaxJobAge        sb_config_types.Duration `json:"max_job_age" env_var:"JOBS_HANDLER_MAX_JOB_AGE"`
	CleanupLoopDelay sb_config_types.Duration `json:"cleanup_loop_delay" env_var:"JOBS_HANDLER_CLEANUP_LOOP_DELAY"`
}

type LoggerConfig struct {
	struct_logger.Config
	HttpAccessLog bool `json:"http_access_log" env_var:"HTTP_ACCESS_LOG"`
}

type Config struct {
	ServerPort                uint                            `json:"server_port" env_var:"SERVER_PORT"`
	ManagerIdPath             string                          `json:"manager_id_path" env_var:"MANAGER_ID_PATH"`
	CoreId                    string                          `json:"core_id" env_var:"CORE_ID"`
	ModuleContainerNetwork    string                          `json:"module_container_network" env_var:"MODULE_CONTAINER_NETWORK"`
	UseUTC                    bool                            `json:"use_utc" env_var:"USE_UTC"`
	JobPollInterval           sb_config_types.Duration        `json:"job_poll_interval" env_var:"JOB_POLL_INTERVAL"`
	ImageNameEscapeDepth      int                             `json:"image_name_escape_depth" env_var:"IMAGE_NAME_ESCAPE_DEPTH"`
	HostDeploymentsPath       string                          `json:"host_deployments_path" env_var:"HOST_DEPLOYMENTS_PATH"`
	HostSecretsPath           string                          `json:"host_secrets_path" env_var:"HOST_SECRETS_PATH"`
	Logger                    LoggerConfig                    `json:"logger"`
	MgwCore                   MgwCoreConfig                   `json:"mgw_core"`
	Database                  DatabaseConfig                  `json:"database"`
	ModulesHandler            ModulesHandlerConfig            `json:"modules_handler"`
	DeploymentsHandler        DeploymentsHandlerConfig        `json:"deployments_handler"`
	AuxDeploymentsHandler     AuxDeploymentsHandlerConfig     `json:"aux_deployments_handler"`
	HostDirRepositoryHandler  HostDirRepositoryHandlerConfig  `json:"host_dir_repository_handler"`
	GitHubRepositoriesHandler GitHubRepositoriesHandlerConfig `json:"github_repositories_handler"`
	JobsHandler               JobsHandlerConfig               `json:"jobs_handler"`
}

var defaultConfig = Config{
	ServerPort:      80,
	ManagerIdPath:   "/opt/module-manager/data/mid",
	UseUTC:          true,
	JobPollInterval: sb_config_types.Duration(time.Millisecond * 500),
	Logger: LoggerConfig{
		Config: struct_logger.Config{
			Handler:    struct_logger.TextHandlerSelector,
			Level:      struct_logger.LevelInfo,
			TimeFormat: time.RFC3339Nano,
			TimeUtc:    true,
		},
	},
	MgwCore: MgwCoreConfig{
		Timeout: sb_config_types.Duration(time.Second * 30),
	},
	Database: DatabaseConfig{
		Database:              "module_manager",
		Timeout:               sb_config_types.Duration(time.Second * 30),
		MaxOpenConnections:    25,
		MaxIdleConnections:    25,
		ConnectionMaxLifetime: sb_config_types.Duration(time.Minute * 5),
	},
	ModulesHandler: ModulesHandlerConfig{
		WorkdirPath: "/opt/module-manager/modules",
	},
	DeploymentsHandler: DeploymentsHandlerConfig{
		WorkdirPath:                "/opt/module-manager/deployments",
		RuntimeMonitorStartupDelay: sb_config_types.Duration(time.Second * 30),
		RuntimeMonitorLoopDelay:    sb_config_types.Duration(time.Second * 5),
	},
	AuxDeploymentsHandler: AuxDeploymentsHandlerConfig{
		RuntimeMonitorStartupDelay: sb_config_types.Duration(time.Second * 30),
		RuntimeMonitorLoopDelay:    sb_config_types.Duration(time.Second * 5),
	},
	HostDirRepositoryHandler: HostDirRepositoryHandlerConfig{
		WorkdirPath: "/opt/module-manager/repositories/host_dir",
		Priority:    0,
	},
	GitHubRepositoriesHandler: GitHubRepositoriesHandlerConfig{
		BaseUrl:     "https://api.github.com",
		WorkdirPath: "/opt/module-manager/repositories/github",
		Timeout:     sb_config_types.Duration(time.Minute),
	},
	JobsHandler: JobsHandlerConfig{
		MaxJobAge:        sb_config_types.Duration(time.Hour * 24),
		CleanupLoopDelay: sb_config_types.Duration(time.Minute * 5),
	},
}

func New(path string) (Config, error) {
	cfg := defaultConfig
	err := sb_config_hdl.Load(&cfg, nil, envTypeParser, nil, path)
	return cfg, err
}

func ToBase64EncodedJson(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}
