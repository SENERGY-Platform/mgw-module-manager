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

package deployment

import (
	"context"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/ctx_handler"
	"strings"
	"time"
)

func (h *Handler) Stop(ctx context.Context, id string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	d, err := h.storageHandler.ReadDep(ctxWt, id)
	if err != nil {
		return err
	}
	if len(d.DepRequiring) > 0 {
		return model.NewInternalError(fmt.Errorf("deplyoment is required by: %s", strings.Join(d.DepRequiring, ", ")))
	}
	if len(d.RequiredDep) > 0 {
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
			if isNotReq(rd.DepRequiring, reqDep, d.ID) {
				if err = h.stop(ctx, rd); err != nil {
					return err
				}
			}
		}
	}
	return h.stop(ctx, d)
}

func (h *Handler) stop(ctx context.Context, dep *model.Deployment) error {
	ch := ctx_handler.New()
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
		_, err = h.cewClient.StopContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cID)
		if err != nil {
			return err
		}
	}
	if err = h.storageHandler.UpdateDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dep.ID, dep.Name, true, dep.Indirect, time.Now().UTC()); err != nil {
		return err
	}
	return nil
}

func (h *Handler) getExtDepReq(ctx context.Context, sl []string, m map[string]*model.Deployment) ([]*model.Deployment, error) {
	ch := ctx_handler.New()
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
