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

package modules

import "time"

type Config struct {
	WorkDirPath     string        `json:"work_dir_path" env_var:"MODULES_HANDLER_WORK_DIR_PATH"`
	JobPollInterval time.Duration `json:"job_poll_interval" env_var:"MODULES_HANDLER_JOB_POLL_INTERVAL"`
	PathEscapeDepth int           `json:"path_escape_depth" env_var:"PATH_ESCAPE_DEPTH"`
}
