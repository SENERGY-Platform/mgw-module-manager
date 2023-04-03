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
	"time"
)

type Handler struct {
	storageHandler itf.DepStorageHandler
	cfgVltHandler  itf.CfgValidationHandler
	stgHdlTimeout  time.Duration
}

func NewHandler(storageHandler itf.DepStorageHandler, cfgVltHandler itf.CfgValidationHandler, storageHandlerTimeout time.Duration) *Handler {
	return &Handler{
		storageHandler: storageHandler,
		cfgVltHandler:  cfgVltHandler,
		stgHdlTimeout:  storageHandlerTimeout,
	}
}

func (h *Handler) List() ([]model.DepMeta, error) {
	return h.storageHandler.List()
}

func (h *Handler) Get(id string) (*model.Deployment, error) {
	return h.storageHandler.Read(id)
}

func (h *Handler) Add(m *module.Module, name *string, hostRes map[string]string, secrets map[string]string, configs map[string]any) (string, error) {
	dep, rad, sad, err := genDeployment(m, name, hostRes, secrets, configs)
	if err != nil {
		return "", err
	}
	if len(rad) > 0 || len(sad) > 0 {
		return "", errors.New("auto resource discovery not implemented")
	}
	if err = h.validateConfigs(dep.Configs, m.Configs); err != nil {
		return "", err
	}
	id, err := h.storageHandler.Create(dep)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (h *Handler) Delete(id string) error {
	return h.storageHandler.Delete(id)
}

func (h *Handler) Update(m *module.Module, name *string, hostRes map[string]string, secrets map[string]string, configs map[string]any) error {
	return nil
}

func (h *Handler) validateConfigs(dCs map[string]model.DepConfig, mCs module.Configs) error {
	for ref, dC := range dCs {
		mC := mCs[ref]
		if err := h.cfgVltHandler.ValidateValue(mC.Type, mC.TypeOpt, dC.Value, mC.IsSlice, mC.DataType); err != nil {
			return fmt.Errorf("validating config '%s' failed: %s", ref, err)
		}
		if mC.Options != nil && !mC.OptExt {
			if err := h.cfgVltHandler.ValidateValInOpt(mC.Options, dC.Value, mC.IsSlice, mC.DataType); err != nil {
				return fmt.Errorf("validating config '%s' failed: %s", ref, err)
			}
		}
	}
	return nil
}
