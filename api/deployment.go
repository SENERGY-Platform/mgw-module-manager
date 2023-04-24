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

package api

import (
	"context"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

func (a *Api) PrepareDeployment(ctx context.Context, id string) (model.InputTemplate, error) {
	m, err := a.moduleHandler.Get(ctx, id)
	if err != nil {
		return model.InputTemplate{}, err
	}
	itm := make(map[string]model.InputTemplateBase)
	err = a.getDepInputTemplates(ctx, m, itm)
	if err != nil {
		return model.InputTemplate{}, err
	}
	return model.InputTemplate{ModuleID: m.ID, InputTemplateBase: genInputTemplate(m), Dependencies: itm}, nil
}

func (a *Api) CreateDeployment(ctx context.Context, dr model.DepRequest) (string, error) {
	return a.deploymentHandler.Create(ctx, dr)
}

func (a *Api) GetDeployments(ctx context.Context, filter model.DepFilter) ([]model.DepMeta, error) {
	return a.deploymentHandler.List(ctx, filter)
}

func (a *Api) GetDeployment(ctx context.Context, id string) (*model.Deployment, error) {
	return a.deploymentHandler.Get(ctx, id)
}

func (a *Api) StartDeployment(ctx context.Context, id string) error {
	return a.deploymentHandler.Start(ctx, id)
}

func (a *Api) StopDeployment(ctx context.Context, id string) error {
	return a.deploymentHandler.Stop(ctx, id)
}

func (a *Api) UpdateDeployment(ctx context.Context, id string, dr model.DepRequest) error {
	panic("not implemented")
}

func (a *Api) DeleteDeployment(ctx context.Context, id string, orphans bool) error {
	return a.deploymentHandler.Delete(ctx, id, orphans)
}

func (a *Api) getDepInputTemplates(ctx context.Context, m *module.Module, itm map[string]model.InputTemplateBase) error {
	for mdID := range m.Dependencies {
		if _, ok := itm[mdID]; !ok {
			ds, err := a.deploymentHandler.List(ctx, model.DepFilter{ModuleID: mdID})
			if err != nil {
				return err
			}
			if len(ds) < 1 {
				md, err := a.moduleHandler.Get(ctx, mdID)
				if err != nil {
					return err
				}
				itm[mdID] = genInputTemplate(md)
				err = a.getDepInputTemplates(ctx, md, itm)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func genInputTemplate(m *module.Module) model.InputTemplateBase {
	it := model.InputTemplateBase{
		HostResources: make(map[string]model.InputTemplateHostRes),
		Secrets:       make(map[string]model.InputTemplateSecret),
		Configs:       make(map[string]model.InputTemplateConfig),
		InputGroups:   m.Inputs.Groups,
	}
	for ref, input := range m.Inputs.Resources {
		it.HostResources[ref] = model.InputTemplateHostRes{
			Input:        input,
			HostResource: m.HostResources[ref],
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
			Required: cv.Required,
		}
		for key, opt := range cv.TypeOpt {
			itc.TypeOpt[key] = opt.Value
		}
		it.Configs[ref] = itc
	}
	return it
}
