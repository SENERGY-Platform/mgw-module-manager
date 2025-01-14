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

package manager

import (
	"context"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/input_tmplt"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
)

func (m *Manager) AddModule(ctx context.Context, id, version string) (string, error) {
	metaStr := fmt.Sprintf("add module (id=%s version=%s)", id, version)
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	if mID, ok := m.pendingModUpdate(ctx); ok {
		m.mu.Unlock()
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(fmt.Errorf("update pending for '%s'", mID)))
	}
	modules, err := m.moduleHandler.List(ctx, lib_model.ModFilter{}, false)
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	modMap := make(map[string]*module.Module)
	for _, m := range modules {
		modMap[m.ID] = m.Module.Module
	}
	jID, err := m.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		err := m.addModule(ctx, id, version, modMap)
		if err == nil {
			err = ctx.Err()
		}
		return nil, err
	})
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (m *Manager) GetModules(ctx context.Context, filter lib_model.ModFilter) (map[string]lib_model.Module, error) {
	modList, err := m.moduleHandler.List(ctx, filter, false)
	if err != nil {
		return nil, newApiErr(fmt.Sprintf("get modules (%s)", getModFilterValues(filter)), err)
	}
	modules := make(map[string]lib_model.Module)
	for mID, m := range modList {
		modules[mID] = m.Module
	}
	return modules, nil
}

func (m *Manager) GetModule(ctx context.Context, id string) (lib_model.Module, error) {
	mod, err := m.moduleHandler.Get(ctx, id, false)
	if err != nil {
		return lib_model.Module{}, newApiErr(fmt.Sprintf("get module (id=%s)", id), err)
	}
	return mod.Module, err
}

func (m *Manager) DeleteModule(ctx context.Context, id string, force bool) (string, error) {
	metaStr := fmt.Sprintf("delete module (id=%s force=%v)", id, force)
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	if mID, ok := m.pendingModUpdate(ctx); ok {
		m.mu.Unlock()
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(fmt.Errorf("update pending for '%s'", mID)))
	}
	ok, err := m.modDeployed(ctx, id)
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	if ok {
		m.mu.Unlock()
		return "", newApiErr(metaStr, lib_model.NewInvalidInputError(errors.New("deployment exists")))
	}
	jID, err := m.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		err := m.moduleHandler.Delete(ctx, id, force)
		if err == nil {
			err = ctx.Err()
		}
		return nil, err
	})
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (m *Manager) CheckModuleUpdates(ctx context.Context) (string, error) {
	metaStr := fmt.Sprintf("check module updates")
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	modules, err := m.moduleHandler.List(ctx, lib_model.ModFilter{}, false)
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	modMap := make(map[string]*module.Module)
	for _, mod := range modules {
		modMap[mod.ID] = mod.Module.Module
	}
	jID, err := m.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		err := m.modUpdateHandler.Check(ctx, modMap)
		if err == nil {
			err = ctx.Err()
		}
		return nil, err
	})
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (m *Manager) GetModuleUpdates(ctx context.Context) (map[string]lib_model.ModUpdate, error) {
	return m.modUpdateHandler.List(ctx), nil
}

func (m *Manager) GetModuleUpdate(ctx context.Context, id string) (lib_model.ModUpdate, error) {
	metaStr := fmt.Sprintf("get module update (id=%s)", id)
	err := m.mu.TryRLock()
	if err != nil {
		return lib_model.ModUpdate{}, newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	defer m.mu.RUnlock()
	update, err := m.modUpdateHandler.Get(ctx, id)
	if err != nil {
		return lib_model.ModUpdate{}, newApiErr(metaStr, err)
	}
	return update, nil
}

func (m *Manager) PrepareModuleUpdate(ctx context.Context, id, version string) (string, error) {
	metaStr := fmt.Sprintf("prepare module update (id=%s version=%s)", id, version)
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	mui, err := m.modUpdateHandler.Get(ctx, id)
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	if !inSlice(version, mui.Versions) {
		m.mu.Unlock()
		return "", newApiErr(metaStr, lib_model.NewInvalidInputError(errors.New("unknown version")))
	}
	modules, err := m.moduleHandler.List(ctx, lib_model.ModFilter{}, false)
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	modMap := make(map[string]*module.Module)
	for _, mod := range modules {
		modMap[mod.ID] = mod.Module.Module
	}
	jID, err := m.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		err := m.prepareModuleUpdate(ctx, modMap, id, version)
		if err == nil {
			err = ctx.Err()
		}
		return nil, err
	})
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (m *Manager) CancelPendingModuleUpdate(ctx context.Context, id string) error {
	metaStr := fmt.Sprintf("cancel pending module update (id=%s)", id)
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	defer m.mu.Unlock()
	err = m.modUpdateHandler.CancelPending(ctx, id)
	if err != nil {
		return newApiErr(metaStr, err)
	}
	return nil
}

