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
	"github.com/SENERGY-Platform/go-service-base/srv-base"
	"github.com/y-du/go-log-level/level"
)

type DatabaseConfig struct {
	Host    string `json:"host" env_var:"DB_HOST"`
	Port    uint   `json:"port" env_var:"DB_PORT"`
	User    string `json:"user" env_var:"DB_USER"`
	Passwd  string `json:"passwd" env_var:"DB_PASSWD"`
	Name    string `json:"name" env_var:"DB_NAME"`
	Timeout int64  `json:"timeout" env_var:"DB_TIMEOUT"`
}

type HttpClientConfig struct {
	CewBaseUrl string `json:"cew_base_url" env_var:"CEW_BASE_URL"`
	Timeout    int64  `json:"timeout" env_var:"HTTP_TIMEOUT"`
}

type ModStorageHandlerConfig struct {
	WorkdirPath string `json:"workdir_path" env_var:"MSH_WORKDIR_PATH"`
	Delimiter   string `json:"delimiter" env_var:"MSH_DELIMITER"`
}

type ModTransferHandlerConfig struct {
	WorkdirPath string `json:"workdir_path" env_var:"MTH_WORKDIR_PATH"`
	Timeout     int64  `json:"timeout" env_var:"MTH_TIMEOUT"`
}

type Config struct {
	ServerPort        uint                    `json:"server_port" env_var:"SERVER_PORT"`
	ModStorageHandler ModStorageHandlerConfig `json:"module_storage_handler" env_var:"MSH_CONFIG"`
	Logger            srv_base.LoggerConfig   `json:"logger" env_var:"LOGGER_CONFIG"`
	ConfigDefsPath    string                  `json:"config_defs_path" env_var:"CONFIG_DEFS_PATH"`
	Database          DatabaseConfig          `json:"database" env_var:"DATABASE_CONFIG"`
	HttpClient        HttpClientConfig        `json:"http_client" env_var:"HTTP_CLIENT_CONFIG"`
}

func NewConfig(path *string) (*Config, error) {
	cfg := Config{
		ModStorageHandler: ModStorageHandlerConfig{
			WorkdirPath: "/opt/manager",
			Delimiter:   "_",
		},
		ModTransferHandler: ModTransferHandlerConfig{
			WorkdirPath: "/opt/manager",
			Timeout:     30000000000,
		},
		Logger: srv_base.LoggerConfig{
			Level:        level.Warning,
			Utc:          true,
			Microseconds: true,
			Terminal:     true,
		},
		ConfigDefsPath: "include/config_definitions.json",
		Database: DatabaseConfig{
			Port:    3306,
			Name:    "module_manager",
			Timeout: 5000000000,
		},
		HttpClient: HttpClientConfig{
			CewBaseUrl: "http://api-gateway/cew",
			Timeout:    10000000000,
		},
	}
	err := srv_base.LoadConfig(path, &cfg, nil, nil, nil)
	return &cfg, err
}
