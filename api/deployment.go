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
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/input_tmplt"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
)

func (a *Api) CreateDeployment(ctx context.Context, id string, depInput model.DepInput, dependencies map[string]model.DepInput) (string, error) {
	err := a.mu.TryLock(fmt.Sprintf("create deployment for '%s'", id))
	if err != nil {
		return "", model.NewResourceBusyError(err)
	}
	defer a.mu.Unlock()
	mod, reqMod, err := a.moduleHandler.GetReq(ctx, id)
	if err != nil {
		return "", err
	}
	if mod.DeploymentType == module.SingleDeployment {
		if l, err := a.deploymentHandler.List(ctx, model.DepFilter{ModuleID: mod.ID}); err != nil {
			return "", err
		} else if len(l) > 0 {
			return "", model.NewInvalidInputError(errors.New("already deployed"))
		}
	}
	if len(reqMod) > 0 {
		modMap := make(map[string]*module.Module)
		for _, m := range reqMod {
			modMap[m.ID] = m.Module
		}
		order, err := sorting.GetModOrder(modMap)
		if err != nil {
			return "", model.NewInternalError(err)
		}
		var er error
		var dIDs []string
		defer func() {
			if er != nil {
				for _, id := range dIDs {
					e := a.DeleteDeployment(context.Background(), id, true)
					if e != nil {
						util.Logger.Error(e)
					}
				}
			}
		}()
		var ok bool
		var dID string
		for _, rmID := range order {
			ok, dID, er = a.createDepIfNotExist(ctx, rmID, dependencies[rmID])
			if er != nil {
				return "", er
			}
			if ok {
				dIDs = append(dIDs, dID)
			}
		}
	}
	dir, err := a.moduleHandler.GetIncl(ctx, mod.ID)
	if err != nil {
		return "", err
	}
	dID, err := a.deploymentHandler.Create(ctx, mod.Module, depInput, dir, false)
	if err != nil {
		return "", err
	}
	return dID, nil
}

func (a *Api) GetDeployments(ctx context.Context, filter model.DepFilter) ([]model.DepMeta, error) {
	return a.deploymentHandler.List(ctx, filter)
}

func (a *Api) GetDeployment(ctx context.Context, id string) (*model.Deployment, error) {
	return a.deploymentHandler.Get(ctx, id)
}

func (a *Api) GetDeploymentInstances(ctx context.Context) (map[string]model.DepInstance, error) {
	return a.deploymentHandler.ListInstances(ctx)
}

func (a *Api) GetDeploymentInstance(ctx context.Context, id string) (model.DepInstance, error) {
	return a.deploymentHandler.GetInstance(ctx, id)
}

func (a *Api) StartDeployment(ctx context.Context, id string) error {
	err := a.mu.TryLock(fmt.Sprintf("start deployment '%s'", id))
	if err != nil {
		return model.NewResourceBusyError(err)
	}
	defer a.mu.Unlock()
	return a.deploymentHandler.Enable(ctx, id)
}

