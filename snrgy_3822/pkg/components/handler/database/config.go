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

package database

import "time"

type Config struct {
	Address  string        `json:"address" env_var:"DATABASE_ADDRESS"`
	Database string        `json:"database" env_var:"DATABASE_NAME"`
	User     string        `json:"user" env_var:"DATABASE_USER"`
	Password string        `json:"password" env_var:"DATABASE_PASSWORD"`
	Timeout  time.Duration `json:"timeout" env_var:"DATABASE_TIMEOUT"`
}
