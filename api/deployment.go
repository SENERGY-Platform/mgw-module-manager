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
	"github.com/SENERGY-Platform/mgw-module-manager/handler/dep_tmplt_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

func (a *Api) GetDeploymentTemplate(ctx context.Context, id string) (*model.DepTemplate, error) {
	mod, reqMod, err := a.moduleHandler.GetWithDep(ctx, id)
	if err != nil {
		return nil, err
	}
	return dep_tmplt_hdl.GetTemplate(mod, reqMod)
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

func (a *Api) StopDeployment(ctx context.Context, id string, dependencies bool) error {
	return a.deploymentHandler.Stop(ctx, id, dependencies)
}

func (a *Api) UpdateDeployment(ctx context.Context, id string, dr model.DepRequest) error {
	panic("not implemented")
}

func (a *Api) DeleteDeployment(ctx context.Context, id string, orphans bool) error {
	return a.deploymentHandler.Delete(ctx, id, orphans)
}
