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
	sm_model "github.com/SENERGY-Platform/mgw-secret-manager/pkg/api_model"
	"time"
)

func (h *Handler) Enable(ctx context.Context, id string) error {
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
			if err = h.enable(ctx, rd); err != nil {
				return err
			}
		}
	}
	return h.enable(ctx, d)
}

func (h *Handler) enable(ctx context.Context, dep *model.Deployment) error {
	instance, err := h.getCurrentInst(ctx, dep.ID)
	if err != nil {
		return err
	}
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, depSecret := range dep.Secrets {
		for _, variant := range depSecret.Variants {
			if variant.AsMount {
				err, _ = h.smClient.LoadPathVariant(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), sm_model.SecretVariantRequest{
					ID:        depSecret.ID,
					Item:      variant.Item,
					Reference: dep.ID,
				})
				if err != nil {
					return model.NewInternalError(err)
				}
			}
		}
	}
	if err = h.startInstance(ctx, instance.ID); err != nil {
		return err
	}
	if err = h.storageHandler.UpdateDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dep.ID, dep.Name, dep.Dir, true, dep.Indirect, time.Now().UTC()); err != nil {
		return err
	}
	return nil
}
