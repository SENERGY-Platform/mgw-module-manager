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
		depReq, err := h.getDepFromIDs(ctx, d.DepRequiring)
		if err != nil {
			return err
		}
		ok, rd := allDepStopped(depReq)
		if !ok {
			return model.NewInternalError(fmt.Errorf("required by '%s'", rd.ID))
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
					depReq, err := h.getDepFromIDs(ctx, rd.DepRequiring)
					if err != nil {
						return err
					}
					ok, _ := allDepStopped(depReq)
					if ok {
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
	instance, err := h.getCurrentInst(ctx, dep.ID)
	if err != nil {
		return err
	}
	containers, err := h.storageHandler.ListInstCtr(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), instance.ID, model.CtrFilter{SortOrder: model.Descending})
	if err != nil {
		return err
	}
	for _, ctr := range containers {
		if err = h.stopContainer(ctx, ctr.ID); err != nil {
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

func (h *Handler) getDepFromIDs(ctx context.Context, dIDs []string) ([]*model.Deployment, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	var dep []*model.Deployment
	for _, dID := range dIDs {
		d, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID)
		if err != nil {
			return nil, err
		}
		dep = append(dep, d)
	}
	return dep, nil
}

func allDepStopped(dep []*model.Deployment) (bool, *model.Deployment) {
	for _, d := range dep {
		if !d.Stopped {
			return false, d
		}
	}
	return true, nil
}
