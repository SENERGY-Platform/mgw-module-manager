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
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-lib/tsort"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/ctx_handler"
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

func (h *Handler) Delete(ctx context.Context, id string) error {
	ch := ctx_handler.New()
	defer ch.CancelAll()
	d, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), id)
	if err != nil {
		return err
	}
	depReqBy, err := h.storageHandler.ListDepRequiring(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), d.ID)
	if err != nil {
		return err
	}
	if len(depReqBy) > 0 {
		return model.NewInternalError(fmt.Errorf("deplyoment is required by '%d' deplyoments", len(depReqBy)))
	}
	il, err := h.storageHandler.ListInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepInstFilter{DepID: d.ID})
	if err != nil {
		return err
	}
	for _, im := range il {
		i, err := h.storageHandler.ReadInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), im.ID)
		if err != nil {
			return err
		}
		for _, cID := range i.Containers {
			err := h.cewClient.RemoveContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cID)
			if err != nil {
				return err
			}
		}
	}
	volumes, err := h.cewClient.GetVolumes(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cew_model.VolumeFilter{Labels: map[string]string{"d_id": d.ID}})
	if err != nil {
		return err
	}
	for _, volume := range volumes {
		err := h.cewClient.RemoveVolume(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), volume.Name)
		if err != nil {
			return err
		}
	}
	return h.storageHandler.DeleteDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), id)
}

func (h *Handler) Update(ctx context.Context, dID string, drb model.DepRequestBase) error {
	panic("not implemented")
}

func (h *Handler) Start(ctx context.Context, id string) error {
	panic("not implemented")
}

func (h *Handler) Stop(ctx context.Context, id string) error {
	panic("not implemented")
}

func getModOrder(modules map[string]*module.Module) (order []string, err error) {
	if len(modules) > 1 {
		nodes := make(tsort.Nodes)
		for _, m := range modules {
			if len(m.Dependencies) > 0 {
				reqIDs := make(map[string]struct{})
				for i := range m.Dependencies {
					reqIDs[i] = struct{}{}
				}
				nodes.Add(m.ID, reqIDs, nil)
			}
		}
		order, err = tsort.GetTopOrder(nodes)
		if err != nil {
			return nil, err
		}
	} else if len(modules) > 0 {
		for _, m := range modules {
			order = append(order, m.ID)
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
