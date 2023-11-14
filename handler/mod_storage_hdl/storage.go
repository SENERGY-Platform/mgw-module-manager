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

package mod_storage_hdl

import (
	"context"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"github.com/google/uuid"
	"io/fs"
	"os"
	"path"
	"sync"
)

type Handler struct {
	wrkSpcPath     string
	modFileHandler handler.ModFileHandler
	indexHandler   *indexHandler
	modules        map[string]lib_model.Module
	mu             sync.RWMutex
}

func New(workspacePath string, modFileHandler handler.ModFileHandler) *Handler {
	return &Handler{
		wrkSpcPath:     workspacePath,
		modFileHandler: modFileHandler,
		indexHandler:   newIndexHandler(workspacePath),
	}
}

func (h *Handler) Init(perm fs.FileMode) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !path.IsAbs(h.wrkSpcPath) {
		return fmt.Errorf("workspace path must be absolute")
	}
	if err := os.MkdirAll(h.wrkSpcPath, perm); err != nil {
		return err
	}
	if err := h.indexHandler.Init(); err != nil {
		return err
	}
	h.loadModules()
	return nil
}

func (h *Handler) List(ctx context.Context, filter lib_model.ModFilter) (map[string]lib_model.Module, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	mm := make(map[string]lib_model.Module)
	for id, m := range h.modules {
		if filterMod(filter, m.Module) {
			mm[id] = m
		}
		if ctx.Err() != nil {
			return nil, lib_model.NewInternalError(ctx.Err())
		}
	}
	return mm, nil
}

func (h *Handler) Get(_ context.Context, mID string) (lib_model.Module, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	m, ok := h.modules[mID]
	if !ok {
		return lib_model.Module{}, lib_model.NewNotFoundError(fmt.Errorf("module '%s' not found", mID))
	}
	return m, nil
}

func (h *Handler) GetDir(_ context.Context, mID string) (dir_fs.DirFS, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	i, err := h.indexHandler.Get(mID)
	if err != nil {
		return "", err
	}
	dir, err := dir_fs.New(path.Join(h.wrkSpcPath, i.Dir))
	if err != nil {
		return "", lib_model.NewInternalError(err)
	}
	return dir, nil
}

func (h *Handler) Add(_ context.Context, mod lib_model.Module, modDir dir_fs.DirFS, modFile string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	dirName := uuid.NewString()
	err := h.indexHandler.Add(mod.ID, dirName, modFile, mod.Indirect, mod.Added)
	if err != nil {
		return err
	}
	err = dir_fs.Copy(modDir, path.Join(h.wrkSpcPath, dirName))
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	h.modules[mod.ID] = mod
	return nil
}

func (h *Handler) Update(_ context.Context, mod lib_model.Module, modDir dir_fs.DirFS, modFile string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	i, err := h.indexHandler.Get(mod.ID)
	if err != nil {
		return err
	}
	if modDir == "" || modFile == "" {
		return h.indexHandler.Update(i.ID, i.Dir, i.ModFile, mod.Indirect, mod.Updated)
	}
	dirName := uuid.NewString()
	err = dir_fs.Copy(modDir, path.Join(h.wrkSpcPath, dirName))
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	err = h.indexHandler.Update(i.ID, dirName, modFile, mod.Indirect, mod.Updated)
	if err != nil {
		return err
	}
	h.modules[mod.ID] = mod
	if err = os.RemoveAll(path.Join(h.wrkSpcPath, i.Dir)); err != nil {
		return lib_model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) Delete(_ context.Context, mID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	i, err := h.indexHandler.Get(mID)
	if err != nil {
		return err
	}
	if err = os.RemoveAll(path.Join(h.wrkSpcPath, i.Dir)); err != nil {
		return lib_model.NewInternalError(err)
	}
	if err = h.indexHandler.Delete(mID); err != nil {
		return lib_model.NewInternalError(err)
	}
	delete(h.modules, mID)
	return nil
}

func (h *Handler) loadModules() {
	items := h.indexHandler.List()
	h.modules = make(map[string]lib_model.Module)
	for _, i := range items {
		f, err := os.Open(path.Join(h.wrkSpcPath, i.Dir, i.ModFile))
		if err != nil {
			util.Logger.Error(err)
			continue
		}
		m, err := h.modFileHandler.GetModule(f)
		if err != nil {
			util.Logger.Error(err)
			continue
		}
		h.modules[m.ID] = lib_model.Module{
			Module:      m,
			ModuleExtra: getModExtra(i),
		}
	}
	return
}

func filterMod(filter lib_model.ModFilter, m *module.Module) bool {
	if filter.Name != "" {
		if m.Name != filter.Name {
			return false
		}
	}
	if filter.Type != "" {
		if m.Type != filter.Type {
			return false
		}
	}
	if filter.DeploymentType != "" {
		if m.DeploymentType != filter.DeploymentType {
			return false
		}
	}
	if filter.Author != "" {
		if m.Author != filter.Author {
			return false
		}
	}
	if len(filter.Tags) > 0 {
		var ok bool
		for tag := range filter.Tags {
			if _, ok = m.Tags[tag]; ok {
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(filter.InDependencies) > 0 {
		var ok bool
		for id := range filter.InDependencies {
			if _, ok = m.Dependencies[id]; ok {
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func getModExtra(i item) lib_model.ModuleExtra {
	return lib_model.ModuleExtra{
		Indirect: i.Indirect,
		Added:    i.Added,
		Updated:  i.Updated,
	}
}
