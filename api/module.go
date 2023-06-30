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
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/input_tmplt"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
)

func (a *Api) AddModule(ctx context.Context, id, version string) (string, error) {
	err := a.mu.TryLock(fmt.Sprintf("add module '%s'", id))
	if err != nil {
		return "", model.NewResourceBusyError(err)
	}
	modules, err := a.moduleHandler.List(ctx, model.ModFilter{})
	if err != nil {
		a.mu.Unlock()
		return "", err
	}
	modMap := make(map[string]*module.Module)
	for _, m := range modules {
		modMap[m.ID] = m.Module
	}
	jID, err := a.jobHandler.Create(fmt.Sprintf("add module '%s'", id), func(ctx context.Context, cf context.CancelFunc) error {
		defer a.mu.Unlock()
		defer cf()
		err := a.addModule(ctx, id, version, modMap)
		if err == nil {
			err = ctx.Err()
		}
		return err
	})
	if err != nil {
		a.mu.Unlock()
		return "", err
	}
	return jID, nil
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
	err := a.mu.TryLock(fmt.Sprintf("delete module '%s'", id))
	if err != nil {
		return model.NewResourceBusyError(err)
	}
	defer a.mu.Unlock()
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
	err := a.mu.TryLock("check for module updates")
	if err != nil {
		return "", model.NewResourceBusyError(err)
	}
	modules, err := a.moduleHandler.List(ctx, model.ModFilter{})
	if err != nil {
		a.mu.Unlock()
		return "", err
	}
	modMap := make(map[string]*module.Module)
	for _, mod := range modules {
		modMap[mod.ID] = mod.Module
	}
	jID, err := a.jobHandler.Create("check for module updates", func(ctx context.Context, cf context.CancelFunc) error {
		defer a.mu.Unlock()
		defer cf()
		err := a.modUpdateHandler.Check(ctx, modMap)
		if err == nil {
			err = ctx.Err()
		}
		return err
	})
	if err != nil {
		a.mu.Unlock()
		return "", err
	}
	return jID, nil
}

func (a *Api) GetModuleUpdates(ctx context.Context) (map[string]model.ModUpdate, error) {
	err := a.mu.TryRLock()
	if err != nil {
		return nil, model.NewResourceBusyError(err)
	}
	defer a.mu.RUnlock()
	return a.modUpdateHandler.List(ctx), nil
}

func (a *Api) GetModuleUpdate(ctx context.Context, id string) (model.ModUpdate, error) {
	err := a.mu.TryRLock()
	if err != nil {
		return model.ModUpdate{}, model.NewResourceBusyError(err)
	}
	defer a.mu.RUnlock()
	return a.modUpdateHandler.Get(ctx, id)
}

func (a *Api) PrepareModuleUpdate(ctx context.Context, id, version string) (string, error) {
	err := a.mu.TryLock(fmt.Sprintf("prepare update for module '%s'", id))
	if err != nil {
		return "", model.NewResourceBusyError(err)
	}
	mui, err := a.modUpdateHandler.Get(ctx, id)
	if err != nil {
		a.mu.Unlock()
		return "", err
	}
	if !inSlice(version, mui.Versions) {
		a.mu.Unlock()
		return "", model.NewInvalidInputError(fmt.Errorf("unknown version '%s'", version))
	}
	modules, err := a.moduleHandler.List(ctx, model.ModFilter{})
	if err != nil {
		a.mu.Unlock()
		return "", err
	}
	modMap := make(map[string]*module.Module)
	for _, mod := range modules {
		modMap[mod.ID] = mod.Module
	}
	jID, err := a.jobHandler.Create(fmt.Sprintf("prepare update for module '%s' to '%s'", id, version), func(ctx context.Context, cf context.CancelFunc) error {
		defer a.mu.Unlock()
		defer cf()
		err := a.prepareModuleUpdate(ctx, modMap, id, version)
		if err == nil {
			err = ctx.Err()
		}
		return err
	})
	if err != nil {
		a.mu.Unlock()
		return "", err
	}
	return jID, nil
}

