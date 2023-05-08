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
	"io/fs"
	"os"
	"path"
	"strings"
)

type Handler struct {
	wrkSpcPath     string
	delimiter      string
	perm           fs.FileMode
	modFileHandler handler.ModFileHandler
	indexHandler   *indexHandler
}

func New(workspacePath string, delimiter string, perm fs.FileMode, modFileHandler handler.ModFileHandler) (*Handler, error) {
	if !path.IsAbs(workspacePath) {
		return nil, fmt.Errorf("workspace path must be absolute")
	}
	return &Handler{
		wrkSpcPath:     workspacePath,
		delimiter:      delimiter,
		perm:           perm,
		modFileHandler: modFileHandler,
		indexHandler:   newIndexHandler(workspacePath),
	}, nil
}

func (h *Handler) InitWorkspace() error {
	if err := os.MkdirAll(h.wrkSpcPath, h.perm); err != nil {
		return err
	}
	return h.indexHandler.Init()
}

func (h *Handler) List(ctx context.Context, filter model.ModFilter) ([]model.ModuleMeta, error) {
	items := h.indexHandler.List()
	var mm []model.ModuleMeta
	for _, i := range items {
		dir, err := util.NewDirFS(path.Join(h.wrkSpcPath, i.Dir))
		if err != nil {
			return nil, err
		}
		m, err := h.modFileHandler.GetModule(dir)
		if err != nil {
			util.Logger.Error(err)
			continue
		}
		if filterMod(filter, m) {
			mm = append(mm, getModMeta(m, i))
		}
		if ctx.Err() != nil {
			return nil, model.NewInternalError(ctx.Err())
		}
	}
	return mm, nil
}

func (h *Handler) Get(ctx context.Context, mID string) (model.Module, error) {
	m, _, err := h.GetDir(ctx, mID)
	if err != nil {
		return model.Module{}, err
	}
	return m, nil
}

func (h *Handler) GetDir(_ context.Context, mID string) (model.Module, util.DirFS, error) {
	i, err := h.indexHandler.Get(mID)
	if err != nil {
		return model.Module{}, "", err
	}
	dir, err := util.NewDirFS(path.Join(h.wrkSpcPath, i.Dir))
	if err != nil {
		return model.Module{}, "", model.NewInternalError(err)
	}
	m, err := h.modFileHandler.GetModule(dir)
	if err != nil {
		return model.Module{}, "", model.NewInternalError(err)
	}
	return m, dir, nil
}

func (h *Handler) Add(_ context.Context, dir util.DirFS, mID string, indirect bool) error {
	err := h.indexHandler.Add(mID, idToDir(mID, h.delimiter), indirect)
	if err != nil {
		return err
	}
	err = util.CopyDir(dir.Path(), path.Join(h.wrkSpcPath, idToDir(mID, h.delimiter)))
	if err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) Delete(_ context.Context, mID string) error {
	i, err := h.indexHandler.Get(mID)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(path.Join(h.wrkSpcPath, i.Dir)); err != nil {
		return model.NewInternalError(err)
	}
	return h.indexHandler.Delete(mID)
}

func idToDir(id string, delimiter string) string {
	return strings.Replace(id, "/", delimiter, -1)
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

func getModMeta(m *module.Module, i item) model.ModuleMeta {
	return model.ModuleMeta{
		ID:             m.ID,
		Name:           m.Name,
		Description:    m.Description,
		Tags:           m.Tags,
		License:        m.License,
		Author:         m.Author,
		Version:        m.Version,
		Type:           m.Type,
		DeploymentType: m.DeploymentType,
		ModuleExtra:    getModExtra(i),
	}
}

func getModExtra(i item) model.ModuleExtra {
	return model.ModuleExtra{
		Indirect: i.Indirect,
		Added:    i.Added,
		Updated:  i.Updated,
	}
}
