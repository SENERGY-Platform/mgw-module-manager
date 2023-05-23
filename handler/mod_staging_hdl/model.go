/*
 * Copyright 2023 InfAI (CC SES)
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

package mod_staging_hdl

import (
	"github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"os"
	"time"
)

type stage struct {
	items       map[string]handler.StageItem
	path        string
	cewClient   client.CewClient
	httpTimeout time.Duration
}

type item struct {
	module   *module.Module
	modFile  string
	dir      dir_fs.DirFS
	indirect bool
}

func (s stage) Item(mID string) handler.StageItem {
	return s.items[mID]
}

func (s stage) Items() map[string]handler.StageItem {
	return s.items
}

func (s stage) Remove() error {
	return os.RemoveAll(s.path)
}

func (i item) Module() *module.Module {
	return i.module
}

func (i item) ModFile() string {
	return i.modFile
}

func (i item) Dir() dir_fs.DirFS {
	return i.dir
}

func (i item) Indirect() bool {
	return i.indirect
}
