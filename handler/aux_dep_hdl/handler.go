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

package aux_dep_hdl

import (
	"context"
	"fmt"
	cew_lib "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"time"
)

type Handler struct {
	storageHandler handler.AuxDepStorageHandler
	cewClient      cew_lib.Api
	dbTimeout      time.Duration
	httpTimeout    time.Duration
	managerID      string
	coreID         string
	moduleNet      string
}

func New(storageHandler handler.AuxDepStorageHandler, cewClient cew_lib.Api, dbTimeout, httpTimeout time.Duration, managerID, moduleNet, coreID string) *Handler {
	return &Handler{
		storageHandler: storageHandler,
		cewClient:      cewClient,
		dbTimeout:      dbTimeout,
		httpTimeout:    httpTimeout,
		managerID:      managerID,
		coreID:         coreID,
		moduleNet:      moduleNet,
	}
}

func (h *Handler) List(ctx context.Context, dID string, filter model.AuxDepFilter, ctrInfo bool) ([]model.AuxDeployment, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	auxDeployments, err := h.storageHandler.List(ctxWt, dID, filter)
	if err != nil {
		return nil, err
	}
	if ctrInfo && len(auxDeployments) > 0 {
		ctrMap, err := h.getContainersMap(ctx, dID)
		if err != nil {
			util.Logger.Error(err)
		} else {
			var auxDeps []model.AuxDeployment
			for _, auxDep := range auxDeployments {
				ctr, ok := ctrMap[auxDep.CtrID]
				if !ok {
					return nil, model.NewInternalError(fmt.Errorf("container '%s' not in map", auxDep.CtrID))
				}
				auxDep.CtrInfo = &model.AuxDepContainer{
					ImageID: ctr.ImageID,
					State:   ctr.State,
				}
				auxDeps = append(auxDeps, auxDep)
			}
			return auxDeps, nil
		}
	}
	return auxDeployments, nil
}

func (h *Handler) Get(ctx context.Context, dID, aID string, ctrInfo bool) (model.AuxDeployment, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	auxDep, err := h.storageHandler.Read(ctxWt, dID, aID)
	if err != nil {
		return model.AuxDeployment{}, err
	}
	if ctrInfo {
		ctxWt2, cf2 := context.WithTimeout(ctx, h.httpTimeout)
		defer cf2()
		ctr, err := h.cewClient.GetContainer(ctxWt2, auxDep.CtrID)
		if err != nil {
			util.Logger.Error(err)
		} else {
			auxDep.CtrInfo = &model.AuxDepContainer{
				ImageID: ctr.ImageID,
				State:   ctr.State,
			}
		}
	}
	return auxDep, nil
}

func (h *Handler) Create(ctx context.Context, auxReq model.AuxDepBase) (string, error) {
	panic("not implemented")
}

func (h *Handler) Update(ctx context.Context, aID string, sdReq model.AuxDepBase) error {
	panic("not implemented")
}

func (h *Handler) Delete(ctx context.Context, dID, aID string) error {
	panic("not implemented")
}

func (h *Handler) DeleteAll(ctx context.Context, dID string, filter model.AuxDepFilter) error {
	panic("not implemented")
}

func (h *Handler) Start(ctx context.Context, dID, aID string) error {
	panic("not implemented")
}

func (h *Handler) StartAll(ctx context.Context, dID string, filter model.AuxDepFilter) error {
	panic("not implemented")
}

func (h *Handler) Stop(ctx context.Context, dID, aID string) error {
	panic("not implemented")
}

func (h *Handler) StopAll(ctx context.Context, dID string, filter model.AuxDepFilter) error {
	panic("not implemented")
}

func (h *Handler) getContainersMap(ctx context.Context, dID string) (map[string]cew_model.Container, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	containers, err := h.cewClient.GetContainers(ctxWt, cew_model.ContainerFilter{Labels: map[string]string{handler.ManagerIDLabel: h.managerID, handler.DeploymentIDLabel: dID}})
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	ctrMap := make(map[string]cew_model.Container)
	for _, container := range containers {
		ctrMap[container.ID] = container
	}
	return ctrMap, nil
}
