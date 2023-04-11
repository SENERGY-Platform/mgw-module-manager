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
	"errors"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

func (a *Api) PrepareDeployment(ctx context.Context, id string) (model.InputTemplate, error) {
	m, err := a.moduleHandler.Get(ctx, id)
	if err != nil {
		return model.InputTemplate{}, err
	}
	if m.DeploymentType == module.SingleDeployment {
		ds, err := a.deploymentHandler.List(ctx, model.DepFilter{ModuleID: m.ID})
		if err != nil {
			return model.InputTemplate{}, err
		}
		if len(ds) > 0 {
			return model.InputTemplate{}, model.NewInternalError(errors.New("already deployed"))
		}
	}
	itm := make(map[string]model.InputTemplateBase)
	err = a.getDepInputTemplates(ctx, m, itm)
	if err != nil {
		return model.InputTemplate{}, err
	}
	return model.InputTemplate{InputTemplateBase: genInputTemplate(m), Dependencies: itm}, nil
}

func (a *Api) CreateDeployment(ctx context.Context, dr model.DepRequest) (string, error) {
	m, err := a.moduleHandler.Get(ctx, dr.ModuleID)
	if err != nil {
		return "", err
	}
	id, err := a.deploymentHandler.Create(ctx, m, dr.Name, dr.HostResources, dr.Secrets, dr.Configs)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (a *Api) GetDeployments(ctx context.Context, filter model.DepFilter) ([]model.DepMeta, error) {
	return a.deploymentHandler.List(ctx, filter)
}

func (a *Api) GetDeployment(ctx context.Context, id string) (*model.Deployment, error) {
	return a.deploymentHandler.Get(ctx, id)
}

func (a *Api) DeployDeployment(ctx context.Context, id string) error {
	panic("not implemented")
}

func (a *Api) StartDeployment(ctx context.Context, id string) error {
	panic("not implemented")
}

func (a *Api) StopDeployment(ctx context.Context, id string) error {
	panic("not implemented")
}

func (a *Api) UpdateDeployment(ctx context.Context, id string, dr model.DepRequest) error {
	m, err := a.moduleHandler.Get(ctx, dr.ModuleID)
	if err != nil {
		return err
	}
	err = a.deploymentHandler.Update(ctx, m, id, dr.Name, dr.HostResources, dr.Secrets, dr.Configs)
	if err != nil {
		return err
	}
	return nil
}

func (a *Api) DeleteDeployment(ctx context.Context, id string) error {
	return a.deploymentHandler.Delete(ctx, id)
}
