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
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"net/url"
	"strings"
	"time"
)

type Handler struct {
	storageHandler handler.ModStorageHandler
	cewClient      cew_lib.Api
	httpTimeout    time.Duration
}

func New(storageHandler handler.ModStorageHandler, cewClient cew_lib.Api, httpTimeout time.Duration) *Handler {
	return &Handler{
		storageHandler: storageHandler,
		cewClient:      cewClient,
		httpTimeout:    httpTimeout,
	}
}

func (h *Handler) List(ctx context.Context, filter model.ModFilter) ([]model.Module, error) {
	return h.storageHandler.List(ctx, filter)
}

func (h *Handler) Get(ctx context.Context, mID string) (model.Module, error) {
	return h.storageHandler.Get(ctx, mID)
}

func (h *Handler) GetReq(ctx context.Context, mID string) (model.Module, map[string]model.Module, error) {
	m, err := h.storageHandler.Get(ctx, mID)
	if err != nil {
		return model.Module{}, nil, err
	}
	dep := make(map[string]model.Module)
	if err := h.getReqMod(ctx, m.Module, dep); err != nil {
		return model.Module{}, nil, err
	}
	return m, dep, nil
}

func (h *Handler) GetDir(ctx context.Context, mID string) (dir_fs.DirFS, error) {
	return h.storageHandler.GetDir(ctx, mID)
}

func (h *Handler) Add(ctx context.Context, mod *module.Module, modDir dir_fs.DirFS, modFile string, indirect bool) error {
	t := time.Now().UTC()
	m := model.Module{
		Module: mod,
		ModuleExtra: model.ModuleExtra{
			Indirect: indirect,
			Added:    t,
			Updated:  t,
		},
	}
	err := h.storageHandler.Add(ctx, m, modDir, modFile)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) Delete(ctx context.Context, mID string, force bool) error {
	l, err := h.storageHandler.List(ctx, model.ModFilter{InDependencies: map[string]struct{}{mID: {}}})
	if err != nil {
		return err
	}
	if len(l) > 0 && !force {
		var ids []string
		for _, meta := range l {
			ids = append(ids, meta.ID)
		}
		return model.NewInternalError(fmt.Errorf("module is required by: %s", strings.Join(ids, ", ")))
	}
	mod, err := h.storageHandler.Get(ctx, mID)
	if err != nil {
		return err
	}
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, srv := range mod.Services {
		err = h.cewClient.RemoveImage(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), url.QueryEscape(srv.Image))
		if err != nil {
			var nfe *cew_model.NotFoundError
			if !errors.As(err, &nfe) {
				return model.NewInternalError(err)
			}
		}
	}
	return h.storageHandler.Delete(ctx, mID)
}

func (h *Handler) Update(ctx context.Context, mod *module.Module, modDir dir_fs.DirFS, modFile string, indirect bool) error {
	m, err := h.storageHandler.Get(ctx, mod.ID)
	if err != nil {
		return err
	}
	t := time.Now().UTC()
	if m.Version == mod.Version {
		if m.Indirect != indirect {
			m.Indirect = indirect
			m.Updated = t
			return h.storageHandler.Update(ctx, m, "", "")
		}
		return nil
	}
	m.Module = mod
	m.Indirect = indirect
	m.Updated = t
	return h.storageHandler.Update(ctx, m, modDir, modFile)
}

func (h *Handler) getReqMod(ctx context.Context, mod *module.Module, reqMod map[string]model.Module) error {
	for mID := range mod.Dependencies {
		if _, ok := reqMod[mID]; !ok {
			m, err := h.storageHandler.Get(ctx, mID)
			if err != nil {
				return err
			}
			reqMod[mID] = m
			if err = h.getReqMod(ctx, m.Module, reqMod); err != nil {
				return err
			}
		}
	}
	return nil
}
