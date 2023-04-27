/*
 * Copyright 2023 InfAI (CC SES)
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

package dep_hdl

import (
	"context"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"time"
)

func (h *Handler) Stop(ctx context.Context, id string, dependencies bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	d, err := h.storageHandler.ReadDep(ctxWt, id)
	if err != nil {
		return err
	}
	if len(d.DepRequiring) > 0 {
		extDepReq, err := h.getExtDepReq(ctx, d.DepRequiring, nil)
		if err != nil {
			return err
		}
		if !allDepReqStopped(extDepReq) {
			return model.NewInternalError(errors.New("required by running deployments"))
		}
	}
	if err = h.stop(ctx, d); err != nil {
		return err
	}
	if dependencies && len(d.RequiredDep) > 0 {
		reqDep := make(map[string]*model.Deployment)
		if err = h.getReqDep(ctx, d, reqDep); err != nil {
			return err
		}
		order, err := h.getDepOrder(reqDep)
		if err != nil {
			return model.NewInternalError(err)
		}
		for i := len(order) - 1; i >= 0; i-- {
			rd := reqDep[order[i]]
			if rd.Indirect {
				if len(rd.DepRequiring) > 0 {
					extDepReq, err := h.getExtDepReq(ctx, rd.DepRequiring, reqDep)
					if err != nil {
						return err
					}
					if allDepReqStopped(extDepReq) {
						if err = h.stop(ctx, rd); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

func (h *Handler) stop(ctx context.Context, dep *model.Deployment) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	m, err := h.moduleHandler.Get(ctx, dep.ModuleID)
	if err != nil {
		return err
	}
	order, err := getSrvOrder(m.Services)
	if err != nil {
		return model.NewInternalError(err)
	}
	instance, err := h.getCurrentInst(ctx, dep.ID)
	if err != nil {
		return err
	}
	for i := len(order) - 1; i >= 0; i-- {
		cID, ok := instance.Containers[order[i]]
		if !ok {
			return model.NewInternalError(fmt.Errorf("no container for service reference '%s'", order[i]))
		}
		if err = h.stopContainer(ctx, cID); err != nil {
			return model.NewInternalError(err)
		}
	}
	if err = h.storageHandler.UpdateDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dep.ID, dep.Name, true, dep.Indirect, time.Now().UTC()); err != nil {
		return err
	}
	return nil
}

func (h *Handler) stopContainer(ctx context.Context, cID string) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	jID, err := h.cewClient.StopContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cID)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			c, cf := context.WithTimeout(context.Background(), h.httpTimeout)
			err = h.cewClient.CancelJob(c, jID)
			if err != nil {
				util.Logger.Error(err)
			}
			cf()
			return ctx.Err()
		case <-ticker.C:
			j, err := h.cewClient.GetJob(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), jID)
			if err != nil {
				return err
			}
			if j.Error != nil {
				return fmt.Errorf("%v", j.Error)
			}
			if j.Completed != nil {
				return nil
			}
		}
	}
}

func (h *Handler) getExtDepReq(ctx context.Context, sl []string, m map[string]*model.Deployment) ([]*model.Deployment, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	var ext []*model.Deployment
	for _, s := range sl {
		if _, ok := m[s]; !ok {
			d, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), s)
			if err != nil {
				return nil, err
			}
			ext = append(ext, d)
		}
	}
	return ext, nil
}

func allDepReqStopped(ext []*model.Deployment) bool {
	for _, d := range ext {
		if !d.Stopped {
			return false
		}
	}
	return true
}
