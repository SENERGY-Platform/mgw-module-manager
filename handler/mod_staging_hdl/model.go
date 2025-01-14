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
	cew_lib "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"os"
	"time"
)

type stage struct {
	modules     map[string]*module.Module
	items       map[string]modExtra
	path        string
	cewClient   cew_lib.Api
	httpTimeout time.Duration
}

type item struct {
	module *module.Module
	modExtra
}

type modExtra struct {
	modFile  string
	dir      dir_fs.DirFS
	indirect bool
}

func (s *stage) Get(mID string) (model.StageItem, bool) {
	extra, ok := s.items[mID]
	return &item{module: s.modules[mID], modExtra: extra}, ok
}

func (s *stage) Items() map[string]model.StageItem {
	items := make(map[string]model.StageItem)
	for mID, extra := range s.items {
		items[mID] = &item{
			module:   s.modules[mID],
			modExtra: extra,
		}
	}
	return items
}

func (s *stage) addItem(mod *module.Module, modFile string, dir dir_fs.DirFS, indirect bool) {
	s.items[mod.ID] = modExtra{
		modFile:  modFile,
		dir:      dir,
		indirect: indirect,
	}
	s.addMod(mod)
}

func (s *stage) addMod(mod *module.Module) {
	s.modules[mod.ID] = mod
}

func (s *stage) Remove() error {
	return os.RemoveAll(s.path)
}

func (i *item) Module() *module.Module {
	return i.module
}

func (i *item) ModFile() string {
	return i.modFile
}

func (i *item) Dir() dir_fs.DirFS {
	return i.dir
}

func (i *item) Indirect() bool {
	return i.indirect
}
