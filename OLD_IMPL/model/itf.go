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

package model

import (
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
)

type ModRepo interface {
	Versions() []string
	Get(ver string) (dir_fs.DirFS, error)
	Remove() error
}

type StageItem interface {
	Module() *module_lib.Module
	ModFile() string
	Dir() dir_fs.DirFS
	Indirect() bool
}

type Stage interface {
	Items() map[string]StageItem
	Get(mID string) (StageItem, bool)
	Remove() error
}
