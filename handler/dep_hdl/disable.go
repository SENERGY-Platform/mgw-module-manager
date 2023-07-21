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
	sm_model "github.com/SENERGY-Platform/mgw-secret-manager/pkg/api_model"
	"time"
)

func (h *Handler) Disable(ctx context.Context, id string, dependencies bool) error {
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
	instance, err := h.getCurrentInst(ctx, dep.ID)
	if err != nil {
		return err
	}
	if err = h.stopInstance(ctx, instance.ID); err != nil {
		return err
	}
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, depSecret := range dep.Secrets {
		for _, variant := range depSecret.Variants {
			if variant.AsMount {
				err, _ = h.smClient.UnloadPathVariant(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), sm_model.SecretVariantRequest{
					ID:        depSecret.ID,
					Item:      variant.Item,
					Reference: dep.ID,
				})
				if err != nil {
					return model.NewInternalError(fmt.Errorf("unloading path variant for secret '%s' failed: %s", depSecret.ID, err))
				}
			}
		}
	}
	dep.Enabled = false
	dep.Updated = time.Now().UTC()
	if err = h.storageHandler.UpdateDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), nil, dep.DepBase); err != nil {
		return err
	}
	return nil
}

func (h *Handler) getDepFromIDs(ctx context.Context, dIDs []string) ([]model.Deployment, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	var dep []model.Deployment
	for _, dID := range dIDs {
		d, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID)
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