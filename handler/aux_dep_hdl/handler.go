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
	cew_lib "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/naming_hdl"
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
	depHostPath    string
}

func New(storageHandler handler.AuxDepStorageHandler, cewClient cew_lib.Api, dbTimeout, httpTimeout time.Duration, managerID, moduleNet, coreID, depHostPath string) *Handler {
	return &Handler{
		storageHandler: storageHandler,
		cewClient:      cewClient,
		dbTimeout:      dbTimeout,
		httpTimeout:    httpTimeout,
		managerID:      managerID,
		coreID:         coreID,
		moduleNet:      moduleNet,
		depHostPath:    depHostPath,
	}
}

func (h *Handler) List(ctx context.Context, dID string, filter lib_model.AuxDepFilter, assets, containerInfo bool) (map[string]lib_model.AuxDeployment, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	auxDeployments, err := h.storageHandler.ListAuxDep(ctxWt, dID, filter, assets)
	if err != nil {
		return nil, err
	}
	if containerInfo && len(auxDeployments) > 0 {
		ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
		defer cf2()
		ctrList, err := h.cewClient.GetContainers(ctxWt2, cew_model.ContainerFilter{Labels: map[string]string{naming_hdl.ManagerIDLabel: h.managerID, naming_hdl.DeploymentIDLabel: dID}})
		if err != nil {
			util.Logger.Errorf("could not retrieve containers: %s", err.Error())
			return auxDeployments, nil
		}
		ctrMap := make(map[string]cew_model.Container)
		for _, ctr := range ctrList {
			ctrMap[ctr.ID] = ctr
		}
		withCtrInfo := make(map[string]lib_model.AuxDeployment)
		for aID, auxDeployment := range auxDeployments {
			ctr, ok := ctrMap[auxDeployment.Container.ID]
			if ok {
				auxDeployment.Container.Info = &lib_model.ContainerInfo{
					ImageID: ctr.ImageID,
					State:   ctr.State,
				}
			} else {
				util.Logger.Warningf("aux deployment '%s' missing container '%s'", aID, auxDeployment.Container.ID)
			}
			withCtrInfo[aID] = auxDeployment
		}
		return withCtrInfo, nil
	}
	return auxDeployments, nil
}

func (h *Handler) Get(ctx context.Context, aID string, assets, containerInfo bool) (lib_model.AuxDeployment, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	auxDeployment, err := h.storageHandler.ReadAuxDep(ctxWt, aID, assets)
	if err != nil {
		return lib_model.AuxDeployment{}, err
	}
	if containerInfo {
		ctxWt2, cf2 := context.WithTimeout(ctx, h.httpTimeout)
		defer cf2()
		ctr, err := h.cewClient.GetContainer(ctxWt2, auxDeployment.Container.ID)
		if err != nil {
			util.Logger.Error(err)
		} else {
			auxDeployment.Container.Info = &lib_model.ContainerInfo{
				ImageID: ctr.ImageID,
				State:   ctr.State,
			}
		}
	}
	return auxDeployment, nil
}
