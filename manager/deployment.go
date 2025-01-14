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
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/input_tmplt"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
	sm_client "github.com/SENERGY-Platform/mgw-secret-manager/pkg/client"
	"time"
)

func (m *Manager) CreateDeployment(ctx context.Context, id string, depInput lib_model.DepInput, dependencies map[string]lib_model.DepInput) (string, error) {
	metaStr := fmt.Sprintf("create deployment (module_id=%s)", id)
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	modTree, err := m.moduleHandler.GetTree(ctx, id)
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	mod := modTree[id]
	delete(modTree, id)
	if mod.DeploymentType == module.SingleDeployment {
		if l, err := m.deploymentHandler.List(ctx, lib_model.DepFilter{ModuleID: mod.ID}, false, false, false, false); err != nil {
			m.mu.Unlock()
			return "", newApiErr(metaStr, err)
		} else if len(l) > 0 {
			m.mu.Unlock()
			return "", newApiErr(metaStr, lib_model.NewInvalidInputError(errors.New("already deployed")))
		}
	}
	jID, err := m.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		dID, err := m.createDeployment(ctx, mod, modTree, depInput, dependencies)
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			return nil, err
		}
		return dID, nil
	})
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (m *Manager) createDeployment(ctx context.Context, mod model.Module, modTree map[string]model.Module, depInput lib_model.DepInput, dependencies map[string]lib_model.DepInput) (string, error) {
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
					if e := m.deploymentHandler.Delete(ctx, id, true); e != nil {
						util.Logger.Error(e)
					}
				}
			}
		}()
		var ok bool
		var dID string
		for _, rmID := range order {
			ok, dID, er = m.createDepIfNotExist(ctx, rmID, dependencies[rmID])
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
	dID, err := m.deploymentHandler.Create(ctx, mod.Module.Module, depInput, dir, false)
	if err != nil {
		return "", err
	}
	return dID, nil
}

func (m *Manager) GetDeployments(ctx context.Context, filter lib_model.DepFilter, assets, containerInfo bool) (map[string]lib_model.Deployment, error) {
	deployments, err := m.deploymentHandler.List(ctx, filter, true, assets, containerInfo, containerInfo)
	if err != nil {
		return nil, newApiErr(fmt.Sprintf("get deployments (%s assets=%v container_info=%v)", getDepFilterValues(filter), assets, containerInfo), err)
	}
	return deployments, nil
}

func (m *Manager) GetDeployment(ctx context.Context, id string, assets, containerInfo bool) (lib_model.Deployment, error) {
	deployment, err := m.deploymentHandler.Get(ctx, id, true, assets, containerInfo, containerInfo)
	if err != nil {
		return lib_model.Deployment{}, newApiErr(fmt.Sprintf("get deployment (id=%s assets=%v container_info=%v)", id, assets, containerInfo), err)
	}
	return deployment, nil
}

