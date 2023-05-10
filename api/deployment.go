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
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
)

func (a *Api) CreateDeployment(ctx context.Context, dr model.DepRequest) (string, error) {
	mod, reqMod, err := a.moduleHandler.GetReq(ctx, dr.ModuleID)
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
		order, err := sorting.GetModOrder(reqMod)
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
			ok, dID, er = a.createDepIfNotExist(ctx, rmID, dr.Dependencies[rmID])
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
	dID, err := a.deploymentHandler.Create(ctx, mod.Module, dr.DepRequestBase, dir, false)
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

func (a *Api) StartDeployment(ctx context.Context, id string) error {
	return a.deploymentHandler.Start(ctx, id)
}

func (a *Api) StopDeployment(_ context.Context, id string, dependencies bool) (string, error) {
	return a.jobHandler.Create(fmt.Sprintf("stop deployment '%s'", id), func(ctx context.Context, cf context.CancelFunc) error {
		defer cf()
		err := a.deploymentHandler.Stop(ctx, id, dependencies)
		if err == nil {
			err = ctx.Err()
		}
		return err
	})
}

func (a *Api) UpdateDeployment(ctx context.Context, id string, dr model.DepRequest) error {
	panic("not implemented")
}

func (a *Api) DeleteDeployment(ctx context.Context, id string, orphans bool) error {
	return a.deploymentHandler.Delete(ctx, id, orphans)
}

func (a *Api) createDepIfNotExist(ctx context.Context, mID string, depReq model.DepRequestBase) (bool, string, error) {
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
