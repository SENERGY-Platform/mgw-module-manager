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
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
	"time"
)

func (h *Handler) Disable(ctx context.Context, id string, dependencies bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	d, err := h.storageHandler.ReadDep(ctxWt, id, true)
	if err != nil {
		return err
	}
	if len(d.DepRequiring) > 0 {
		depReq, err := h.getDepFromIDs(ctx, d.DepRequiring)
		if err != nil {
			return err
		}
		ok, rdID := allDepStopped(depReq)
		if !ok {
			return model.NewInternalError(fmt.Errorf("required by '%s'", rdID))
		}
	}
	if err = h.disable(ctx, d); err != nil {
		return err
	}
	if dependencies && len(d.RequiredDep) > 0 {
		reqDep := make(map[string]model.Deployment)
		if err = h.getReqDep(ctx, d, reqDep); err != nil {
			return err
		}
		order, err := sorting.GetDepOrder(reqDep)
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
						if err = h.disable(ctx, rd); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

func (h *Handler) disable(ctx context.Context, dep model.Deployment) error {
	if err := h.stopInstance(ctx, dep); err != nil {
		return err
	}
	if err := h.unloadSecrets(ctx, dep.ID); err != nil {
		return err
	}
	dep.Enabled = false
	dep.Updated = time.Now().UTC()
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	return h.storageHandler.UpdateDep(ctxWt, nil, dep.DepBase)
}

func (h *Handler) getDepFromIDs(ctx context.Context, dIDs []string) ([]model.Deployment, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	var dep []model.Deployment
	for _, dID := range dIDs {
		d, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID, false)
		if err != nil {
			return nil, err
		}
		dep = append(dep, d)
	}
	return dep, nil
}

func allDepStopped(dep []model.Deployment) (bool, string) {
	for _, d := range dep {
		if d.Enabled {
			return false, d.ID
		}
	}
	return true, ""
}