func (m *Manager) UpdateDeployment(ctx context.Context, dID string, depInput lib_model.DepInput) (string, error) {
	metaStr := fmt.Sprintf("update deployment (id=%s)", dID)
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	dep, err := m.deploymentHandler.Get(ctx, dID, false, false, false, false)
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	mod, err := m.moduleHandler.Get(ctx, dep.Module.ID, false)
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	if mod.ID != dep.Module.ID {
		m.mu.Unlock()
		return "", newApiErr(metaStr, errors.New("module ID mismatch"))
	}
	jID, err := m.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		err := m.deploymentHandler.Update(ctx, dep.ID, mod.Module.Module, depInput, "")
		if err == nil {
			err = ctx.Err()
		}
		if err == nil {
			if _, e := m.updateAllAuxDeployments(ctx, dID, mod.Module.Module); e != nil {
				util.Logger.Errorf("%s: %s", metaStr, e)
			}
		}
		return nil, err
	})
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (m *Manager) DeleteDeployment(ctx context.Context, id string, force bool) (string, error) {
	metaStr := fmt.Sprintf("delete deployment (id=%s force=%v)", id, force)
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := m.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		err := m.deploymentHandler.Delete(ctx, id, force)
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			if _, e := m.auxDeploymentHandler.DeleteAll(ctx, id, lib_model.AuxDepFilter{}, true); e != nil {
				util.Logger.Errorf("%s: %s", metaStr, e)
			}
			if e := m.advHandler.DeleteAll(ctx, id); e != nil {
				var nfe *lib_model.NotFoundError
				if !errors.As(e, &nfe) {
					util.Logger.Errorf("%s: %s", metaStr, e)
				}
			}
		}
		return nil, err
	})
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (m *Manager) DeleteDeployments(ctx context.Context, filter lib_model.DepFilter, force bool) (string, error) {
	metaStr := fmt.Sprintf("delete deployments (%s force=%v)", getDepFilterValues(filter), force)
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := m.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		deleted, err := m.deploymentHandler.DeleteAll(ctx, filter, force)
		if err == nil {
			err = ctx.Err()
		}
		for _, dID := range deleted {
			if _, e := m.auxDeploymentHandler.DeleteAll(ctx, dID, lib_model.AuxDepFilter{}, true); e != nil {
				util.Logger.Errorf("%s: %s", metaStr, e)
			}
			if e := m.advHandler.DeleteAll(ctx, dID); e != nil {
				var nfe *lib_model.NotFoundError
				if !errors.As(e, &nfe) {
					util.Logger.Errorf("%s: %s", metaStr, e)
				}
			}
		}
		return deleted, err
	})
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (m *Manager) GetDeploymentUpdateTemplate(ctx context.Context, id string) (lib_model.DepUpdateTemplate, error) {
	metaStr := fmt.Sprintf("get deployment update template (id=%s)", id)
	err := m.mu.TryRLock()
	if err != nil {
		return lib_model.DepUpdateTemplate{}, newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	defer m.mu.RUnlock()
	dep, err := m.deploymentHandler.Get(ctx, id, false, true, false, false)
	if err != nil {
		return lib_model.DepUpdateTemplate{}, newApiErr(metaStr, err)
	}
	mod, err := m.moduleHandler.Get(ctx, dep.Module.ID, false)
	if err != nil {
		return lib_model.DepUpdateTemplate{}, newApiErr(metaStr, err)
	}
	return lib_model.DepUpdateTemplate{
		Name:          dep.Name,
		InputTemplate: input_tmplt.GetDepUpTemplate(mod.Module.Module, dep),
	}, nil
}

func (m *Manager) StartDeployment(ctx context.Context, dID string, dependencies bool) (string, error) {
	metaStr := fmt.Sprintf("start deployment (id=%s dependencies=%v)", dID, dependencies)
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := m.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		started, err := m.deploymentHandler.Start(ctx, dID, dependencies)
		if err == nil {
			err = ctx.Err()
		}
		for _, id := range started {
			if _, e := m.auxDeploymentHandler.StartAll(ctx, id, lib_model.AuxDepFilter{Enabled: lib_model.Yes}); e != nil {
				util.Logger.Errorf("%s: %s", metaStr, e)
			}
		}
		return started, err
	})
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (m *Manager) StartDeployments(ctx context.Context, filter lib_model.DepFilter, dependencies bool) (string, error) {
	metaStr := fmt.Sprintf("start deployments (%s dependencies=%v)", getDepFilterValues(filter), dependencies)
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := m.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		started, err := m.deploymentHandler.StartAll(ctx, filter, dependencies)
		if err == nil {
			err = ctx.Err()
		}
		for _, id := range started {
			if _, e := m.auxDeploymentHandler.StartAll(ctx, id, lib_model.AuxDepFilter{Enabled: lib_model.Yes}); e != nil {
				util.Logger.Errorf("%s: %s", metaStr, e)
			}
		}
		return started, err
	})
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (m *Manager) StopDeployment(ctx context.Context, dID string, force bool) (string, error) {
	metaStr := fmt.Sprintf("stop deployment (id=%s force=%v)", dID, force)
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := m.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		err := m.deploymentHandler.Stop(ctx, dID, force)
		if err == nil {
			err = ctx.Err()
		}
		if err == nil {
			if _, e := m.auxDeploymentHandler.StopAll(ctx, dID, lib_model.AuxDepFilter{}, true); e != nil {
				util.Logger.Errorf("%s: %s", metaStr, e)
			}
			if e := m.advHandler.DeleteAll(ctx, dID); e != nil {
				var nfe *lib_model.NotFoundError
				if !errors.As(e, &nfe) {
					util.Logger.Errorf("%s: %s", metaStr, e)
				}
			}
		}
		return nil, err
	})
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (m *Manager) StopDeployments(ctx context.Context, filter lib_model.DepFilter, force bool) (string, error) {
	metaStr := fmt.Sprintf("stop deployments (%s force=%v)", getDepFilterValues(filter), force)
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := m.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		stopped, err := m.deploymentHandler.StopAll(ctx, filter, force)
		if err == nil {
			err = ctx.Err()
		}
		for _, id := range stopped {
			if _, e := m.auxDeploymentHandler.StopAll(ctx, id, lib_model.AuxDepFilter{}, true); e != nil {
				util.Logger.Errorf("%s: %s", metaStr, e)
			}
			if e := m.advHandler.DeleteAll(ctx, id); e != nil {
				var nfe *lib_model.NotFoundError
				if !errors.As(e, &nfe) {
					util.Logger.Errorf("%s: %s", metaStr, e)
				}
			}
		}
		return stopped, err
	})
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (m *Manager) RestartDeployment(ctx context.Context, id string) (string, error) {
	metaStr := fmt.Sprintf("restart deployment (id=%s)", id)
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := m.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		err := m.deploymentHandler.Restart(ctx, id)
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

func (m *Manager) RestartDeployments(ctx context.Context, filter lib_model.DepFilter) (string, error) {
	metaStr := fmt.Sprintf("restart deployment (%s)", getDepFilterValues(filter))
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return "", newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	jID, err := m.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		restarted, err := m.deploymentHandler.RestartAll(ctx, filter)
		if err == nil {
			err = ctx.Err()
		}
		return restarted, err
	})
	if err != nil {
		m.mu.Unlock()
		return "", newApiErr(metaStr, err)
	}
	return jID, nil
}

