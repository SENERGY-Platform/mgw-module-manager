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
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"github.com/google/uuid"
	"io/fs"
	"os"
	"path"
)

type Handler struct {
	wrkSpcPath     string
	modFileHandler handler.ModFileHandler
	indexHandler   *indexHandler
	modules        map[string]model.Module
}

func New(workspacePath string, modFileHandler handler.ModFileHandler) *Handler {
	return &Handler{
		wrkSpcPath:     workspacePath,
		modFileHandler: modFileHandler,
		indexHandler:   newIndexHandler(workspacePath),
	}
}

func (h *Handler) Init(perm fs.FileMode) error {
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

func (h *Handler) List(ctx context.Context, filter model.ModFilter) ([]model.Module, error) {
	var mm []model.Module
	for _, m := range h.modules {
		if filterMod(filter, m.Module) {
			mm = append(mm, m)
		}
		if ctx.Err() != nil {
			return nil, model.NewInternalError(ctx.Err())
		}
	}
	return mm, nil
}

func (h *Handler) Get(_ context.Context, mID string) (model.Module, error) {
	m, ok := h.modules[mID]
	if !ok {
		return model.Module{}, model.NewNotFoundError(fmt.Errorf("module '%s' not found", mID))
	}
	return m, nil
}

func (h *Handler) GetDir(_ context.Context, mID string) (model.Module, dir_fs.DirFS, error) {
	m, ok := h.modules[mID]
	if !ok {
		return model.Module{}, "", model.NewNotFoundError(fmt.Errorf("module '%s' not found", mID))
	}
	i, err := h.indexHandler.Get(m.ID)
	if err != nil {
		return model.Module{}, "", err
	}
	dir, err := dir_fs.New(path.Join(h.wrkSpcPath, i.Dir))
	if err != nil {
		return model.Module{}, "", model.NewInternalError(err)
	}
	return m, dir, nil
}

func (h *Handler) Add(_ context.Context, mod model.Module, modDir dir_fs.DirFS, modFile string) error {
	dirName := uuid.NewString()
	err := h.indexHandler.Add(mod.ID, dirName, modFile, mod.Indirect, mod.Added)
	if err != nil {
		return err
	}
	err = util.CopyDir(modDir.Path(), path.Join(h.wrkSpcPath, dirName))
	if err != nil {
		return model.NewInternalError(err)
	}
	h.modules[mod.ID] = mod
	return nil
}

func (h *Handler) Delete(_ context.Context, mID string) error {
	i, err := h.indexHandler.Get(mID)
	if err != nil {
		return err
	}
	if err = os.RemoveAll(path.Join(h.wrkSpcPath, i.Dir)); err != nil {
		return model.NewInternalError(err)
	}
	if err = h.indexHandler.Delete(mID); err != nil {
		return model.NewInternalError(err)
	}
	delete(h.modules, mID)
	return nil
}

func (h *Handler) loadModules() {
	items := h.indexHandler.List()
	h.modules = make(map[string]model.Module)
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
		h.modules[m.ID] = model.Module{
			Module:      m,
			ModuleExtra: getModExtra(i),
		}
	}
	return
}

func filterMod(filter model.ModFilter, m *module.Module) bool {
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

func getModExtra(i item) model.ModuleExtra {
	return model.ModuleExtra{
		Indirect: i.Indirect,
		Added:    i.Added,
		Updated:  i.Updated,
	}
}
