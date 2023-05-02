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

const (
	modDir  = "modules"
	inclDir = "deployments"
)

type Handler struct {
	modWrkSpcPath     string
	inclDirWrkSpcPath string
	delimiter         string
	perm              fs.FileMode
	modFileHandler    handler.ModFileHandler
}

func New(workspacePath string, delimiter string, perm fs.FileMode, modFileHandler handler.ModFileHandler) (*Handler, error) {
	if !path.IsAbs(workspacePath) {
		return nil, fmt.Errorf("workspace path must be absolute")
	}
	return &Handler{
		modWrkSpcPath:     path.Join(workspacePath, modDir),
		inclDirWrkSpcPath: path.Join(workspacePath, inclDir),
		delimiter:         delimiter,
		perm:              perm,
		modFileHandler:    modFileHandler,
	}, nil
}

func (h *Handler) InitWorkspace() error {
	if err := os.MkdirAll(h.modWrkSpcPath, h.perm); err != nil {
		return err
	}
	if err := os.MkdirAll(h.inclDirWrkSpcPath, h.perm); err != nil {
		return err
	}
	return nil
}

func (h *Handler) List(ctx context.Context, filter model.ModFilter) ([]model.ModuleMeta, error) {
	dir, err := util.NewDirFS(h.modWrkSpcPath)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	dirEntries, err := fs.ReadDir(dir, ".")
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	var mm []model.ModuleMeta
	for _, entry := range dirEntries {
		if entry.IsDir() {
			sub, err := dir.Sub(entry.Name())
			if err != nil {
				return nil, err
			}
			m, err := h.modFileHandler.GetModule(sub)
			if err != nil {
				continue
			}
			if filterMod(filter, m) {
				mm = append(mm, model.ModuleMeta{
					ID:             m.ID,
					Name:           m.Name,
					Description:    m.Description,
					Tags:           m.Tags,
					License:        m.License,
					Author:         m.Author,
					Version:        m.Version,
					Type:           m.Type,
					DeploymentType: m.DeploymentType,
				})
			}
		}
		if ctx.Err() != nil {
			return nil, model.NewInternalError(err)
		}
	}
	return mm, nil
}

func (h *Handler) Get(_ context.Context, mID string) (*module.Module, error) {
	dir, err := util.NewDirFS(path.Join(h.modWrkSpcPath, idToDir(mID, h.delimiter)))
	if err != nil {
		return nil, model.NewNotFoundError(err)
	}
	m, err := h.modFileHandler.GetModule(dir)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (h *Handler) Add(_ context.Context, dir util.DirFS, mID string) error {
	defer os.RemoveAll(dir.Path())
	err := util.CopyDir(dir.Path(), path.Join(h.modWrkSpcPath, idToDir(mID, h.delimiter)))
	if err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) Delete(_ context.Context, mID string) error {
	if err := os.RemoveAll(path.Join(h.modWrkSpcPath, idToDir(mID, h.delimiter))); err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) MakeInclDir(_ context.Context, mID, iID string) (util.DirFS, error) {
	p := path.Join(h.inclDirWrkSpcPath, iID)
	if err := util.CopyDir(path.Join(h.modWrkSpcPath, idToDir(mID, h.delimiter)), p); err != nil {
		_ = os.RemoveAll(p)
		return "", model.NewInternalError(err)
	}
	dir, err := util.NewDirFS(p)
	if err != nil {
		return "", model.NewInternalError(err)
	}
	return dir, nil
}

func (h *Handler) GetInclDir(_ context.Context, iID string) (util.DirFS, error) {
	dir, err := util.NewDirFS(path.Join(h.inclDirWrkSpcPath, iID))
	if err != nil {
		return "", model.NewInternalError(err)
	}
	return dir, nil
}

func (h *Handler) RemoveInclDir(_ context.Context, iID string) error {
	if err := os.RemoveAll(path.Join(h.inclDirWrkSpcPath, iID)); err != nil {
		return model.NewInternalError(err)
	}
	return nil
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
