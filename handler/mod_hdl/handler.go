/*
 * Copyright 2022 InfAI (CC SES)
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

package mod_hdl

import (
	"context"
	"errors"
	"fmt"
	cew_lib "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-go-service-base/context-hdl"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"github.com/google/uuid"
	"io/fs"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

type Handler struct {
	storageHandler StorageHandler
	modFileHandler ModFileHandler
	cewClient      cew_lib.Api
	dbTimeout      time.Duration
	httpTimeout    time.Duration
	wrkSpcPath     string
	mu             sync.RWMutex
}

func New(storageHandler StorageHandler, modFileHandler ModFileHandler, cewClient cew_lib.Api, dbTimeout, httpTimeout time.Duration, workspacePath string) *Handler {
	return &Handler{
		storageHandler: storageHandler,
		modFileHandler: modFileHandler,
		cewClient:      cewClient,
		dbTimeout:      dbTimeout,
		httpTimeout:    httpTimeout,
		wrkSpcPath:     workspacePath,
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
	return nil
}

func (h *Handler) List(ctx context.Context, filter lib_model.ModFilter, dependencyInfo bool) (map[string]model.Module, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	modMap, err := h.storageHandler.ListMod(ctxWt, model.ModFilter{IDs: filter.IDs}, dependencyInfo)
	if err != nil {
		return nil, err
	}
	modules := make(map[string]model.Module)
	for _, mod := range modMap {
		mod.Module.Module, err = h.readModule(mod.Dir, mod.ModFile)
		if err != nil {
			util.Logger.Error(err)
			continue
		}
		mod.Path = h.wrkSpcPath
		if filterMod(filter, mod.Module.Module) {
			modules[mod.ID] = mod
		}
	}
	return modules, nil
}

func (h *Handler) Get(ctx context.Context, mID string, dependencyInfo bool) (model.Module, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	mod, err := h.storageHandler.ReadMod(ctxWt, mID, dependencyInfo)
	if err != nil {
		return model.Module{}, err
	}
	mod.Module.Module, err = h.readModule(mod.Dir, mod.ModFile)
	if err != nil {
		return model.Module{}, lib_model.NewInternalError(err)
	}
	mod.Path = h.wrkSpcPath
	return mod, nil
}

func (h *Handler) GetTree(ctx context.Context, mID string) (map[string]model.Module, error) {
	mod, err := h.storageHandler.ReadMod(ctx, mID, true)
	if err != nil {
		return nil, err
	}
	mod.Module.Module, err = h.readModule(mod.Dir, mod.ModFile)
	if err != nil {
		return nil, lib_model.NewInternalError(err)
	}
	mod.Path = h.wrkSpcPath
	tree := map[string]model.Module{mod.ID: mod}
	if err = h.appendModTree(ctx, mod, tree); err != nil {
		return nil, err
	}
	return tree, nil
}

func (h *Handler) AppendModTree(ctx context.Context, tree map[string]model.Module) error {
	for _, mod := range tree {
		if err := h.appendModTree(ctx, mod, tree); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) Add(ctx context.Context, mod *module.Module, modDir dir_fs.DirFS, modFile string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	dirName := newUUID.String()
	t := time.Now().UTC()
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	if err = h.storageHandler.CreateMod(ctxWt, tx, model.Module{
		Module: lib_model.Module{
			Module:  mod,
			Added:   t,
			Updated: t,
		},
		Dir:     dirName,
		ModFile: modFile,
	}); err != nil {
		return err
	}
	if len(mod.Dependencies) > 0 {
		dependencies := make([]string, 0, len(mod.Dependencies))
		for mID := range mod.Dependencies {
			dependencies = append(dependencies, mID)
		}
		ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
		defer cf2()
		if err = h.storageHandler.CreateModDependencies(ctxWt2, tx, mod.ID, dependencies); err != nil {
			return err
		}
	}
	dstPath := path.Join(h.wrkSpcPath, dirName)
	if err = dir_fs.Copy(modDir, dstPath); err != nil {
		return lib_model.NewInternalError(err)
	}
	if err = tx.Commit(); err != nil {
		if e := os.RemoveAll(dstPath); err != nil {
			util.Logger.Error(e)
		}
		return lib_model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) Delete(ctx context.Context, mID string, force bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	mod, err := h.storageHandler.ReadMod(ctxWt, mID, true)
	if err != nil {
		return err
	}
	mod.Module.Module, err = h.readModule(mod.Dir, mod.ModFile)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	if !force && len(mod.ModRequiring) > 0 {
		ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
		defer cf2()
		modules, err := h.storageHandler.ListMod(ctxWt2, model.ModFilter{IDs: mod.ModRequiring}, false)
		if err != nil {
			return err
		}
		var reqBy []string
		for id := range modules {
			reqBy = append(reqBy, id)
		}
		return lib_model.NewInternalError(fmt.Errorf("required by: %s", strings.Join(reqBy, ", ")))
	}
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, srv := range mod.Services {
		err = h.cewClient.RemoveImage(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), url.QueryEscape(url.QueryEscape(srv.Image)))
		if err != nil {
			var nfe *cew_model.NotFoundError
			if !errors.As(err, &nfe) {
				util.Logger.Error(err)
			}
		}
	}
	if err = os.RemoveAll(path.Join(h.wrkSpcPath, mod.Dir)); err != nil {
		return lib_model.NewInternalError(err)
	}
	return h.storageHandler.DeleteMod(ctx, nil, mID)
}

func (h *Handler) Update(ctx context.Context, mod *module.Module, modDir dir_fs.DirFS, modFile string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := context_hdl.New()
	defer ch.CancelAll()
	oldMod, err := h.storageHandler.ReadMod(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), mod.ID, false)
	if err != nil {
		return err
	}
	oldMod.Module.Module, err = h.readModule(oldMod.Dir, oldMod.ModFile)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	dirName := newUUID.String()
	t := time.Now().UTC()
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = h.storageHandler.DeleteModDependencies(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, mod.ID); err != nil {
		return err
	}
	if err = h.storageHandler.UpdateMod(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, model.Module{
		Module: lib_model.Module{
			Module:  mod,
			Added:   oldMod.Added,
			Updated: t,
		},
		Dir:     dirName,
		ModFile: modFile,
	}); err != nil {
		return err
	}
	if len(mod.Dependencies) > 0 {
		dependencies := make([]string, 0, len(mod.Dependencies))
		for mID := range mod.Dependencies {
			dependencies = append(dependencies, mID)
		}
		if err = h.storageHandler.CreateModDependencies(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, mod.ID, dependencies); err != nil {
			return err
		}
	}
	dstPath := path.Join(h.wrkSpcPath, dirName)
	if err = dir_fs.Copy(modDir, dstPath); err != nil {
		return lib_model.NewInternalError(err)
	}
	if err = tx.Commit(); err != nil {
		if e := os.RemoveAll(dstPath); err != nil {
			util.Logger.Error(e)
		}
		return lib_model.NewInternalError(err)
	}
	if e := os.RemoveAll(path.Join(h.wrkSpcPath, oldMod.Dir)); e != nil {
		util.Logger.Error(e)
	}
	images := make(map[string]struct{})
	for _, srv := range mod.Services {
		images[srv.Image] = struct{}{}
	}
	for _, srv := range oldMod.Services {
		if _, ok := images[srv.Image]; !ok {
			err = h.cewClient.RemoveImage(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), url.QueryEscape(url.QueryEscape(srv.Image)))
			if err != nil {
				var nfe *cew_model.NotFoundError
				if !errors.As(err, &nfe) {
					util.Logger.Error(err)
				}
			}
		}
	}
	return nil
}

func (h *Handler) appendModTree(ctx context.Context, mod model.Module, tree map[string]model.Module) error {
	for _, mID := range mod.RequiredMod {
		if _, ok := tree[mID]; !ok {
			m, err := h.storageHandler.ReadMod(ctx, mID, true)
			if err != nil {
				return err
			}
			m.Module.Module, err = h.readModule(m.Dir, m.ModFile)
			if err != nil {
				return err
			}
			m.Path = h.wrkSpcPath
			tree[mID] = m
			if err = h.appendModTree(ctx, m, tree); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) readModule(dir, modFile string) (*module.Module, error) {
	f, err := os.Open(path.Join(h.wrkSpcPath, dir, modFile))
	if err != nil {
		return nil, err
	}
	m, err := h.modFileHandler.GetModule(f)
	if err != nil {
		return nil, err
	}
	return m, nil
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
	return true
}
