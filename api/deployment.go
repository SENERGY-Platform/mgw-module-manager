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
	sm_client "github.com/SENERGY-Platform/mgw-secret-manager/pkg/client"
	"strings"
	"time"
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
	dir, err := a.moduleHandler.GetDir(ctx, mod.ID)
	if err != nil {
		return "", err
	}
	dID, err := a.deploymentHandler.Create(ctx, mod.Module, depInput, dir, false)
	if err != nil {
		return "", err
	}
	return dID, nil
}

func (a *Api) GetDeployments(ctx context.Context, filter model.DepFilter) ([]model.DepBase, error) {
	return a.deploymentHandler.List(ctx, filter)
}

func (a *Api) GetDeployment(ctx context.Context, id string) (model.Deployment, error) {
	return a.deploymentHandler.Get(ctx, id, true, true)
}

func (a *Api) GetDeploymentsHealth(ctx context.Context) (map[string]model.DepHealthInfo, error) {
	deployments, err := a.deploymentHandler.List(ctx, model.DepFilter{})
	if err != nil {
		return nil, err
	}
	instances := make(map[string]model.DepInstance)
	for _, dep := range deployments {
		if dep.Enabled {
			d, err := a.deploymentHandler.Get(ctx, dep.ID, false, true)
			if err != nil {
				return nil, err
			}
			instances[dep.ID] = d.Instance
		}
	}
	return a.depHealthHandler.List(ctx, instances)
}

func (a *Api) GetDeploymentHealth(ctx context.Context, dID string) (model.DepHealthInfo, error) {
	dep, err := a.deploymentHandler.Get(ctx, dID, false, true)
	if err != nil {
		return model.DepHealthInfo{}, err
	}
	if !dep.Enabled {
		return model.DepHealthInfo{}, model.NewInvalidInputError(fmt.Errorf("deployment '%s' not started", dID))
	}
	return a.depHealthHandler.Get(ctx, dep.Instance)
}

