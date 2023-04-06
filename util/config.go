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

type DataBaseConfig struct {
	Host    string `json:"host" env_var:"DB_HOST"`
	Port    uint   `json:"port" env_var:"DB_PORT"`
	User    string `json:"user" env_var:"DB_USER"`
	Passwd  string `json:"passwd" env_var:"DB_PASSWD"`
	Name    string `json:"name" env_var:"DB_NAME"`
	Timeout int64  `json:"timeout" env_var:"DB_TIMEOUT"`
}

type ModuleFileHandlerConfig struct {
	WorkdirPath string `json:"workdir_path" env_var:"MFH_WORKDIR_PATH"`
	Delimiter   string `json:"delimiter" env_var:"MFH_DELIMITER"`
}

type Config struct {
	ServerPort        uint                    `json:"server_port" env_var:"SERVER_PORT"`
	ModuleFileHandler ModuleFileHandlerConfig `json:"module_file_handler" env_var:"MFH_CONFIG"`
	Logger            srv_base.LoggerConfig   `json:"logger" env_var:"LOGGER_CONFIG"`
	ConfigDefsPath    string                  `json:"config_defs_path" env_var:"CONFIG_DEFS_PATH"`
	DB                DataBaseConfig          `json:"db" env_var:"DB_CONFIG"`
	CewBaseUrl        string                  `json:"cew_base_url" env_var:"CEW_BASE_URL"`
}

func NewConfig(path *string) (*Config, error) {
	cfg := Config{
		ModuleFileHandler: ModuleFileHandlerConfig{
			WorkdirPath: "/opt/manager",
			Delimiter:   "_",
		},
		Logger: srv_base.LoggerConfig{
			Level:        level.Warning,
			Utc:          true,
			Microseconds: true,
			Terminal:     true,
		},
		ConfigDefsPath: "include/config_definitions.json",
		DB: DataBaseConfig{
			Port:    3306,
			Name:    "module_manager",
			Timeout: 5000000000,
		},
		CewBaseUrl: "http://api-gateway/cew",
	}
	err := srv_base.LoadConfig(path, &cfg, nil, nil, nil)
	return &cfg, err
}
