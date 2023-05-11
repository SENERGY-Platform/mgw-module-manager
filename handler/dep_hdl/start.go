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
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
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
		order, err := sorting.GetDepOrder(reqDep)
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
	instance, err := h.getCurrentInst(ctx, dep.ID)
	if err != nil {
		return err
	}
	if err = h.startInstance(ctx, instance.ID); err != nil {
		return err
	}
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	if err = h.storageHandler.UpdateDep(ctxWt, dep.ID, dep.Name, false, dep.Indirect, time.Now().UTC()); err != nil {
		return err
	}
	return nil
}

func (h *Handler) startInstance(ctx context.Context, iID string) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	containers, err := h.storageHandler.ListInstCtr(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), iID, model.CtrFilter{SortOrder: model.Ascending})
	if err != nil {
		return err
	}
	for _, ctr := range containers {
		err = h.cewClient.StartContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), ctr.ID)
		if err != nil {
			return model.NewInternalError(err)
		}
	}
	return nil
}