func (m *Manager) UpdateModule(ctx context.Context, id string, depInput lib_model.DepInput, dependencies map[string]lib_model.DepInput) (string, error) {
	metaStr := fmt.Sprintf("update module (id=%s)", id)
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	deployments, err := m.deploymentHandler.List(ctx, lib_model.DepFilter{}, false, false, false, false)
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	stg, newIDs, uptIDs, ophIDs, err := m.modUpdateHandler.GetPending(ctx, id)
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	jID, err := m.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		err := m.updateModule(ctx, id, depInput, dependencies, stg, newIDs, uptIDs, ophIDs, deployments)
		if err == nil {
			err = ctx.Err()
		}
		return nil, err
	})
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (m *Manager) GetModuleUpdateTemplate(ctx context.Context, id string) (lib_model.ModUpdateTemplate, error) {
	metaStr := fmt.Sprintf("get module update template (id=%s)", id)
	err := m.mu.TryRLock()
	if err != nil {
		return lib_model.ModUpdateTemplate{}, newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	defer m.mu.RUnlock()
	stg, newIDs, uptIDs, _, err := m.modUpdateHandler.GetPending(ctx, id)
	if err != nil {
		return lib_model.ModUpdateTemplate{}, newApiErr(metaStr, err)
	}
	deployments, err := m.deploymentHandler.List(ctx, lib_model.DepFilter{}, false, false, false, false)
	if err != nil {
		return lib_model.ModUpdateTemplate{}, newApiErr(metaStr, err)
	}
	moduleDepMap := make(map[string]string)
	for _, dep := range deployments {
		moduleDepMap[dep.Module.ID] = dep.ID
	}
	updateTemplate := lib_model.ModUpdateTemplate{
		Dependencies: make(map[string]lib_model.InputTemplate),
	}
	depID, ok := moduleDepMap[id]
	if ok {
		dep, err := m.deploymentHandler.Get(ctx, depID, true, true, false, false)
		if err != nil {
			return lib_model.ModUpdateTemplate{}, newApiErr(metaStr, err)
		}
		stgItem, ok := stg.Get(id)
		if !ok {
			return lib_model.ModUpdateTemplate{}, newApiErr(metaStr, lib_model.NewInternalError(fmt.Errorf("module '%s' not staged", id)))
		}
		updateTemplate.InputTemplate = input_tmplt.GetDepUpTemplate(stgItem.Module(), dep)
		for newID := range newIDs {
			stgItem, ok := stg.Get(newID)
			if !ok {
				return lib_model.ModUpdateTemplate{}, newApiErr(metaStr, lib_model.NewInternalError(fmt.Errorf("module '%s' not staged", newID)))
			}
			updateTemplate.Dependencies[newID] = input_tmplt.GetModDepTemplate(stgItem.Module())
		}
	}
	for uptID := range uptIDs {
		if uptID == id {
			continue
		}
		depID, ok := moduleDepMap[id]
		if ok {
			dep, err := m.deploymentHandler.Get(ctx, depID, true, true, false, false)
			if err != nil {
				var nfe *lib_model.NotFoundError
				if !errors.As(err, &nfe) {
					return lib_model.ModUpdateTemplate{}, newApiErr(metaStr, err)
				}
				continue
			}
			stgItem, ok := stg.Get(uptID)
			if !ok {
				return lib_model.ModUpdateTemplate{}, newApiErr(metaStr, lib_model.NewInternalError(fmt.Errorf("module '%s' not staged", uptID)))
			}
			updateTemplate.Dependencies[uptID] = input_tmplt.GetDepUpTemplate(stgItem.Module(), dep)
		}
	}
	return updateTemplate, nil
}

func (m *Manager) GetModuleDeployTemplate(ctx context.Context, id string) (lib_model.ModDeployTemplate, error) {
	metaStr := fmt.Sprintf("get module deploy template (id=%s)", id)
	err := m.mu.TryRLock()
	if err != nil {
		return lib_model.ModDeployTemplate{}, newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	defer m.mu.RUnlock()
	modTree, err := m.moduleHandler.GetTree(ctx, id)
	if err != nil {
		return lib_model.ModDeployTemplate{}, newApiErr(metaStr, err)
	}
	mod := modTree[id]
	delete(modTree, id)
	dt := lib_model.ModDeployTemplate{InputTemplate: input_tmplt.GetModDepTemplate(mod.Module.Module)}
	if len(modTree) > 0 {
		rdt := make(map[string]lib_model.InputTemplate)
		for _, rm := range modTree {
			ok, err := m.modDeployed(ctx, rm.ID)
			if err != nil {
				return lib_model.ModDeployTemplate{}, newApiErr(metaStr, err)
			}
			if !ok {
				rdt[rm.ID] = input_tmplt.GetModDepTemplate(rm.Module.Module)
			}
		}
		dt.Dependencies = rdt
	}
	return dt, nil
}

func (m *Manager) modDeployed(ctx context.Context, id string) (bool, error) {
	l, err := m.deploymentHandler.List(ctx, lib_model.DepFilter{ModuleID: id}, false, false, false, false)
	if err != nil {
		return false, err
	}
	if len(l) > 0 {
		return true, nil
	}
	return false, nil
}

func (m *Manager) addModule(ctx context.Context, id, version string, modMap map[string]*module.Module) error {
	stage, err := m.modStagingHandler.Prepare(ctx, modMap, id, version)
	if err != nil {
		return err
	}
	defer stage.Remove()
	for _, item := range stage.Items() {
		err = m.moduleHandler.Add(ctx, item.Module(), item.Dir(), item.ModFile())
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) prepareModuleUpdate(ctx context.Context, modules map[string]*module.Module, id, version string) error {
	stg, err := m.modStagingHandler.Prepare(ctx, nil, id, version)
	if err != nil {
		return err
	}
	err = m.modUpdateHandler.Prepare(ctx, modules, stg, id)
	if err != nil {
		stg.Remove()
		return err
	}
	return nil
}

func (m *Manager) updateModule(ctx context.Context, id string, depInput lib_model.DepInput, dependencies map[string]lib_model.DepInput, stg handler.Stage, newIDs, uptIDs, ophIDs map[string]struct{}, deployments map[string]lib_model.Deployment) error {
	defer stg.Remove()
	modDepMap := make(map[string]lib_model.Deployment)
	for _, dep := range deployments {
		modDepMap[dep.Module.ID] = dep
	}
	oldRootDep, rootDeployed := modDepMap[id]
	stgMods := make(map[string]*module.Module)
	for mID, stgItem := range stg.Items() {
		stgMods[mID] = stgItem.Module()
	}
	order, err := sorting.GetModOrder(stgMods)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	for _, mID := range order {
		stgItem, _ := stg.Get(mID)
		if _, ok := newIDs[mID]; ok {
			if rootDeployed {
				dID, err := m.deploymentHandler.Create(ctx, stgItem.Module(), dependencies[mID], stgItem.Dir(), true)
				if err != nil {
					return err
				}
				if oldRootDep.Enabled {
					if _, err = m.deploymentHandler.Start(ctx, dID, false); err != nil {
						return err
					}
				}
			}
			if err = m.moduleHandler.Add(ctx, stgItem.Module(), stgItem.Dir(), stgItem.ModFile()); err != nil {
				return err
			}
		}
		if _, ok := uptIDs[mID]; ok {
			var dInput lib_model.DepInput
			if mID == id {
				dInput = depInput
			} else {
				dInput = dependencies[mID]
			}
			oldDep, deployed := modDepMap[mID]
			if deployed {
				dInput.Name = &oldDep.Name
				err = m.deploymentHandler.Update(ctx, oldDep.ID, stgItem.Module(), dInput, stgItem.Dir())
				if err != nil {
					return err
				}
				if _, err = m.updateAllAuxDeployments(ctx, oldDep.ID, stgItem.Module()); err != nil {
					util.Logger.Error(err)
				}
			} else {
				if rootDeployed {
					dID, err := m.deploymentHandler.Create(ctx, stgItem.Module(), dependencies[mID], stgItem.Dir(), true)
					if err != nil {
						return err
					}
					if oldRootDep.Enabled {
						if _, err = m.deploymentHandler.Start(ctx, dID, false); err != nil {
							return err
						}
					}
				}
			}
			err = m.moduleHandler.Update(ctx, stgItem.Module(), stgItem.Dir(), stgItem.ModFile())
			if err != nil {
				return err
			}
		}
	}
	// [REMINDER] implement orphan handling
	for mID := range uptIDs {
		err := m.modUpdateHandler.Remove(ctx, mID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) pendingModUpdate(ctx context.Context) (string, bool) {
	updates := m.modUpdateHandler.List(ctx)
	for mID, update := range updates {
		if update.Pending {
			return mID, true
		}
	}
	return "", false
}

func inSlice(s string, sl []string) bool {
	for _, s2 := range sl {
		if s2 == s {
			return true
		}
	}
	return false
}

func getModFilterValues(filter lib_model.ModFilter) string {
	return fmt.Sprintf("ids=%v type=%s deployment_type=%v name=%s author=%s tags=%v", filter.IDs, filter.Type, filter.DeploymentType, filter.Name, filter.Author, filter.Tags)
}
