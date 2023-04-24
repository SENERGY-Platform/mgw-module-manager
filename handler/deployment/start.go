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
	"time"
)

func (h *Handler) Start(ctx context.Context, id string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	d, err := h.storageHandler.ReadDep(ctxWt, id)
	if err != nil {
		return err
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
		for _, rdID := range order {
			rd := reqDep[rdID]
			if err = h.start(ctx, rd); err != nil {
				return err
			}
		}
	}
	return h.start(ctx, d)
}

func (h *Handler) start(ctx context.Context, dep *model.Deployment) error {
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
	instances, err := h.storageHandler.ListInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepInstFilter{DepID: dep.ID})
	if err != nil {
		return err
	}
	if len(instances) != 1 {
		return model.NewInternalError(fmt.Errorf("invalid number of instances: %d", len(instances)))
	}
	instance, err := h.storageHandler.ReadInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), instances[0].ID)
	if err != nil {
		return err
	}
	for _, sRef := range order {
		cID, ok := instance.Containers[sRef]
		if !ok {
			return model.NewInternalError(fmt.Errorf("no container for service reference '%s'", sRef))
		}
		err = h.cewClient.StartContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cID)
		if err != nil {
			return err
		}
	}
	if err = h.storageHandler.UpdateDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dep.ID, dep.Name, false, dep.Indirect, time.Now().UTC()); err != nil {
		return err
	}
	return nil
}