func (a *Api) CancelPendingModuleUpdate(ctx context.Context, id string) error {
	err := a.mu.TryLock(fmt.Sprintf("cancel pending update for module '%s'", id))
	if err != nil {
		return model.NewResourceBusyError(err)
	}
	defer a.mu.Unlock()
	return a.modUpdateHandler.CancelPending(ctx, id)
}

func (a *Api) UpdateModule(ctx context.Context, id string, depInput model.DepInput, dependencies map[string]model.DepInput, orphans bool) (string, error) {
	err := a.mu.TryLock(fmt.Sprintf("delete module '%s'", id))
	if err != nil {
		return "", model.NewResourceBusyError(err)
	}
	depList, err := a.deploymentHandler.List(ctx, model.DepFilter{})
	if err != nil {
		a.mu.Unlock()
		return "", err
	}
	stg, newIDs, uptIDs, ophIDs, err := a.modUpdateHandler.GetPending(ctx, id)
	if err != nil {
		a.mu.Unlock()
		return "", err
	}
	jID, err := a.jobHandler.Create(fmt.Sprintf("update module '%s'", id), func(ctx context.Context, cf context.CancelFunc) error {
		defer a.mu.Unlock()
		defer cf()
		err := a.updateModule(ctx, id, depInput, dependencies, orphans, stg, newIDs, uptIDs, ophIDs, depList)
		if err == nil {
			err = ctx.Err()
		}
		return err
	})
	if err != nil {
		a.mu.Unlock()
		return "", err
	}
	return jID, nil
}

func (a *Api) GetModuleUpdateTemplate(ctx context.Context, id string) (model.ModUpdateTemplate, error) {
	err := a.mu.TryRLock()
	if err != nil {
		return model.ModUpdateTemplate{}, model.NewResourceBusyError(err)
	}
	defer a.mu.RUnlock()
	stg, newIDs, uptIDs, _, err := a.modUpdateHandler.GetPending(ctx, id)
	if err != nil {
		return model.ModUpdateTemplate{}, err
	}
	depList, err := a.deploymentHandler.List(ctx, model.DepFilter{})
	if err != nil {
		return model.ModUpdateTemplate{}, err
	}
	depMap := make(map[string]string)
	for _, depMeta := range depList {
		depMap[depMeta.ModuleID] = depMeta.ID
	}
	updateTemplate := model.ModUpdateTemplate{
		Dependencies: make(map[string]model.InputTemplate),
	}
	depID, ok := depMap[id]
	if ok {
		dep, err := a.deploymentHandler.Get(ctx, depID)
		if err != nil {
			return model.ModUpdateTemplate{}, err
		}
		stgItem, ok := stg.Get(id)
		if !ok {
			return model.ModUpdateTemplate{}, model.NewInternalError(fmt.Errorf("module '%s' not staged", id))
		}
		updateTemplate.InputTemplate = input_tmplt.GetDepUpTemplate(stgItem.Module(), dep)
		for newID := range newIDs {
			stgItem, ok := stg.Get(newID)
			if !ok {
				return model.ModUpdateTemplate{}, model.NewInternalError(fmt.Errorf("module '%s' not staged", newID))
			}
			updateTemplate.Dependencies[newID] = input_tmplt.GetModDepTemplate(stgItem.Module())
		}
	}
	for uptID := range uptIDs {
		if uptID == id {
			continue
		}
		depID, ok := depMap[id]
		if ok {
			dep, err := a.deploymentHandler.Get(ctx, depID)
			if err != nil {
				var nfe *model.NotFoundError
				if !errors.As(err, &nfe) {
					return model.ModUpdateTemplate{}, err
				}
				continue
			}
			stgItem, ok := stg.Get(uptID)
			if !ok {
				return model.ModUpdateTemplate{}, model.NewInternalError(fmt.Errorf("module '%s' not staged", uptID))
			}
			updateTemplate.Dependencies[uptID] = input_tmplt.GetDepUpTemplate(stgItem.Module(), dep)
		}
	}
	return updateTemplate, nil
}

