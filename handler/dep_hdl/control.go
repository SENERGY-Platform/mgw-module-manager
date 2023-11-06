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
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
	"strings"
)

func (h *Handler) Start(ctx context.Context, id string, dependencies bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	dep, err := h.storageHandler.ReadDep(ctxWt, id, true)
	if err != nil {
		return err
	}
	if dependencies && len(dep.RequiredDep) > 0 {
		reqDep := make(map[string]model.Deployment)
		if err = h.getReqDep(ctx, dep, reqDep); err != nil {
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
	return h.start(ctx, dep)
}

func (h *Handler) Stop(ctx context.Context, id string, force bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	dep, err := h.storageHandler.ReadDep(ctxWt, id, true)
	if err != nil {
		return err
	}
	if !force && len(dep.DepRequiring) > 0 {
		depReq, err := h.getDepFromIDs(ctx, dep.DepRequiring)
		if err != nil {
			return err
		}
		var reqBy []string
		for _, dr := range depReq {
			if dr.Enabled {
				reqBy = append(reqBy, fmt.Sprintf("%s (%s)", dr.Name, dr.ID))
			}
		}
		if len(reqBy) > 0 {
			return model.NewInternalError(fmt.Errorf("required by: %s", strings.Join(reqBy, ", ")))
		}
	}
	if err = h.stopInstance(ctx, dep); err != nil {
		return err
	}
	if err = h.unloadSecrets(ctx, dep.ID); err != nil {
		return err
	}
	if dep.Enabled {
		dep.Enabled = false
		ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
		defer cf2()
		if err := h.storageHandler.UpdateDep(ctxWt2, nil, dep.DepBase); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) Restart(ctx context.Context, id string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	dep, err := h.storageHandler.ReadDep(ctxWt, id, true)
	if err != nil {
		return err
	}
	if err = h.stopInstance(ctx, dep); err != nil {
		return err
	}
	return h.startInstance(ctx, dep)
}

func (h *Handler) start(ctx context.Context, dep model.Deployment) error {
	if err := h.loadSecrets(ctx, dep); err != nil {
		return err
	}
	if err := h.startInstance(ctx, dep); err != nil {
		return err
	}
	if !dep.Enabled {
		dep.Enabled = true
		ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
		defer cf()
		return h.storageHandler.UpdateDep(ctxWt, nil, dep.DepBase)
	}
	return nil
}
