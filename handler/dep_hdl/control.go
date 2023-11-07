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

func (h *Handler) StartList(ctx context.Context, dIDs []string, dependencies bool) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	depMap := make(map[string]model.Deployment)
	for _, dID := range dIDs {
		if _, ok := depMap[dID]; !ok {
			dep, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID, true)
			if err != nil {
				return err
			}
			depMap[dep.ID] = dep
			if dependencies {
				if err = h.getReqDep(ctx, dep, depMap); err != nil {
					return model.NewInternalError(err)
				}
			}

		}
	}
	order, err := sorting.GetDepOrder(depMap)
	if err != nil {
		return err
	}
	for _, dID := range order {
		dep, ok := depMap[dID]
		if !ok {
			return model.NewInternalError(fmt.Errorf("deployment '%s' does not exist", dID))
		}
		if err = h.start(ctx, dep); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) StartFilter(ctx context.Context, filter model.DepFilter, dependencies bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	depList, err := h.storageHandler.ListDep(ctxWt, filter)
	if err != nil {
		return err
	}
	var dIDs []string
	for _, depBase := range depList {
		dIDs = append(dIDs, depBase.ID)
	}
	return h.StartList(ctx, dIDs, dependencies)
}

func (h *Handler) Stop(ctx context.Context, id string, force bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	dep, err := h.storageHandler.ReadDep(ctxWt, id, true)
	if err != nil {
		return err
	}
	if dep.Enabled && !force && len(dep.DepRequiring) > 0 {
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
	return h.stop(ctx, dep)
}

func (h *Handler) StopList(ctx context.Context, dIDs []string, force bool) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	depMap := make(map[string]model.Deployment)
	for _, dID := range dIDs {
		if _, ok := depMap[dID]; !ok {
			dep, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID, true)
			if err != nil {
				return err
			}
			depMap[dep.ID] = dep
		}
	}
	for _, dep := range depMap {
		if dep.Enabled && !force && len(dep.DepRequiring) > 0 {
			var reqBy []string
			for _, drID := range dep.DepRequiring {
				if _, ok := depMap[drID]; !ok {
					dr, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), drID, false)
					if err != nil {
						return err
					}
					if dr.Enabled {
						reqBy = append(reqBy, fmt.Sprintf("%s (%s)", dr.Name, dr.ID))
					}
				}
			}
			if len(reqBy) > 0 {
				return model.NewInternalError(fmt.Errorf("required by: %s", strings.Join(reqBy, ", ")))
			}
		}
	}
	order, err := sorting.GetDepOrder(depMap)
	if err != nil {
		return err
	}
	for i := len(order) - 1; i >= 0; i-- {
		dep, ok := depMap[order[i]]
		if !ok {
			return model.NewInternalError(fmt.Errorf("deployment '%s' does not exist", order[i]))
		}
		if err = h.stop(ctx, dep); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) StopFilter(ctx context.Context, filter model.DepFilter, force bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	depList, err := h.storageHandler.ListDep(ctxWt, filter)
	if err != nil {
		return err
	}
	var dIDs []string
	for _, depBase := range depList {
		dIDs = append(dIDs, depBase.ID)
	}
	return h.StopList(ctx, dIDs, force)
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

func (h *Handler) RestartList(ctx context.Context, dIDs []string) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, dID := range dIDs {
		dep, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID, true)
		if err != nil {
			return err
		}
		if err = h.stopInstance(ctx, dep); err != nil {
			return err
		}
		if err = h.startInstance(ctx, dep); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) RestartFilter(ctx context.Context, filter model.DepFilter) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	depList, err := h.storageHandler.ListDep(ctxWt, filter)
	if err != nil {
		return err
	}
	var dIDs []string
	for _, depBase := range depList {
		dIDs = append(dIDs, depBase.ID)
	}
	return h.RestartList(ctx, dIDs)
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
