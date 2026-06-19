/*
 * Copyright 2026 InfAI (CC SES)
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

package models

import (
	"io/fs"
	"time"

	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

type Module struct {
	external_models.ModuleLibModule
	Source     string
	Channel    string
	Added      time.Time
	Updated    time.Time
	Files      map[string]ModuleFile
	FileSystem fs.FS
}

type ModuleFile struct {
	external_models.ModuleLibFile
	DefaultData []byte
}

type ModulesFilterWithName struct {
	ModulesFilter
	Name string
}

type DatabaseModule struct {
	Id      string
	DirName string
	Source  string
	Channel string
	Added   time.Time
	Updated time.Time
}

type ModulesFilter struct {
	Ids     []string
	Source  string
	Channel string
}