func (a *Api) GetModuleDeployTemplate(ctx context.Context, id string) (model.ModDeployTemplate, error) {
	err := a.mu.TryRLock()
	if err != nil {
		return model.ModDeployTemplate{}, model.NewResourceBusyError(err)
	}
	defer a.mu.RUnlock()
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

func (a *Api) addModule(ctx context.Context, id, version string, modMap map[string]*module.Module) error {
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
	err = a.modUpdateHandler.Prepare(ctx, modules, stg, id)
	if err != nil {
		stg.Remove()
		return err
	}
	return nil
}

func (a *Api) updateModule(ctx context.Context, id string, depInput model.DepInput, dependencies map[string]model.DepInput, orphans bool, stg handler.Stage, newIDs, uptIDs, ophIDs map[string]struct{}, depList []model.DepMeta) error {
	defer stg.Remove()
	var modDep *model.DepMeta
	depMap := make(map[string]model.DepMeta)
	for _, depMeta := range depList {
		depMap[depMeta.ModuleID] = depMeta
		if depMeta.ModuleID == id {
			modDep = &depMeta
		}
	}
	newMods := make(map[string]*module.Module)
	uptMods := make(map[string]*module.Module)
	for mID := range newIDs {
		stgItem, ok := stg.Get(mID)
		if !ok {
			return model.NewInternalError(fmt.Errorf("module '%s' not staged", mID))
		}
		mod := stgItem.Module()
		err := a.moduleHandler.Add(ctx, mod, stgItem.Dir(), stgItem.ModFile(), stgItem.Indirect())
		if err != nil {
			return err
		}
		newMods[mID] = mod
	}
	for mID := range uptIDs {
		stgItem, ok := stg.Get(mID)
		if !ok {
			return model.NewInternalError(fmt.Errorf("module '%s' not staged", mID))
		}
		modOld, err := a.moduleHandler.Get(ctx, mID)
		if err != nil {
			return err
		}
		mod := stgItem.Module()
		err = a.moduleHandler.Update(ctx, mod, stgItem.Dir(), stgItem.ModFile(), modOld.Indirect)
		if err != nil {
			return err
		}
		uptMods[mID] = mod
	}
	if modDep != nil && len(newMods) > 0 {
		order, err := sorting.GetModOrder(newMods)
		if err != nil {
			return model.NewInternalError(err)
		}
		for _, mID := range order {
			dir, err := a.moduleHandler.GetIncl(ctx, mID)
			if err != nil {
				return err
			}
			dID, err := a.deploymentHandler.Create(ctx, newMods[mID], dependencies[mID], dir, true)
			if err != nil {
				return err
			}
			if !modDep.Stopped {
				err = a.deploymentHandler.Start(ctx, dID)
				if err != nil {
					return err
				}
			}
		}
	}
	if len(uptMods) > 0 {
		order, err := sorting.GetModOrder(uptMods)
		if err != nil {
			return model.NewInternalError(err)
		}
		for _, mID := range order {
			if mod, ok := uptMods[mID]; ok {
				if dep, ok := depMap[mID]; ok {
					dir, err := a.moduleHandler.GetIncl(ctx, mID)
					if err != nil {
						return err
					}
					var dInput model.DepInput
					if mID == id {
						dInput = depInput
					} else {
						dInput = dependencies[mID]
					}
					stopped := dep.Stopped
					if modDep != nil && !modDep.Stopped {
						stopped = modDep.Stopped
					}
					dInput.Name = &dep.Name
					err = a.deploymentHandler.Update(ctx, mod, dInput, dir, dep.ID, dep.Dir, stopped, dep.Indirect)
					if err != nil {
						return err
					}
				} else {
					if modDep != nil {
						return model.NewInternalError(fmt.Errorf("missing deployment for module '%s'", mID))
					}
				}
			}
		}
	}
	// [REMINDER] implement orphan handling
	for mID := range uptIDs {
		err := a.modUpdateHandler.Remove(ctx, mID)
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

func inSlice(s string, sl []string) bool {
	for _, s2 := range sl {
		if s2 == s {
			return true
		}
	}
	return false
}
