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

func (a *Api) AddModule(_ context.Context, id, version string) (string, error) {
	return a.jobHandler.Create(fmt.Sprintf("add module '%s'", id), func(ctx context.Context, cf context.CancelFunc) error {
		defer cf()
		err := a.addModule(ctx, id, version)
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

func (a *Api) CheckModuleUpdates(ctx context.Context) (string, error) {
	modules, err := a.moduleHandler.List(ctx, model.ModFilter{})
	if err != nil {
		return "", err
	}
	modMap := make(map[string]*module.Module)
	for _, mod := range modules {
		modMap[mod.ID] = mod.Module
	}
	return a.jobHandler.Create("check for module updates", func(ctx context.Context, cf context.CancelFunc) error {
		defer cf()
		err := a.modUpdateHandler.Check(ctx, modMap)
		if err == nil {
			err = ctx.Err()
		}
		return err
	})
}

func (a *Api) GetModuleUpdates(ctx context.Context) map[string]model.ModUpdate {
	return a.modUpdateHandler.List(ctx)
}

func (a *Api) GetModuleUpdate(ctx context.Context, id string) (model.ModUpdate, error) {
	return a.modUpdateHandler.Get(ctx, id)
}

func (a *Api) PrepareModuleUpdate(ctx context.Context, id, version string) (string, error) {
	mui, err := a.modUpdateHandler.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if !inSlice(version, mui.Versions) {
		return "", model.NewInvalidInputError(fmt.Errorf("unknown version '%s'", version))
	}
	modules, err := a.moduleHandler.List(ctx, model.ModFilter{})
	if err != nil {
		return "", err
	}
	modMap := make(map[string]*module.Module)
	for _, mod := range modules {
		modMap[mod.ID] = mod.Module
	}
	return a.jobHandler.Create(fmt.Sprintf("prepare update for module '%s' to '%s'", id, version), func(ctx context.Context, cf context.CancelFunc) error {
		defer cf()
		err := a.prepareModuleUpdate(ctx, modMap, id, version)
		if err == nil {
			err = ctx.Err()
		}
		return err
	})
}

func (a *Api) CancelPendingModuleUpdate(ctx context.Context, id string) error {
	return a.modUpdateHandler.CancelPending(ctx, id)
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

func (a *Api) addModule(ctx context.Context, id, version string) error {
	modules, err := a.moduleHandler.List(ctx, model.ModFilter{})
	if err != nil {
		return err
	}
	modMap := make(map[string]*module.Module)
	for _, m := range modules {
		modMap[m.ID] = m.Module
	}
	stage, err := a.modStagingHandler.Prepare(ctx, modMap, id, version)
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

func (a *Api) prepareModuleUpdate(ctx context.Context, modules map[string]*module.Module, id, version string) error {
	stg, err := a.modStagingHandler.Prepare(ctx, nil, id, version)
	if err != nil {
		return err
	}
	return a.modUpdateHandler.Prepare(ctx, modules, stg, id)
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

func inSlice(s string, sl []string) bool {
	for _, s2 := range sl {
		if s2 == s {
			return true
		}
	}
	return false
}
