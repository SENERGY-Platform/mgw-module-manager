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
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dep_tmplt"
)

func (a *Api) AddModule(_ context.Context, mr model.ModRequest) (string, error) {
	return a.jobHandler.Create(fmt.Sprintf("add module '%s'", mr.ID), func(ctx context.Context, cf context.CancelFunc) error {
		defer cf()
		err := a.moduleHandler.Add(ctx, mr)
		if err == nil {
			err = ctx.Err()
		}
		return err
	})
}

func (a *Api) GetModules(ctx context.Context, filter model.ModFilter) ([]model.ModuleMeta, error) {
	return a.moduleHandler.List(ctx, filter)
}

func (a *Api) GetModule(ctx context.Context, id string) (model.Module, error) {
	return a.moduleHandler.Get(ctx, id)
}

func (a *Api) DeleteModule(ctx context.Context, id string, orphans bool) error {
	l, err := a.deploymentHandler.List(ctx, model.DepFilter{ModuleID: id})
	if err != nil {
		return err
	}
	if len(l) > 0 {
		return model.NewInvalidInputError(errors.New("deployment exists"))
	}
	return a.moduleHandler.Delete(ctx, id)
}

func (a *Api) GetDeploymentTemplate(ctx context.Context, id string) (*model.DepTemplate, error) {
	mod, reqMod, err := a.moduleHandler.GetReq(ctx, id)
	if err != nil {
		return nil, err
	}
	dt := model.DepTemplate{ModuleID: mod.ID, InputTemplate: dep_tmplt.GetDepTemplate(mod.Module)}
	if len(reqMod) > 0 {
		rdt := make(map[string]model.InputTemplate)
		for _, rm := range reqMod {
			rdt[rm.ID] = dep_tmplt.GetDepTemplate(rm.Module)
		}
		dt.Dependencies = rdt
	}
	return &dt, nil
}
