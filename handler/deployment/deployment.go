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
	"module-manager/itf"
	"module-manager/model"
)

type Handler struct {
	storageHandler itf.DeploymentStorageHandler
	cfgVltHandler  itf.ConfigValidationHandler
}

func NewHandler(storageHandler itf.DeploymentStorageHandler, cfgVltHandler itf.ConfigValidationHandler) *Handler {
	return &Handler{storageHandler: storageHandler, cfgVltHandler: cfgVltHandler}
}

func (h *Handler) List() ([]model.Deployment, error) {
	return nil, nil
}

func (h *Handler) Read(id string) (model.Deployment, error) {
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

func (h *Handler) InputTemplate(m *module.Module) model.InputTemplate {
	it := model.InputTemplate{
		Resources:   make(map[string]model.InputTemplateResource),
		Secrets:     make(map[string]model.InputTemplateSecret),
		Configs:     make(map[string]model.InputTemplateConfig),
		InputGroups: m.Inputs.Groups,
	}
	for ref, input := range m.Inputs.Resources {
		it.Resources[ref] = model.InputTemplateResource{
			Input: input,
			Tags:  m.Resources[ref],
		}
	}
	for ref, input := range m.Inputs.Secrets {
		it.Secrets[ref] = model.InputTemplateSecret{
			Input:  input,
			Secret: m.Secrets[ref],
		}
	}
	for ref, input := range m.Inputs.Configs {
		cv := m.Configs[ref]
		itc := model.InputTemplateConfig{
			Input:    input,
			Default:  cv.Default,
			Options:  cv.Options,
			OptExt:   cv.OptExt,
			Type:     cv.Type,
			TypeOpt:  make(map[string]any),
			DataType: cv.DataType,
			IsList:   cv.IsSlice,
		}
		for key, opt := range cv.TypeOpt {
			itc.TypeOpt[key] = opt.Value
		}
		it.Configs[ref] = itc
	}
	return it
}
