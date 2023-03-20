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

package module

import (
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/go-service-base/srv-base"
	"github.com/SENERGY-Platform/go-service-base/srv-base/types"
	"github.com/SENERGY-Platform/mgw-modfile-lib/modfile"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-lib/validation"
	"gopkg.in/yaml.v3"
	"module-manager/manager/itf"
	"net/http"
	"os"
)

type Handler struct {
	storageHandler          itf.ModuleStorageHandler
	configValidationHandler itf.ConfigValidationHandler
	mfDecoders              modfile.Decoders
	mfGenerators            modfile.Generators
}

func NewHandler(storageHandler itf.ModuleStorageHandler, configValidationHandler itf.ConfigValidationHandler, mfDecoders modfile.Decoders, mfGenerators modfile.Generators) *Handler {
	return &Handler{
		storageHandler:          storageHandler,
		configValidationHandler: configValidationHandler,
		mfDecoders:              mfDecoders,
		mfGenerators:            mfGenerators,
	}
}

func (h *Handler) List() ([]*module.Module, error) {
	mIds, err := h.storageHandler.List()
	if err != nil {
		return nil, srv_base_types.NewError(http.StatusInternalServerError, "listing modules failed", err)
	}
	var modules []*module.Module
	for _, id := range mIds {
		file, err := h.storageHandler.Open(id)
		if err != nil {
			srv_base.Logger.Errorf("opening module '%s' failed: %s", id, err)
			continue
		}
		yd := yaml.NewDecoder(file)
		mf := modfile.New(h.mfDecoders, h.mfGenerators)
		if err = yd.Decode(&mf); err != nil {
			srv_base.Logger.Errorf("decoding modfile '%s' failed: %s", id, err)
			continue
		}
		m, err := mf.GetModule()
		if err != nil {
			srv_base.Logger.Errorf("getting module '%s' failed: %s", id, err)
			continue
		}
		modules = append(modules, m)
	}
	return modules, nil
}

func (h *Handler) Read(id string) (itf.Module, error) {
	var m itf.Module
	modFile, err := h.storageHandler.Open(id)
	if err != nil {
		code := http.StatusInternalServerError
		if os.IsNotExist(errors.Unwrap(err)) {
			code = http.StatusNotFound
		}
		return m, srv_base_types.NewError(code, fmt.Sprintf("opening module '%s' failed", id), err)
	}
	yd := yaml.NewDecoder(modFile)
	var mf modfile.ModFile
	err = yd.Decode(&mf)
	if err != nil {
		return m, srv_base_types.NewError(http.StatusInternalServerError, fmt.Sprintf("decoding modfile '%s' failed", id), err)
	}
	m, err = mf.ParseModule()
	if err != nil {
		return m, srv_base_types.NewError(http.StatusInternalServerError, fmt.Sprintf("parsing module '%s' failed", id), err)
	}
	return m, nil
}

func (h *Handler) Add(id string) error {
	return nil
}

func (h *Handler) Delete(id string) error {
	if err := h.storageHandler.Delete(id); err != nil {
		code := http.StatusInternalServerError
		if os.IsNotExist(errors.Unwrap(err)) {
			code = http.StatusNotFound
		}
		return srv_base_types.NewError(code, fmt.Sprintf("deleting module '%s' failed", id), err)
	}
	return nil
}

func (h *Handler) Update(id string) error {
	return nil
}
