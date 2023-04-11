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
	"context"
	"github.com/SENERGY-Platform/go-service-base/srv-base"
	"github.com/SENERGY-Platform/mgw-modfile-lib/modfile"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"gopkg.in/yaml.v3"
)

type Handler struct {
	storageHandler          handler.ModStorageHandler
	transferHandler         handler.ModTransferHandler
	configValidationHandler handler.CfgValidationHandler
	mfDecoders              modfile.Decoders
	mfGenerators            modfile.Generators
}

func NewHandler(storageHandler handler.ModStorageHandler, transferHandler handler.ModTransferHandler, configValidationHandler handler.CfgValidationHandler, mfDecoders modfile.Decoders, mfGenerators modfile.Generators) *Handler {
	return &Handler{
		storageHandler:          storageHandler,
		transferHandler:         transferHandler,
		configValidationHandler: configValidationHandler,
		mfDecoders:              mfDecoders,
		mfGenerators:            mfGenerators,
	}
}

func (h *Handler) List(ctx context.Context) ([]*module.Module, error) {
	mIds, err := h.storageHandler.List(ctx)
	if err != nil {
		return nil, err
	}
	var modules []*module.Module
	for _, id := range mIds {
		file, err := h.storageHandler.Open(ctx, id)
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

func (h *Handler) Get(ctx context.Context, id string) (*module.Module, error) {
	file, err := h.storageHandler.Open(ctx, id)
	if err != nil {
		return nil, err
	}
	yd := yaml.NewDecoder(file)
	mf := modfile.New(h.mfDecoders, h.mfGenerators)
	err = yd.Decode(&mf)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	m, err := mf.GetModule()
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	return m, nil
}

func (h *Handler) Add(ctx context.Context, id string) error {
	panic("not implemented")
}

func (h *Handler) Delete(ctx context.Context, id string) error {
	return h.storageHandler.Delete(ctx, id)
}

func (h *Handler) Update(ctx context.Context, id string) error {
	panic("not implemented")
}
