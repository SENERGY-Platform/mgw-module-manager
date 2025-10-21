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
	"time"

	sb_config_hdl "github.com/SENERGY-Platform/go-service-base/config-hdl"
	struct_logger "github.com/SENERGY-Platform/go-service-base/struct-logger"
	handler_database "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/database"
	handler_modules "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/modules"
	helper_sql_db "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/sql_db"
)

type MGWConfig struct {
	CewBaseUrl string        `json:"cew_base_url" env_var:"MGW_CEW_BASE_URL"`
	CmBaseUrl  string        `json:"cm_base_url" env_var:"MGW_CM_BASE_URL"`
	HmBaseUrl  string        `json:"hm_base_url" env_var:"MGW_HM_BASE_URL"`
	SmBaseUrl  string        `json:"sm_base_url" env_var:"MGW_SM_BASE_URL"`
	Timeout    time.Duration `json:"timeout" env_var:"MGW_HTTP_TIMEOUT"`
}

type GitHubModulesRepoHandlerConfig struct {
	BaseUrl     string        `json:"base_url" env_var:"GITHUB_BASE_URL"`
	Timeout     time.Duration `json:"timeout" env_var:"GITHUB_TIMEOUT"`
	WorkDirPath string        `json:"work_dir_path" env_var:"GITHUB_MODULES_REPO_HANDLER_WORK_DIR_PATH"`
}

type JobsConfig struct {
	BufferSize  int   `json:"buffer_size" env_var:"JOBS_BUFFER_SIZE"`
	MaxNumber   int   `json:"max_number" env_var:"JOBS_MAX_NUMBER"`
	CCHInterval int   `json:"cch_interval" env_var:"JOBS_CCH_INTERVAL"`
	JHInterval  int   `json:"jh_interval" env_var:"JOBS_JH_INTERVAL"`
	PJHInterval int64 `json:"pjh_interval" env_var:"JOBS_PJH_INTERVAL"`
	MaxAge      int64 `json:"max_age" env_var:"JOBS_MAX_AGE"`
}

type DatabaseConfig struct {
	MySQL handler_database.Config
	SQL   helper_sql_db.Config
}

type Config struct {
	ServerPort               uint                           `json:"server_port" env_var:"SERVER_PORT"`
	MGW                      MGWConfig                      `json:"mgw"`
	ModulesHandler           handler_modules.Config         `json:"modules_handler"`
	Logger                   struct_logger.Config           `json:"logger"`
	GitHubModulesRepoHandler GitHubModulesRepoHandlerConfig `json:"github_modules_repo_handler"`
	Jobs                     JobsConfig                     `json:"jobs"`
	Database                 DatabaseConfig                 `json:"database"`
	ManagerIDPath            string                         `json:"manager_id_path" env_var:"MANAGER_ID_PATH"`
	CoreID                   string                         `json:"core_id" env_var:"CORE_ID"`
	HttpAccessLog            bool                           `json:"http_access_log" env_var:"HTTP_ACCESS_LOG"`
	UseUTC                   bool                           `json:"use_utc" env_var:"USE_UTC"`
}

func New(path string) (*Config, error) {
	cfg := Config{
		ServerPort: 80,
		MGW: MGWConfig{
			CewBaseUrl: "http://core-api/ce-wrapper",
			CmBaseUrl:  "http://core-api/c-manager",
			HmBaseUrl:  "http://core-api/h-manager",
			SmBaseUrl:  "http://secret-manager",
			Timeout:    time.Second * 30,
		},
		ModulesHandler: handler_modules.Config{
			WorkDirPath:     "/opt/module-manager/modules",
			JobPollInterval: time.Millisecond * 500,
		},
		Logger: struct_logger.Config{
			Handler:    struct_logger.TextHandlerSelector,
			Level:      struct_logger.LevelInfo,
			TimeFormat: time.RFC3339Nano,
			TimeUtc:    true,
			AddMeta:    false,
		},
		GitHubModulesRepoHandler: GitHubModulesRepoHandlerConfig{
			BaseUrl: "https://api.github.com",
			Timeout: time.Minute,
		},
		Jobs: JobsConfig{
			BufferSize:  200,
			MaxNumber:   20,
			CCHInterval: 500000,
			JHInterval:  500000,
			PJHInterval: 300000000000,
			MaxAge:      172800000000000,
		},
		Database: DatabaseConfig{
			MySQL: handler_database.Config{
				Address:  "core-db:3306",
				Database: "module_manager",
				Timeout:  time.Second * 30,
			},
			SQL: helper_sql_db.Config{
				MaxOpenConns:    25,
				MaxIdleConns:    25,
				ConnMaxLifetime: time.Minute * 5,
			},
		},
		ManagerIDPath: "/opt/module-manager/data/mid",
		UseUTC:        true,
	}
	err := sb_config_hdl.Load(&cfg, nil, envTypeParser, nil, path)
	return &cfg, err
}
