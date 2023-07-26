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
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
	"time"
)

func (h *Handler) Enable(ctx context.Context, id string, dependencies bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	d, err := h.storageHandler.ReadDep(ctxWt, id, true)
	if err != nil {
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
		for _, rdID := range order {
			rd := reqDep[rdID]
			if err = h.enable(ctx, rd); err != nil {
				return err
			}
		}
	}
	return h.enable(ctx, d)
}

func (h *Handler) enable(ctx context.Context, dep model.Deployment) error {
	if err := h.loadSecrets(ctx, dep); err != nil {
		return err
	}
	if err := h.startInstance(ctx, dep); err != nil {
		return err
	}
	dep.Enabled = true
	dep.Updated = time.Now().UTC()
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	return h.storageHandler.UpdateDep(ctxWt, nil, dep.DepBase)
}
