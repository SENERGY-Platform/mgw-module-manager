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
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/input_tmplt"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
	sm_client "github.com/SENERGY-Platform/mgw-secret-manager/pkg/client"
	"time"
)

func (a *Api) CreateDeployment(ctx context.Context, id string, depInput lib_model.DepInput, dependencies map[string]lib_model.DepInput) (string, error) {
	metaStr := fmt.Sprintf("create deployment (module_id=%s)", id)
	err := a.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	modTree, err := a.moduleHandler.GetTree(ctx, id)
	if err != nil {
		a.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	mod := modTree[id]
	delete(modTree, id)
	if mod.DeploymentType == module.SingleDeployment {
		if l, err := a.deploymentHandler.List(ctx, lib_model.DepFilter{ModuleID: mod.ID}, false, false, false, false); err != nil {
			a.mu.Unlock()
			return "", newApiErr(metaStr, err)
		} else if len(l) > 0 {
			a.mu.Unlock()
			return "", newApiErr(metaStr, lib_model.NewInvalidInputError(errors.New("already deployed")))
		}
	}
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer a.mu.Unlock()
		defer cf()
		dID, err := a.createDeployment(ctx, mod, modTree, depInput, dependencies)
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			return nil, err
		}
		return dID, nil
	})
	if err != nil {
		a.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (a *Api) createDeployment(ctx context.Context, mod model.Module, modTree map[string]model.Module, depInput lib_model.DepInput, dependencies map[string]lib_model.DepInput) (string, error) {
	if len(modTree) > 0 {
		modMap := make(map[string]*module.Module)
		for _, m := range modTree {
			modMap[m.ID] = m.Module.Module
		}
		order, err := sorting.GetModOrder(modMap)
		if err != nil {
			return "", lib_model.NewInternalError(err)
		}
		var er error
		var dIDs []string
		defer func() {
			if er != nil {
				for _, id := range dIDs {
					if e := a.deploymentHandler.Delete(ctx, id, true); e != nil {
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
	dir, err := mod.GetDirFS()
	if err != nil {
		return "", err
	}
	dID, err := a.deploymentHandler.Create(ctx, mod.Module.Module, depInput, dir, false)
	if err != nil {
		return "", err
	}
	return dID, nil
}

func (a *Api) GetDeployments(ctx context.Context, filter lib_model.DepFilter, assets, containerInfo bool) (map[string]lib_model.Deployment, error) {
	deployments, err := a.deploymentHandler.List(ctx, filter, true, assets, containerInfo, containerInfo)
	if err != nil {
		return nil, newApiErr(fmt.Sprintf("get deployments (%s assets=%v container_info=%v)", getDepFilterValues(filter), assets, containerInfo), err)
	}
	return deployments, nil
}

func (a *Api) GetDeployment(ctx context.Context, id string, assets, containerInfo bool) (lib_model.Deployment, error) {
	deployment, err := a.deploymentHandler.Get(ctx, id, true, assets, containerInfo, containerInfo)
	if err != nil {
		return lib_model.Deployment{}, newApiErr(fmt.Sprintf("get deployment (id=%s assets=%v container_info=%v)", id, assets, containerInfo), err)
	}
	return deployment, nil
}

func (a *Api) UpdateDeployment(ctx context.Context, dID string, depInput lib_model.DepInput) (string, error) {
	metaStr := fmt.Sprintf("update deployment (id=%s)", dID)
	err := a.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	dep, err := a.deploymentHandler.Get(ctx, dID, false, false, false, false)
	if err != nil {
		a.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	mod, err := a.moduleHandler.Get(ctx, dep.Module.ID, false)
	if err != nil {
		a.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	if mod.ID != dep.Module.ID {
		a.mu.Unlock()
		return "", newApiErr(metaStr, errors.New("module ID mismatch"))
	}
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer a.mu.Unlock()
		defer cf()
		err := a.deploymentHandler.Update(ctx, dep.ID, mod.Module.Module, depInput, "")
		if err == nil {
			err = ctx.Err()
		}
		if err == nil {
			if _, e := a.updateAllAuxDeployments(ctx, dID, mod.Module.Module); e != nil {
				util.Logger.Errorf("%s: %s", metaStr, e)
			}
		}
		return nil, err
	})
	if err != nil {
		a.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (a *Api) DeleteDeployment(ctx context.Context, id string, force bool) (string, error) {
	metaStr := fmt.Sprintf("delete deployment (id=%s force=%v)", id, force)
	err := a.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer a.mu.Unlock()
		defer cf()
		err := a.deploymentHandler.Delete(ctx, id, force)
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			if _, e := a.auxDeploymentHandler.DeleteAll(ctx, id, lib_model.AuxDepFilter{}, true); e != nil {
				util.Logger.Errorf("%s: %s", metaStr, e)
			}
		}
		return nil, err
	})
	if err != nil {
		a.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (a *Api) DeleteDeployments(ctx context.Context, filter lib_model.DepFilter, force bool) (string, error) {
	metaStr := fmt.Sprintf("delete deployments (%s force=%v)", getDepFilterValues(filter), force)
	err := a.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer a.mu.Unlock()
		defer cf()
		deleted, err := a.deploymentHandler.DeleteAll(ctx, filter, force)
		if err == nil {
			err = ctx.Err()
		}
		for _, dID := range deleted {
			if _, e := a.auxDeploymentHandler.DeleteAll(ctx, dID, lib_model.AuxDepFilter{}, true); e != nil {
				util.Logger.Errorf("%s: %s", metaStr, e)
			}
		}
		return deleted, err
	})
	if err != nil {
		a.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (a *Api) GetDeploymentUpdateTemplate(ctx context.Context, id string) (lib_model.DepUpdateTemplate, error) {
	metaStr := fmt.Sprintf("get deployment update template (id=%s)", id)
	err := a.mu.TryRLock()
	if err != nil {
		return lib_model.DepUpdateTemplate{}, newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	defer a.mu.RUnlock()
	dep, err := a.deploymentHandler.Get(ctx, id, false, true, false, false)
	if err != nil {
		return lib_model.DepUpdateTemplate{}, newApiErr(metaStr, err)
	}
	mod, err := a.moduleHandler.Get(ctx, dep.Module.ID, false)
	if err != nil {
		return lib_model.DepUpdateTemplate{}, newApiErr(metaStr, err)
	}
	return lib_model.DepUpdateTemplate{
		Name:          dep.Name,
		InputTemplate: input_tmplt.GetDepUpTemplate(mod.Module.Module, dep),
	}, nil
}

func (a *Api) StartDeployment(ctx context.Context, dID string, dependencies bool) (string, error) {
	metaStr := fmt.Sprintf("start deployment (id=%s dependencies=%v)", dID, dependencies)
	err := a.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer a.mu.Unlock()
		defer cf()
		started, err := a.deploymentHandler.Start(ctx, dID, dependencies)
		if err == nil {
			err = ctx.Err()
		}
		for _, id := range started {
			if _, e := a.auxDeploymentHandler.StartAll(ctx, id, lib_model.AuxDepFilter{Enabled: lib_model.Yes}); e != nil {
				util.Logger.Errorf("%s: %s", metaStr, e)
			}
		}
		return started, err
	})
	if err != nil {
		a.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (a *Api) StartDeployments(ctx context.Context, filter lib_model.DepFilter, dependencies bool) (string, error) {
	metaStr := fmt.Sprintf("start deployments (%s dependencies=%v)", getDepFilterValues(filter), dependencies)
	err := a.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer a.mu.Unlock()
		defer cf()
		started, err := a.deploymentHandler.StartAll(ctx, filter, dependencies)
		if err == nil {
			err = ctx.Err()
		}
		for _, id := range started {
			if _, e := a.auxDeploymentHandler.StartAll(ctx, id, lib_model.AuxDepFilter{Enabled: lib_model.Yes}); e != nil {
				util.Logger.Errorf("%s: %s", metaStr, e)
			}
		}
		return started, err
	})
	if err != nil {
		a.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (a *Api) StopDeployment(ctx context.Context, dID string, force bool) (string, error) {
	metaStr := fmt.Sprintf("stop deployment (id=%s force=%v)", dID, force)
	err := a.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer a.mu.Unlock()
		defer cf()
		err := a.deploymentHandler.Stop(ctx, dID, force)
		if err == nil {
			err = ctx.Err()
		}
		if err == nil {
			if _, e := a.auxDeploymentHandler.StopAll(ctx, dID, lib_model.AuxDepFilter{}, true); e != nil {
				util.Logger.Errorf("%s: %s", metaStr, e)
			}
		}
		return nil, err
	})
	if err != nil {
		a.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (a *Api) StopDeployments(ctx context.Context, filter lib_model.DepFilter, force bool) (string, error) {
	metaStr := fmt.Sprintf("stop deployments (%s force=%v)", getDepFilterValues(filter), force)
	err := a.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer a.mu.Unlock()
		defer cf()
		stopped, err := a.deploymentHandler.StopAll(ctx, filter, force)
		if err == nil {
			err = ctx.Err()
		}
		for _, id := range stopped {
			if _, e := a.auxDeploymentHandler.StopAll(ctx, id, lib_model.AuxDepFilter{}, true); e != nil {
				util.Logger.Errorf("%s: %s", metaStr, e)
			}
		}
		return stopped, err
	})
	if err != nil {
		a.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (a *Api) RestartDeployment(ctx context.Context, id string) (string, error) {
	metaStr := fmt.Sprintf("restart deployment (id=%s)", id)
	err := a.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer a.mu.Unlock()
		defer cf()
		err := a.deploymentHandler.Restart(ctx, id)
		if err == nil {
			err = ctx.Err()
		}
		return nil, err
	})
	if err != nil {
		a.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (a *Api) RestartDeployments(ctx context.Context, filter lib_model.DepFilter) (string, error) {
	metaStr := fmt.Sprintf("restart deployment (%s)", getDepFilterValues(filter))
	err := a.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer a.mu.Unlock()
		defer cf()
		restarted, err := a.deploymentHandler.RestartAll(ctx, filter)
		if err == nil {
			err = ctx.Err()
		}
		return restarted, err
	})
	if err != nil {
		a.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (a *Api) StartEnabledDeployments(smClient sm_client.Client, delay time.Duration, retries int) error {
	metaStr := fmt.Sprintf("start enabled deployments (delay=%d retries=%d)", delay, retries)
	err := a.mu.TryLock(metaStr)
	if err != nil {
		return newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	_, err = a.jobHandler.Create(context.Background(), metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer a.mu.Unlock()
		defer cf()
		started, err := a.startEnabledDeployments(ctx, smClient, delay, retries, metaStr)
		if err == nil {
			err = ctx.Err()
		}
		return started, err
	})
	if err != nil {
		a.mu.Unlock()
		return newApiErr(metaStr, err)
	}
	return nil
}

func (a *Api) createDepIfNotExist(ctx context.Context, mID string, depReq lib_model.DepInput) (bool, string, error) {
	depList, err := a.deploymentHandler.List(ctx, lib_model.DepFilter{ModuleID: mID}, false, false, false, false)
	if err != nil {
		return false, "", err
	}
	if len(depList) == 0 {
		rMod, err := a.moduleHandler.Get(ctx, mID, false)
		if err != nil {
			return false, "", err
		}
		dir, err := rMod.GetDirFS()
		if err != nil {
			return false, "", err
		}
		dID, err := a.deploymentHandler.Create(ctx, rMod.Module.Module, depReq, dir, true)
		if err != nil {
			return false, "", err
		}
		return true, dID, nil
	}
	return false, "", nil
}

func (a *Api) startEnabledDeployments(ctx context.Context, smClient sm_client.Client, delay time.Duration, retries int, metaStr string) ([]string, error) {
	if err := waitForSM(ctx, smClient, delay, retries); err != nil {
		return nil, err
	}
	started, err := a.deploymentHandler.StartAll(ctx, lib_model.DepFilter{Enabled: lib_model.Yes}, false)
	for _, id := range started {
		if _, e := a.auxDeploymentHandler.StartAll(ctx, id, lib_model.AuxDepFilter{Enabled: lib_model.Yes}); e != nil {
			util.Logger.Errorf("%s: %s", metaStr, e)
		}
	}
	return started, err
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

func getDepFilterValues(filter lib_model.DepFilter) string {
	return fmt.Sprintf("ids=%v enabled=%v name=%s indirect=%v module_id=%s", filter.IDs, filter.Enabled, filter.Name, filter.Indirect, filter.ModuleID)
}
