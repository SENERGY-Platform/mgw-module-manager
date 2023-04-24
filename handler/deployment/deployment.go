/*
 * Copyright 2022 InfAI (CC SES)
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
	"github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-lib/tsort"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/ctx_handler"
	"strings"
	"time"
)

type Handler struct {
	storageHandler handler.DepStorageHandler
	cfgVltHandler  handler.CfgValidationHandler
	moduleHandler  handler.ModuleHandler
	cewClient      client.CewClient
	dbTimeout      time.Duration
	httpTimeout    time.Duration
}

func NewHandler(storageHandler handler.DepStorageHandler, cfgVltHandler handler.CfgValidationHandler, moduleHandler handler.ModuleHandler, cewClient client.CewClient, dbTimeout time.Duration, httpTimeout time.Duration) *Handler {
	return &Handler{
		storageHandler: storageHandler,
		cfgVltHandler:  cfgVltHandler,
		moduleHandler:  moduleHandler,
		cewClient:      cewClient,
		dbTimeout:      dbTimeout,
		httpTimeout:    httpTimeout,
	}
}

func (h *Handler) List(ctx context.Context, filter model.DepFilter) ([]model.DepMeta, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	return h.storageHandler.ListDep(ctxWt, filter)
}

func (h *Handler) Get(ctx context.Context, id string) (*model.Deployment, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	return h.storageHandler.ReadDep(ctxWt, id)
}

func (h *Handler) Update(ctx context.Context, dID string, drb model.DepRequestBase) error {
	panic("not implemented")
}

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

func isNotReq(sl []string, m map[string]*model.Deployment, id string) bool {
	for _, s := range sl {
		if s == id {
			continue
		}
		if _, ok := m[s]; !ok {
			return false
		}
	}
	return true
}

func (h *Handler) getReqDep(ctx context.Context, dep *model.Deployment, reqDep map[string]*model.Deployment) error {
	ch := ctx_handler.New()
	defer ch.CancelAll()
	for _, dID := range dep.RequiredDep {
		if _, ok := reqDep[dID]; !ok {
			d, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID)
			if err != nil {
				return err
			}
			reqDep[dID] = d
			if err = h.getReqDep(ctx, d, reqDep); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) getDepOrder(dep map[string]*model.Deployment) (order []string, err error) {
	if len(dep) > 1 {
		nodes := make(tsort.Nodes)
		for _, d := range dep {
			if len(d.RequiredDep) > 0 {
				reqIDs := make(map[string]struct{})
				for _, i := range d.RequiredDep {
					reqIDs[i] = struct{}{}
				}
				nodes.Add(d.ID, reqIDs, nil)
			}
		}
		order, err = tsort.GetTopOrder(nodes)
		if err != nil {
			return nil, err
		}
	} else if len(dep) > 0 {
		for _, d := range dep {
			order = append(order, d.ID)
		}
	}
	return
}

func getSrvOrder(services map[string]*module.Service) (order []string, err error) {
	if len(services) > 1 {
		nodes := make(tsort.Nodes)
		for ref, srv := range services {
			nodes.Add(ref, srv.RequiredSrv, srv.RequiredBySrv)
		}
		order, err = tsort.GetTopOrder(nodes)
		if err != nil {
			return nil, err
		}
	} else if len(services) > 0 {
		for ref := range services {
			order = append(order, ref)
		}
	}
	return
}
