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
	"github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-lib/validation"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"io/fs"
	"net/url"
	"strings"
	"time"
)

type Handler struct {
	storageHandler          handler.ModStorageHandler
	transferHandler         handler.ModTransferHandler
	modFileHandler          handler.ModFileHandler
	configValidationHandler handler.CfgValidationHandler
	cewJobHandler           handler.CewJobHandler
	cewClient               client.CewClient
	httpTimeout             time.Duration
}

func New(storageHandler handler.ModStorageHandler, transferHandler handler.ModTransferHandler, modFileHandler handler.ModFileHandler, configValidationHandler handler.CfgValidationHandler, cewJobHandler handler.CewJobHandler, cewClient client.CewClient, httpTimeout time.Duration) *Handler {
	return &Handler{
		storageHandler:          storageHandler,
		transferHandler:         transferHandler,
		modFileHandler:          modFileHandler,
		configValidationHandler: configValidationHandler,
		cewJobHandler:           cewJobHandler,
		cewClient:               cewClient,
		httpTimeout:             httpTimeout,
	}
}

func (h *Handler) List(ctx context.Context, filter model.ModFilter) ([]model.ModuleMeta, error) {
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

func (h *Handler) GetIncl(ctx context.Context, mID string) (dir_fs.DirFS, error) {
	mod, dir, err := h.storageHandler.GetDir(ctx, mID)
	if err != nil {
		return "", err
	}
	for _, srv := range mod.Services {
		for _, bindMount := range srv.BindMounts {
			_, err = fs.Stat(dir, bindMount.Source)
			if err != nil {
				return "", err
			}
		}
	}
	return dir, nil
}

func (h *Handler) Delete(ctx context.Context, mID string) error {
	l, err := h.storageHandler.List(ctx, model.ModFilter{InDependencies: map[string]struct{}{mID: {}}})
	if err != nil {
		return err
	}
	if len(l) > 0 {
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

func (h *Handler) Update(ctx context.Context, mID string) error {
	panic("not implemented")
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

func (h *Handler) validateModule(m *module.Module, mID, ver string) error {
	if mID != m.ID {
		return fmt.Errorf("module ID mismatch: %s != %s", mID, m.ID)
	}
	if ver != m.Version {
		return fmt.Errorf("version mismatch: %s != %s", ver, m.Version)
	}
	err := validation.Validate(m)
	if err != nil {
		return err
	}
	for _, cv := range m.Configs {
		if err = h.configValidationHandler.ValidateBase(cv.Type, cv.TypeOpt, cv.DataType); err != nil {
			return err
		}
		if err = h.configValidationHandler.ValidateTypeOptions(cv.Type, cv.TypeOpt); err != nil {
			return err
		}
		if cv.Default != nil {
			if err = h.configValidationHandler.ValidateValue(cv.Type, cv.TypeOpt, cv.Default, cv.IsSlice, cv.DataType); err != nil {
				return err
			}
		}
	}
	return nil
}
