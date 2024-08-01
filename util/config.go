/*
 * Copyright 2022 InfAI (CC SES)
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

package util

import (
	"github.com/SENERGY-Platform/go-service-base/config-hdl"
	cfg_types "github.com/SENERGY-Platform/go-service-base/config-hdl/types"
	"github.com/y-du/go-log-level/level"
)

type DatabaseConfig struct {
	Host       string           `json:"host" env_var:"DB_HOST"`
	Port       uint             `json:"port" env_var:"DB_PORT"`
	User       string           `json:"user" env_var:"DB_USER"`
	Passwd     cfg_types.Secret `json:"passwd" env_var:"DB_PASSWD"`
	Name       string           `json:"name" env_var:"DB_NAME"`
	Timeout    int64            `json:"timeout" env_var:"DB_TIMEOUT"`
	SchemaPath string           `json:"schema_path" env_var:"DB_SCHEMA_PATH"`
}

type HttpClientConfig struct {
	CewBaseUrl string `json:"cew_base_url" env_var:"CEW_BASE_URL"`
	CmBaseUrl  string `json:"cm_base_url" env_var:"CM_BASE_URL"`
	HmBaseUrl  string `json:"hm_base_url" env_var:"HM_BASE_URL"`
	SmBaseUrl  string `json:"sm_base_url" env_var:"SM_BASE_URL"`
	Timeout    int64  `json:"timeout" env_var:"HTTP_TIMEOUT"`
}

type ModHandlerConfig struct {
	WorkdirPath string `json:"workdir_path" env_var:"MH_WORKDIR_PATH"`
}

type ModTransferHandlerConfig struct {
	WorkdirPath string `json:"workdir_path" env_var:"MTH_WORKDIR_PATH"`
	Timeout     int64  `json:"timeout" env_var:"MTH_TIMEOUT"`
}

type ModStagingHandlerConfig struct {
	WorkdirPath string `json:"workdir_path" env_var:"MSH_WORKDIR_PATH"`
}

type DepHandlerConfig struct {
	WorkdirPath string `json:"workdir_path" env_var:"DH_WORKDIR_PATH"`
	HostDepPath string `json:"host_dep_path" env_var:"DH_HOST_DEP_PATH"`
	HostSecPath string `json:"host_sec_path" env_var:"DH_HOST_SEC_PATH"`
	ModuleNet   string `json:"module_net" env_var:"DH_MODULE_NET"`
}

type JobsConfig struct {
	BufferSize  int   `json:"buffer_size" env_var:"JOBS_BUFFER_SIZE"`
	MaxNumber   int   `json:"max_number" env_var:"JOBS_MAX_NUMBER"`
	CCHInterval int   `json:"cch_interval" env_var:"JOBS_CCH_INTERVAL"`
	JHInterval  int   `json:"jh_interval" env_var:"JOBS_JH_INTERVAL"`
	PJHInterval int64 `json:"pjh_interval" env_var:"JOBS_PJH_INTERVAL"`
	MaxAge      int64 `json:"max_age" env_var:"JOBS_MAX_AGE"`
}

type LoggerConfig struct {
	Level        level.Level `json:"level" env_var:"LOGGER_LEVEL"`
	Utc          bool        `json:"utc" env_var:"LOGGER_UTC"`
	Path         string      `json:"path" env_var:"LOGGER_PATH"`
	FileName     string      `json:"file_name" env_var:"LOGGER_FILE_NAME"`
	Terminal     bool        `json:"terminal" env_var:"LOGGER_TERMINAL"`
	Microseconds bool        `json:"microseconds" env_var:"LOGGER_MICROSECONDS"`
	Prefix       string      `json:"prefix" env_var:"LOGGER_PREFIX"`
}

type Config struct {
	ServerPort         uint                     `json:"server_port" env_var:"SERVER_PORT"`
	ModHandler         ModHandlerConfig         `json:"module_handler" env_var:"MH_CONFIG"`
	ModTransferHandler ModTransferHandlerConfig `json:"module_transfer_handler" env_var:"MTH_CONFIG"`
	ModStagingHandler  ModStagingHandlerConfig  `json:"module_staging_handler" env_var:"MSH_CONFIG"`
	DepHandler         DepHandlerConfig         `json:"deployment_handler" env_var:"DH_CONFIG"`
	Logger             LoggerConfig             `json:"logger" env_var:"LOGGER_CONFIG"`
	ConfigDefsPath     string                   `json:"config_defs_path" env_var:"CONFIG_DEFS_PATH"`
	Database           DatabaseConfig           `json:"database" env_var:"DATABASE_CONFIG"`
	HttpClient         HttpClientConfig         `json:"http_client" env_var:"HTTP_CLIENT_CONFIG"`
	Jobs               JobsConfig               `json:"jobs" env_var:"JOBS_CONFIG"`
	ManagerIDPath      string                   `json:"manager_id_path" env_var:"MANAGER_ID_PATH"`
	CoreID             string                   `json:"core_id" env_var:"CORE_ID"`
}

func NewConfig(path string) (*Config, error) {
	cfg := Config{
		ServerPort: 80,
		ModHandler: ModHandlerConfig{
			WorkdirPath: "/opt/module-manager/modules",
		},
		ModTransferHandler: ModTransferHandlerConfig{
			WorkdirPath: "/opt/module-manager/transfer",
			Timeout:     30000000000,
		},
		ModStagingHandler: ModStagingHandlerConfig{
			WorkdirPath: "/opt/module-manager/staging",
		},
		DepHandler: DepHandlerConfig{
			WorkdirPath: "/opt/module-manager/deployments",
			ModuleNet:   "module-net",
		},
		Logger: LoggerConfig{
			Level:        level.Warning,
			Utc:          true,
			Microseconds: true,
			Terminal:     true,
		},
		ConfigDefsPath: "include/config_definitions.json",
		Database: DatabaseConfig{
			Host:       "core-db",
			Port:       3306,
			Name:       "module_manager",
			Timeout:    5000000000,
			SchemaPath: "include/dep_storage_schema.sql",
		},
		HttpClient: HttpClientConfig{
			CewBaseUrl: "http://core-api/ce-wrapper",
			CmBaseUrl:  "http://core-api/core-manager",
			HmBaseUrl:  "http://core-api/host-manager",
			SmBaseUrl:  "http://secret-manager",
			Timeout:    10000000000,
		},
		Jobs: JobsConfig{
			BufferSize:  200,
			MaxNumber:   20,
			CCHInterval: 500000,
			JHInterval:  500000,
			PJHInterval: 300000000000,
			MaxAge:      172800000000000,
		},
		ManagerIDPath: "/opt/module-manager/data/mid",
	}
	err := config_hdl.Load(&cfg, nil, nil, nil, path)
	return &cfg, err
}
