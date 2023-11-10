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
	cew_lib "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	hm_lib "github.com/SENERGY-Platform/mgw-host-manager/lib"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/naming_hdl"
	sm_client "github.com/SENERGY-Platform/mgw-secret-manager/pkg/client"
	"io/fs"
	"os"
	"path"
	"time"
)

type Handler struct {
	storageHandler handler.DepStorageHandler
	cfgVltHandler  handler.CfgValidationHandler
	cewClient      cew_lib.Api
	hmClient       hm_lib.Api
	smClient       sm_client.Client
	dbTimeout      time.Duration
	httpTimeout    time.Duration
	wrkSpcPath     string
	depHostPath    string
	secHostPath    string
	managerID      string
	coreID         string
	moduleNet      string
}

func New(storageHandler handler.DepStorageHandler, cfgVltHandler handler.CfgValidationHandler, cewClient cew_lib.Api, hmClient hm_lib.Api, smClient sm_client.Client, dbTimeout time.Duration, httpTimeout time.Duration, workspacePath, depHostPath, secHostPath, managerID, moduleNet, coreID string) *Handler {
	return &Handler{
		storageHandler: storageHandler,
		cfgVltHandler:  cfgVltHandler,
		cewClient:      cewClient,
		hmClient:       hmClient,
		smClient:       smClient,
		dbTimeout:      dbTimeout,
		httpTimeout:    httpTimeout,
		wrkSpcPath:     workspacePath,
		depHostPath:    depHostPath,
		secHostPath:    secHostPath,
		managerID:      managerID,
		coreID:         coreID,
		moduleNet:      moduleNet,
	}
}

type secretVariant struct {
	Item  *string
	Path  string
	AsEnv bool
	Value string
}

type secret struct {
	ID       string
	Variants map[string]secretVariant
}

func (h *Handler) InitWorkspace(perm fs.FileMode) error {
	if !path.IsAbs(h.wrkSpcPath) {
		return fmt.Errorf("workspace path must be absolute")
	}
	if err := os.MkdirAll(h.wrkSpcPath, perm); err != nil {
		return err
	}
	return nil
}

func (h *Handler) List(ctx context.Context, filter model.DepFilter, dependencyInfo, assets, containers, containerInfo bool) (map[string]model.Deployment, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	if containerInfo {
		containers = true
	}
	deployments, err := h.storageHandler.ListDep(ctxWt, filter, dependencyInfo, assets, containers)
	if err != nil {
		return nil, err
	}
	if containerInfo {
		ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
		defer cf2()
		ctrList, err := h.cewClient.GetContainers(ctxWt2, cew_model.ContainerFilter{Labels: map[string]string{naming_hdl.ManagerIDLabel: h.managerID}})
		if err != nil {
			util.Logger.Errorf("could not retrieve containers: %s", err.Error())
			return deployments, nil
		}
		withCtrInfo := make(map[string]model.Deployment)
		for dID, deployment := range deployments {
			if deployment.Enabled {
				deployment.State, deployment.Containers = getDepHealthAndCtrInfo(dID, deployment.Containers, ctrList)
			}
			withCtrInfo[dID] = deployment
		}
		return withCtrInfo, nil
	}
	return deployments, nil
}

func (h *Handler) Get(ctx context.Context, id string, dependencyInfo, assets, containers, containerInfo bool) (model.Deployment, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	if containerInfo {
		containers = true
	}
	deployment, err := h.storageHandler.ReadDep(ctxWt, id, dependencyInfo, assets, containers)
	if err != nil {
		return model.Deployment{}, err
	}
	if containerInfo && deployment.Enabled {
		ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
		defer cf2()
		ctrList, err := h.cewClient.GetContainers(ctxWt2, cew_model.ContainerFilter{Labels: map[string]string{naming_hdl.ManagerIDLabel: h.managerID, naming_hdl.DeploymentIDLabel: id}})
		if err != nil {
			util.Logger.Errorf("could not retrieve containers: %s", err.Error())
			return deployment, nil
		}
		deployment.State, deployment.Containers = getDepHealthAndCtrInfo(deployment.ID, deployment.Containers, ctrList)
	}
	return deployment, nil
}

func (h *Handler) getModDependencyDeployments(ctx context.Context, modDependencies map[string]string) (map[string]model.Deployment, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	m := make(map[string]model.Deployment)
	for mID := range modDependencies {
		deployments, err := h.storageHandler.ListDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepFilter{ModuleID: mID}, false, false, true)
		if err != nil {
			return nil, err
		}
		if len(deployments) == 0 {
			return nil, model.NewInternalError(fmt.Errorf("dependency '%s' not deployed", mID))
		}
		if len(deployments) > 1 {
			return nil, model.NewInternalError(fmt.Errorf("dependency '%s' has multiple deployments", mID))
		}
		for _, dep := range deployments {
			m[mID] = dep
			break
		}
	}
	return m, nil
}

func getDepHealthAndCtrInfo(dID string, depContainers map[string]model.DepContainer, ctrList []cew_model.Container) (*model.HealthState, map[string]model.DepContainer) {
	ctrMap := make(map[string]cew_model.Container)
	for _, ctr := range ctrList {
		ctrMap[ctr.ID] = ctr
	}
	var state model.HealthState
	withCtrInfo := make(map[string]model.DepContainer)
	for ref, depCtr := range depContainers {
		ctr, ok := ctrMap[depCtr.ID]
		if !ok {
			state = model.DepUnhealthy
			util.Logger.Warningf("deployment '%s' missing container '%s'", dID, depCtr.ID)
		}
		if state == "" {
			if ctr.Health != nil {
				switch *ctr.Health {
				case cew_model.TransitionState:
					state = model.DepTrans
				case cew_model.UnhealthyState:
					state = model.DepUnhealthy
				}
			} else {
				switch ctr.State {
				case cew_model.InitState, cew_model.RestartingState, cew_model.RemovingState:
					state = model.DepTrans
				case cew_model.StoppedState, cew_model.DeadState, cew_model.PausedState:
					state = model.DepUnhealthy
				}
			}
		}
		depCtr.Info = &model.ContainerInfo{
			ImageID: ctr.ImageID,
			State:   ctr.State,
		}
		withCtrInfo[ref] = depCtr
	}
	if state == "" {
		state = model.DepHealthy
	}
	return &state, withCtrInfo
}