func (m *Manager) StartEnabledDeployments(smClient sm_client.Client, delay time.Duration, retries int) error {
	metaStr := fmt.Sprintf("start enabled deployments (delay=%d retries=%d)", delay, retries)
	err := m.mu.TryLock(metaStr)
	if err != nil {
		return newApiErr(metaStr, lib_model.NewResourceBusyError(err))
	}
	_, err = m.jobHandler.Create(context.Background(), metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer m.mu.Unlock()
		defer cf()
		started, err := m.startEnabledDeployments(ctx, smClient, delay, retries, metaStr)
		if err == nil {
			err = ctx.Err()
		}
		return started, err
	})
	if err != nil {
		m.mu.Unlock()
		return newApiErr(metaStr, err)
	}
	return nil
}

func (m *Manager) createDepIfNotExist(ctx context.Context, mID string, depReq lib_model.DepInput) (bool, string, error) {
	depList, err := m.deploymentHandler.List(ctx, lib_model.DepFilter{ModuleID: mID}, false, false, false, false)
	if err != nil {
		return false, "", err
	}
	if len(depList) == 0 {
		rMod, err := m.moduleHandler.Get(ctx, mID, false)
		if err != nil {
			return false, "", err
		}
		dir, err := rMod.GetDirFS()
		if err != nil {
			return false, "", err
		}
		dID, err := m.deploymentHandler.Create(ctx, rMod.Module.Module, depReq, dir, true)
		if err != nil {
			return false, "", err
		}
		return true, dID, nil
	}
	return false, "", nil
}

func (m *Manager) startEnabledDeployments(ctx context.Context, smClient sm_client.Client, delay time.Duration, retries int, metaStr string) ([]string, error) {
	if err := waitForSM(ctx, smClient, delay, retries); err != nil {
		return nil, err
	}
	started, err := m.deploymentHandler.StartAll(ctx, lib_model.DepFilter{Enabled: lib_model.Yes}, false)
	for _, id := range started {
		if _, e := m.auxDeploymentHandler.StartAll(ctx, id, lib_model.AuxDepFilter{Enabled: lib_model.Yes}); e != nil {
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
