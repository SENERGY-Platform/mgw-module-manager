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

type Config struct {
	ServerPort int                   `json:"server_port" env_var:"SERVER_PORT"`
	WorkDir    string                `json:"work_dir" env_var:"WORK_DIR"`
	Logger     srv_base.LoggerConfig `json:"logger" env_var:"LOGGER_CONFIG"`
}

func NewConfig(path *string) (*Config, error) {
	cfg := Config{
		Logger: srv_base.LoggerConfig{
			Level:        level.Warning,
			Utc:          true,
			Path:         "/var/log/",
			FileName:     "mgw-deployment-manager",
			Microseconds: true,
		},
	}
	err := srv_base.LoadConfig(path, &cfg, nil, nil, nil)
	return &cfg, err
}
