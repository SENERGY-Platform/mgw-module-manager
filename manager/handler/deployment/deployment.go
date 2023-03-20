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
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"module-manager/manager/itf"
)

type Handler struct {
	storageHandler itf.DeploymentStorageHandler
}

func NewHandler(storageHandler itf.DeploymentStorageHandler) *Handler {
	return &Handler{storageHandler: storageHandler}
}

func (h *Handler) List() ([]itf.Deployment, error) {
	return nil, nil
}

func (h *Handler) Read(id string) (itf.Deployment, error) {
	return itf.Deployment{}, nil
}

func (h *Handler) Add(b itf.DeploymentBase, m module.Module) error {
	return nil
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

func (h *Handler) InputTemplate(m *module.Module) itf.InputTemplate {
	it := itf.InputTemplate{InputGroups: m.Inputs.Groups}
	if m.Inputs.Resources != nil {
		it.Resources = make(map[string]module.Input)
		for ref, input := range m.Inputs.Resources {
			it.Resources[ref] = input
		}
	}
	if m.Inputs.Secrets != nil {
		it.Secrets = make(map[string]itf.InputTemplateSecret)
		for ref, input := range m.Inputs.Secrets {
			it.Secrets[ref] = itf.InputTemplateSecret{
				Input:  input,
				Secret: m.Secrets[ref],
			}
		}
	}
	if m.Inputs.Configs != nil {
		it.Configs = make(map[string]itf.InputTemplateConfig)
		for ref, input := range m.Inputs.Configs {
			cv := m.Configs[ref]
			itc := itf.InputTemplateConfig{
				Input:    input,
				Default:  cv.Default,
				Options:  cv.Options,
				OptExt:   cv.OptExt,
				Type:     cv.Type,
				DataType: cv.DataType,
				IsList:   cv.IsSlice,
			}
			if cv.TypeOpt != nil {
				itc.TypeOpt = make(map[string]any)
				for key, opt := range cv.TypeOpt {
					itc.TypeOpt[key] = opt.Value
				}
			}
			it.Configs[ref] = itc
		}
	}
	return it
}