func (a *Api) UpdateDeployment(ctx context.Context, dID string, depInput model.DepInput) (string, error) {
	err := a.mu.TryLock(fmt.Sprintf("update deployment '%s'", dID))
	if err != nil {
		return "", model.NewResourceBusyError(err)
	}
	depBase, err := a.deploymentHandler.Get(ctx, dID, false, false)
	if err != nil {
		a.mu.Unlock()
		return "", err
	}
	mod, err := a.moduleHandler.Get(ctx, depBase.Module.ID)
	if err != nil {
		a.mu.Unlock()
		return "", err
	}
	if mod.ID != depBase.Module.ID {
		a.mu.Unlock()
		return "", model.NewInvalidInputError(errors.New("module ID mismatch"))
	}
	jID, err := a.jobHandler.Create(fmt.Sprintf("update deployment '%s'", dID), func(ctx context.Context, cf context.CancelFunc) error {
		defer a.mu.Unlock()
		defer cf()
		err := a.deploymentHandler.Update(ctx, depBase.ID, mod.Module, depInput, "")
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

func (a *Api) DeleteDeployment(ctx context.Context, id string, force bool) error {
	err := a.mu.TryLock(fmt.Sprintf("delete deployment '%s'", id))
	if err != nil {
		return model.NewResourceBusyError(err)
	}
	defer a.mu.Unlock()
	return a.deploymentHandler.Delete(ctx, id, force)
}

func (a *Api) DeleteDeployments(ctx context.Context, ids []string, force bool) error {
	err := a.mu.TryLock(fmt.Sprintf("delete deployments '%s'", strings.Join(ids, ", ")))
	if err != nil {
		return model.NewResourceBusyError(err)
	}
	defer a.mu.Unlock()
	return a.deploymentHandler.DeleteList(ctx, ids, force)
}

func (a *Api) GetDeploymentUpdateTemplate(ctx context.Context, id string) (model.DepUpdateTemplate, error) {
	err := a.mu.TryRLock()
	if err != nil {
		return model.DepUpdateTemplate{}, model.NewResourceBusyError(err)
	}
	defer a.mu.RUnlock()
	dep, err := a.deploymentHandler.Get(ctx, id, true, false)
	if err != nil {
		return model.DepUpdateTemplate{}, err
	}
	mod, err := a.moduleHandler.Get(ctx, dep.Module.ID)
	if err != nil {
		return model.DepUpdateTemplate{}, err
	}
	return model.DepUpdateTemplate{
		Name:          dep.Name,
		InputTemplate: input_tmplt.GetDepUpTemplate(mod.Module, dep),
	}, nil
}

func (a *Api) StartDeployment(ctx context.Context, dID string, dependencies bool) error {
	err := a.mu.TryLock(fmt.Sprintf("start deployment '%s'", dID))
	if err != nil {
		return model.NewResourceBusyError(err)
	}
	defer a.mu.Unlock()
	return a.deploymentHandler.Start(ctx, dID, dependencies)
}

func (a *Api) StartDeployments(ctx context.Context, dIDs []string, dependencies bool) error {
	err := a.mu.TryLock(fmt.Sprintf("start deployments '%s'", strings.Join(dIDs, ", ")))
	if err != nil {
		return model.NewResourceBusyError(err)
	}
	defer a.mu.Unlock()
	return a.deploymentHandler.StartList(ctx, dIDs, dependencies)
}

func (a *Api) StartAllDeployments(ctx context.Context, filter model.DepFilter, dependencies bool) error {
	err := a.mu.TryLock(fmt.Sprintf("start all deployments '%v'", filter))
	if err != nil {
		return model.NewResourceBusyError(err)
	}
	defer a.mu.Unlock()
	return a.deploymentHandler.StartFilter(ctx, filter, dependencies)
}

func (a *Api) StopDeployment(_ context.Context, dID string, dependencies bool) (string, error) {
	err := a.mu.TryLock(fmt.Sprintf("stop deployment '%s'", dID))
	if err != nil {
		return "", model.NewResourceBusyError(err)
	}
	jID, err := a.jobHandler.Create(fmt.Sprintf("stop deployment '%s'", dID), func(ctx context.Context, cf context.CancelFunc) error {
		defer a.mu.Unlock()
		defer cf()
		err := a.deploymentHandler.Stop(ctx, dID, dependencies)
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

func (a *Api) StopDeployments(_ context.Context, dIDs []string, force bool) (string, error) {
	err := a.mu.TryLock(fmt.Sprintf("stop deployment '%s'", strings.Join(dIDs, ",")))
	if err != nil {
		return "", model.NewResourceBusyError(err)
	}
	jID, err := a.jobHandler.Create(fmt.Sprintf("stop deployment '%s'", strings.Join(dIDs, ",")), func(ctx context.Context, cf context.CancelFunc) error {
		defer a.mu.Unlock()
		defer cf()
		err := a.deploymentHandler.StopList(ctx, dIDs, force)
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

func (a *Api) StopAllDeployments(_ context.Context, filter model.DepFilter, force bool) (string, error) {
	err := a.mu.TryLock(fmt.Sprintf("stop all deployment '%v'", filter))
	if err != nil {
		return "", model.NewResourceBusyError(err)
	}
	jID, err := a.jobHandler.Create(fmt.Sprintf("stop all deployment '%v'", filter), func(ctx context.Context, cf context.CancelFunc) error {
		defer a.mu.Unlock()
		defer cf()
		err := a.deploymentHandler.StopFilter(ctx, filter, force)
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

func (a *Api) RestartDeployment(_ context.Context, id string) (string, error) {
	err := a.mu.TryLock(fmt.Sprintf("restart deployment '%s'", id))
	if err != nil {
		return "", model.NewResourceBusyError(err)
	}
	jID, err := a.jobHandler.Create(fmt.Sprintf("restart deployment '%s'", id), func(ctx context.Context, cf context.CancelFunc) error {
		defer a.mu.Unlock()
		defer cf()
		err := a.deploymentHandler.Restart(ctx, id)
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

func (a *Api) RestartDeployments(_ context.Context, ids []string) (string, error) {
	err := a.mu.TryLock(fmt.Sprintf("restart deployment '%s'", strings.Join(ids, ", ")))
	if err != nil {
		return "", model.NewResourceBusyError(err)
	}
	jID, err := a.jobHandler.Create(fmt.Sprintf("restart deployment '%s'", strings.Join(ids, ", ")), func(ctx context.Context, cf context.CancelFunc) error {
		defer a.mu.Unlock()
		defer cf()
		err := a.deploymentHandler.RestartList(ctx, ids)
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

func (a *Api) RestartAllDeployments(_ context.Context, filter model.DepFilter) (string, error) {
	err := a.mu.TryLock(fmt.Sprintf("restart all deployment '%v'", filter))
	if err != nil {
		return "", model.NewResourceBusyError(err)
	}
	jID, err := a.jobHandler.Create(fmt.Sprintf("restart all deployment '%v'", filter), func(ctx context.Context, cf context.CancelFunc) error {
		defer a.mu.Unlock()
		defer cf()
		err := a.deploymentHandler.RestartFilter(ctx, filter)
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

func (a *Api) StartupDeployments(smClient sm_client.Client, delay time.Duration, retries int) error {
	depList, err := a.deploymentHandler.List(context.Background(), model.DepFilter{})
	if err != nil {
		return err
	}
	if len(depList) > 0 {
		err = a.mu.TryLock("deployments startup")
		if err != nil {
			return model.NewResourceBusyError(err)
		}
		_, err = a.jobHandler.Create("deployments startup", func(ctx context.Context, cf context.CancelFunc) error {
			defer a.mu.Unlock()
			defer cf()
			err := a.startupDeployments(ctx, smClient, delay, retries)
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
		dir, err := a.moduleHandler.GetDir(ctx, mID)
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

func (a *Api) startupDeployments(ctx context.Context, smClient sm_client.Client, delay time.Duration, retries int) error {
	if err := waitForSM(ctx, smClient, delay, retries); err != nil {
		return err
	}
	return a.deploymentHandler.StartFilter(ctx, model.DepFilter{Enabled: true}, false)
}

func waitForSM(ctx context.Context, smClient sm_client.Client, delay time.Duration, retries int) error {
	_, err, _ := smClient.GetSecrets(ctx)
	if err != nil {
		ticker := time.NewTicker(delay)
		defer ticker.Stop()
		count := 0
		util.Logger.Warningf("connecting to secret-manager failed (%d/%d): %s", count, retries, err)
		for {
			select {
			case <-ticker.C:
				count += 1
				_, err, _ = smClient.GetSecrets(ctx)
				if err != nil {
					util.Logger.Warningf("connecting to secret-manager failed (%d/%d): %s", count, retries, err)
				} else {
					return nil
				}
				if count >= retries {
					return fmt.Errorf("connecting to secret-manager failed: %s", err)
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}