func (a *Api) StopDeployment(_ context.Context, id string, dependencies bool) (string, error) {
	err := a.mu.TryLock(fmt.Sprintf("stop deployment '%s'", id))
	if err != nil {
		return "", model.NewResourceBusyError(err)
	}
	jID, err := a.jobHandler.Create(fmt.Sprintf("stop deployment '%s'", id), func(ctx context.Context, cf context.CancelFunc) error {
		defer a.mu.Unlock()
		defer cf()
		err := a.deploymentHandler.Disable(ctx, id, dependencies)
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

func (a *Api) StartDeployments() error {
	depList, err := a.deploymentHandler.List(context.Background(), model.DepFilter{})
	if err != nil {
		return err
	}
	if len(depList) > 0 {
		err = a.mu.TryLock("start deployments")
		if err != nil {
			return model.NewResourceBusyError(err)
		}
		depMap := make(map[string]*model.Deployment)
		for _, depMeta := range depList {
			dep, err := a.deploymentHandler.Get(context.Background(), depMeta.ID)
			if err != nil {
				a.mu.Unlock()
				return err
			}
			depMap[depMeta.ID] = dep
		}
		order, err := sorting.GetDepOrder(depMap)
		if err != nil {
			a.mu.Unlock()
			return err
		}
		_, err = a.jobHandler.Create("start deployments", func(ctx context.Context, cf context.CancelFunc) error {
			defer a.mu.Unlock()
			defer cf()
			err := a.startDeployments(ctx, depMap, order)
			if err == nil {
				err = ctx.Err()
			}
			return err
		})
		if err != nil {
			a.mu.Unlock()
			return err
		}
	}
	return nil
}

func (a *Api) GetDeploymentsHealth(ctx context.Context) (map[string]model.DepHealthInfo, error) {
	deployments, err := a.deploymentHandler.List(ctx, model.DepFilter{})
	if err != nil {
		return nil, err
	}
	instances := make(map[string]model.DepInstance)
	for _, dep := range deployments {
		if dep.Enabled {
			instance, err := a.deploymentHandler.GetInstance(ctx, dep.ID)
			if err != nil {
				return nil, err
			}
			instances[dep.ID] = instance
		}
	}
	return a.depHealthHandler.List(ctx, instances)
}

func (a *Api) GetDeploymentHealth(ctx context.Context, dID string) (model.DepHealthInfo, error) {
	instance, err := a.deploymentHandler.GetInstance(ctx, dID)
	if err != nil {
		return model.DepHealthInfo{}, err
	}
	return a.depHealthHandler.Get(ctx, instance)
}

func (a *Api) startDeployments(ctx context.Context, depMap map[string]*model.Deployment, order []string) error {
	for _, dID := range order {
		dep, ok := depMap[dID]
		if !ok {
			return fmt.Errorf("deployment '%s' does not exist", dID)
		}
		if dep.Enabled {
			err := a.deploymentHandler.Enable(ctx, dID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Api) UpdateDeployment(ctx context.Context, dID string, depInput model.DepInput) (string, error) {
	err := a.mu.TryLock(fmt.Sprintf("update deployment '%s'", dID))
	if err != nil {
		return "", model.NewResourceBusyError(err)
	}
	dep, err := a.deploymentHandler.Get(ctx, dID)
	if err != nil {
		a.mu.Unlock()
		return "", err
	}
	mod, err := a.moduleHandler.Get(ctx, dep.ModuleID)
	if err != nil {
		a.mu.Unlock()
		return "", err
	}
	if mod.ID != dep.ModuleID {
		a.mu.Unlock()
		return "", model.NewInvalidInputError(errors.New("module ID mismatch"))
	}
	jID, err := a.jobHandler.Create(fmt.Sprintf("update deployment '%s'", dID), func(ctx context.Context, cf context.CancelFunc) error {
		defer a.mu.Unlock()
		defer cf()
		err := a.deploymentHandler.Update(ctx, mod.Module, depInput, "", dID, dep.Dir, dep.Enabled, dep.Indirect)
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

func (a *Api) DeleteDeployment(ctx context.Context, id string, orphans bool) error {
	err := a.mu.TryLock(fmt.Sprintf("delete deployment '%s'", id))
	if err != nil {
		return model.NewResourceBusyError(err)
	}
	defer a.mu.Unlock()
	return a.deploymentHandler.Delete(ctx, id, orphans)
}

func (a *Api) GetDeploymentUpdateTemplate(ctx context.Context, id string) (model.DepUpdateTemplate, error) {
	err := a.mu.TryRLock()
	if err != nil {
		return model.DepUpdateTemplate{}, model.NewResourceBusyError(err)
	}
	defer a.mu.RUnlock()
	dep, err := a.deploymentHandler.Get(ctx, id)
	if err != nil {
		return model.DepUpdateTemplate{}, err
	}
	mod, err := a.moduleHandler.Get(ctx, dep.ModuleID)
	if err != nil {
		return model.DepUpdateTemplate{}, err
	}
	return model.DepUpdateTemplate{
		Name:          dep.Name,
		InputTemplate: input_tmplt.GetDepUpTemplate(mod.Module, dep),
	}, nil
}

func (a *Api) createDepIfNotExist(ctx context.Context, mID string, depReq model.DepInput) (bool, string, error) {
	depList, err := a.deploymentHandler.List(ctx, model.DepFilter{ModuleID: mID})
	if err != nil {
		return false, "", err
	}
	if len(depList) == 0 {
		rMod, err := a.moduleHandler.Get(ctx, mID)
		if err != nil {
			return false, "", err
		}
		dir, err := a.moduleHandler.GetIncl(ctx, mID)
		if err != nil {
			return false, "", err
		}
		dID, err := a.deploymentHandler.Create(ctx, rMod.Module, depReq, dir, true)
		if err != nil {
			return false, "", err
		}
		return true, dID, nil
	}
	return false, "", nil
}
