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

package dep_hdl

import (
	"context"
	"fmt"
	"github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-lib/tsort"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
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

func New(storageHandler handler.DepStorageHandler, cfgVltHandler handler.CfgValidationHandler, moduleHandler handler.ModuleHandler, cewClient client.CewClient, dbTimeout time.Duration, httpTimeout time.Duration) *Handler {
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

func (h *Handler) GetTemplate(ctx context.Context, mID string) (*model.DepTemplate, error) {
	m, rms, err := h.moduleHandler.GetWithDep(ctx, mID)
	if err != nil {
		return nil, err
	}
	dt := model.DepTemplate{ModuleID: m.ID, DepTemplateBase: getDepTemplateBase(m)}
	if len(rms) > 0 {
		rdt := make(map[string]model.DepTemplateBase)
		for _, rm := range rms {
			rdt[rm.ID] = getDepTemplateBase(rm)
		}
		dt.Dependencies = rdt
	}
	return &dt, nil
}

func (h *Handler) Update(ctx context.Context, dID string, drb model.DepRequestBase) error {
	panic("not implemented")
}

func (h *Handler) getReqDep(ctx context.Context, dep *model.Deployment, reqDep map[string]*model.Deployment) error {
	ch := context_hdl.New()
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
			return
		}
	} else if len(dep) > 0 {
		for _, d := range dep {
			order = append(order, d.ID)
		}
	}
	return
}

func (h *Handler) getCurrentInst(ctx context.Context, dID string) (*model.DepInstance, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	instances, err := h.storageHandler.ListInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepInstFilter{DepID: dID})
	if err != nil {
		return nil, err
	}
	if len(instances) != 1 {
		return nil, model.NewInternalError(fmt.Errorf("invalid number of instances: %d", len(instances)))
	}
	return h.storageHandler.ReadInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), instances[0].ID)
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

func getDepTemplateBase(m *module.Module) model.DepTemplateBase {
	it := model.DepTemplateBase{
		HostResources: make(map[string]model.DepTemplateHostRes),
		Secrets:       make(map[string]model.DepTemplateSecret),
		Configs:       make(map[string]model.DepTemplateConfig),
		InputGroups:   m.Inputs.Groups,
	}
	for ref, input := range m.Inputs.Resources {
		it.HostResources[ref] = model.DepTemplateHostRes{
			Input:        input,
			HostResource: m.HostResources[ref],
		}
	}
	for ref, input := range m.Inputs.Secrets {
		it.Secrets[ref] = model.DepTemplateSecret{
			Input:  input,
			Secret: m.Secrets[ref],
		}
	}
	for ref, input := range m.Inputs.Configs {
		cv := m.Configs[ref]
		itc := model.DepTemplateConfig{
			Input:    input,
			Default:  cv.Default,
			Options:  cv.Options,
			OptExt:   cv.OptExt,
			Type:     cv.Type,
			TypeOpt:  make(map[string]any),
			DataType: cv.DataType,
			IsList:   cv.IsSlice,
			Required: cv.Required,
		}
		for key, opt := range cv.TypeOpt {
			itc.TypeOpt[key] = opt.Value
		}
		it.Configs[ref] = itc
	}
	return it
}
