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
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/input_tmplt"
)

func (a *Api) AddModule(_ context.Context, mr model.ModAddRequest) (string, error) {
	return a.jobHandler.Create(fmt.Sprintf("add module '%s'", mr.ID), func(ctx context.Context, cf context.CancelFunc) error {
		defer cf()
		err := a.addModule(ctx, mr)
		if err == nil {
			err = ctx.Err()
		}
		return err
	})
}

func (a *Api) GetModules(ctx context.Context, filter model.ModFilter) ([]model.ModuleMeta, error) {
	modules, err := a.moduleHandler.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	var metaList []model.ModuleMeta
	for _, m := range modules {
		metaList = append(metaList, getModMeta(m))
	}
	return metaList, nil
}

func (a *Api) GetModule(ctx context.Context, id string) (model.Module, error) {
	return a.moduleHandler.Get(ctx, id)
}

func (a *Api) DeleteModule(ctx context.Context, id string, orphans, force bool) error {
	ok, err := a.modDeployed(ctx, id)
	if err != nil {
		return err
	}
	if ok {
		return model.NewInvalidInputError(errors.New("deployment exists"))
	}
	return a.moduleHandler.Delete(ctx, id, force)
}

func (a *Api) GetModuleDeployTemplate(ctx context.Context, id string) (model.ModDeployTemplate, error) {
	mod, reqMod, err := a.moduleHandler.GetReq(ctx, id)
	if err != nil {
		return model.ModDeployTemplate{}, err
	}
	dt := model.ModDeployTemplate{InputTemplate: input_tmplt.GetModDepTemplate(mod.Module)}
	if len(reqMod) > 0 {
		rdt := make(map[string]model.InputTemplate)
		for _, rm := range reqMod {
			ok, err := a.modDeployed(ctx, rm.ID)
			if err != nil {
				return model.ModDeployTemplate{}, err
			}
			if !ok {
				rdt[rm.ID] = input_tmplt.GetModDepTemplate(rm.Module)
			}
		}
		dt.Dependencies = rdt
	}
	return dt, nil
}

func (a *Api) modDeployed(ctx context.Context, id string) (bool, error) {
	l, err := a.deploymentHandler.List(ctx, model.DepFilter{ModuleID: id})
	if err != nil {
		return false, err
	}
	if len(l) > 0 {
		return true, nil
	}
	return false, nil
}

func (a *Api) addModule(ctx context.Context, mr model.ModAddRequest) error {
	modules, err := a.moduleHandler.List(ctx, model.ModFilter{})
	if err != nil {
		return err
	}
	modMap := make(map[string]*module.Module)
	for _, m := range modules {
		modMap[m.ID] = m.Module
	}
	stage, err := a.modStagingHandler.Prepare(ctx, modMap, mr.ID, mr.Version)
	if err != nil {
		return err
	}
	defer stage.Remove()
	for _, item := range stage.Items() {
		err = a.moduleHandler.Add(ctx, item.Module(), item.Dir(), item.ModFile(), item.Indirect())
		if err != nil {
			return err
		}
	}
	return nil
}

func getModMeta(m model.Module) model.ModuleMeta {
	return model.ModuleMeta{
		ID:             m.ID,
		Name:           m.Name,
		Description:    m.Description,
		Tags:           m.Tags,
		License:        m.License,
		Author:         m.Author,
		Version:        m.Version,
		Type:           m.Type,
		DeploymentType: m.DeploymentType,
		ModuleExtra:    m.ModuleExtra,
	}
}
