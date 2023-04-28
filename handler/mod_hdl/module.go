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
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-lib/validation"
	"github.com/SENERGY-Platform/mgw-module-lib/validation/sem_ver"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"os"
	"sort"
	"strings"
)

type Handler struct {
	storageHandler          handler.ModStorageHandler
	transferHandler         handler.ModTransferHandler
	modFileHandler          handler.ModFileHandler
	configValidationHandler handler.CfgValidationHandler
}

func New(storageHandler handler.ModStorageHandler, transferHandler handler.ModTransferHandler, modFileHandler handler.ModFileHandler, configValidationHandler handler.CfgValidationHandler) *Handler {
	return &Handler{
		storageHandler:          storageHandler,
		transferHandler:         transferHandler,
		modFileHandler:          modFileHandler,
		configValidationHandler: configValidationHandler,
	}
}

func (h *Handler) List(ctx context.Context, filter model.ModFilter) ([]model.ModuleMeta, error) {
	return h.storageHandler.List(ctx, filter)
}

func (h *Handler) Get(ctx context.Context, mID string) (*module.Module, error) {
	return h.storageHandler.Get(ctx, mID)
}

func (h *Handler) GetWithDep(ctx context.Context, mID string) (*module.Module, map[string]*module.Module, error) {
	m, err := h.storageHandler.Get(ctx, mID)
	if err != nil {
		return nil, nil, err
	}
	dep := make(map[string]*module.Module)
	if err := h.getReqMod(ctx, m, dep); err != nil {
		return nil, nil, err
	}
	return m, dep, nil
}

func (h *Handler) Add(ctx context.Context, mID, version string) error {
	var sub string
	var ver string
	if version != "" {
		i := strings.LastIndex(version, "/")
		if i > 0 {
			sub = version[:i]
			ver = version[i+1:]
		} else {
			ver = version
		}
	} else {
		verList, err := h.transferHandler.ListVersions(ctx, mID)
		if err != nil {
			return err
		}
		if len(verList) == 0 {
			return model.NewInternalError(errors.New("no versions available"))
		}
		sort.Strings(verList)
		ver = verList[len(verList)-1]
	}
	if !sem_ver.IsValidSemVer(ver) {
		return model.NewInvalidInputError(fmt.Errorf("version '%s' invalid", version))
	}
	dir, err := h.transferHandler.Get(ctx, mID, ver, sub)
	if err != nil {
		return err
	}
	if err = h.validateModule(dir, mID); err != nil {
		os.RemoveAll(dir.Path())
		return err
	}
	return h.storageHandler.Add(ctx, dir, mID)
}

func (h *Handler) Delete(ctx context.Context, mID string) error {
	return h.storageHandler.Delete(ctx, mID)
}

func (h *Handler) Update(ctx context.Context, mID string) error {
	panic("not implemented")
}

func (h *Handler) CreateInclDir(ctx context.Context, mID, iID string) (string, error) {
	dir, err := h.storageHandler.MakeInclDir(ctx, mID, iID)
	if err != nil {
		return "", err
	}
	return dir.Path(), nil
}

func (h *Handler) DeleteInclDir(ctx context.Context, iID string) error {
	return h.storageHandler.RemoveInclDir(ctx, iID)
}

func (h *Handler) getReqMod(ctx context.Context, mod *module.Module, reqMod map[string]*module.Module) error {
	for mID := range mod.Dependencies {
		if _, ok := reqMod[mID]; !ok {
			m, err := h.storageHandler.Get(ctx, mID)
			if err != nil {
				return err
			}
			reqMod[mID] = m
			if err = h.getReqMod(ctx, m, reqMod); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) validateModule(dir util.DirFS, mID string) error {
	m, err := h.modFileHandler.GetModule(dir)
	if err != nil {
		return err
	}
	if mID != m.ID {
		return fmt.Errorf("module ID missmatch: %s != %s", mID, m.ID)
	}
	err = validation.Validate(m)
	if err != nil {
		return err
	}
	for _, cv := range m.Configs {
		if err := h.configValidationHandler.ValidateBase(cv.Type, cv.TypeOpt, cv.DataType); err != nil {
			return err
		}
		if err := h.configValidationHandler.ValidateTypeOptions(cv.Type, cv.TypeOpt); err != nil {
			return err
		}
		if cv.Default != nil {
			if err := h.configValidationHandler.ValidateValue(cv.Type, cv.TypeOpt, cv.Default, cv.IsSlice, cv.DataType); err != nil {
				return err
			}
		}
	}
	return nil
}
