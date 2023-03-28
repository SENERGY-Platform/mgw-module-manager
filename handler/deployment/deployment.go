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

package deployment

import (
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"module-manager/itf"
	"module-manager/model"
)

type Handler struct {
	storageHandler itf.DepStorageHandler
	cfgVltHandler  itf.CfgValidationHandler
}

func NewHandler(storageHandler itf.DepStorageHandler, cfgVltHandler itf.CfgValidationHandler) *Handler {
	return &Handler{storageHandler: storageHandler, cfgVltHandler: cfgVltHandler}
}

func (h *Handler) List() ([]model.Deployment, error) {
	return nil, nil
}

func (h *Handler) Get(id string) (model.Deployment, error) {
	return model.Deployment{}, nil
}

func (h *Handler) Add(b model.DeploymentBase, m *module.Module) (string, error) {
	return "", nil
}

func (h *Handler) Start(id string) error {
	return nil
}

func (h *Handler) Stop(id string) error {
	return nil
}

func (h *Handler) Delete(id string) error {
	return nil
}

func (h *Handler) Update(id string) error {
	return nil
}

func (h *Handler) validateConfigs(dCs map[string]any, mCs module.Configs) error {
	for ref, val := range dCs {
		mC := mCs[ref]
		if mC.IsSlice {
			if err := h.cfgVltHandler.ValidateValSlice(mC.Type, mC.TypeOpt, val, mC.DataType); err != nil {
				return fmt.Errorf("validating config '%s' failed: %s", ref, err)
			}
		} else {
			if err := h.cfgVltHandler.ValidateValue(mC.Type, mC.TypeOpt, val); err != nil {
				return fmt.Errorf("validating config '%s' failed: %s", ref, err)
			}
		}
	}
	return nil
}
