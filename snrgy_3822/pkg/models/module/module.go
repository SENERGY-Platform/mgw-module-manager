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

package module

import (
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	"time"
)

type ModuleBase struct {
	Source  string    `json:"source"`
	Channel string    `json:"channel"`
	Added   time.Time `json:"added"`
	Updated time.Time `json:"updated"`
}

type ModuleAbbreviated struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Desc    string `json:"description"`
	Version string `json:"version"`
	ModuleBase
}

type Module struct {
	module_lib.Module
	ModuleBase
}

type ModuleFilter struct {
	IDs []string
}
