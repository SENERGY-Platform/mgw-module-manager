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
	"database/sql/driver"
	"fmt"
	"github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/parser"
	"io/fs"
	"os"
	"path"
	"time"
)

type Handler struct {
	storageHandler handler.DepStorageHandler
	cfgVltHandler  handler.CfgValidationHandler
	cewJobHandler  handler.CewJobHandler
	cewClient      client.CewClient
	dbTimeout      time.Duration
	httpTimeout    time.Duration
	wrkSpcPath     string
	perm           fs.FileMode
}

func New(storageHandler handler.DepStorageHandler, cfgVltHandler handler.CfgValidationHandler, cewJobHandler handler.CewJobHandler, cewClient client.CewClient, dbTimeout time.Duration, httpTimeout time.Duration, workspacePath string, perm fs.FileMode) (*Handler, error) {
	if !path.IsAbs(workspacePath) {
		return nil, fmt.Errorf("workspace path must be absolute")
	}
	return &Handler{
		storageHandler: storageHandler,
		cfgVltHandler:  cfgVltHandler,
		cewJobHandler:  cewJobHandler,
		cewClient:      cewClient,
		dbTimeout:      dbTimeout,
		httpTimeout:    httpTimeout,
		wrkSpcPath:     workspacePath,
		perm:           perm,
	}, nil
}

func (h *Handler) InitWorkspace() error {
	if err := os.MkdirAll(h.wrkSpcPath, h.perm); err != nil {
		return err
	}
	return nil
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

func (h *Handler) prepareDep(mod *module.Module, depReq model.DepRequestBase) (name string, userConfigs map[string]any, hostRes, secrets map[string]string, err error) {
	name = getName(mod.Name, depReq.Name)
	userConfigs, err = h.getUserConfigs(mod.Configs, depReq.Configs)
	if err != nil {
		return "", nil, nil, nil, model.NewInvalidInputError(err)
	}
	hostRes, err = h.getHostRes(mod.HostResources, depReq.HostResources)
	if err != nil {
		return "", nil, nil, nil, err
	}
	secrets, err = h.getSecrets(mod.Secrets, depReq.Secrets)
	if err != nil {
		return "", nil, nil, nil, err
	}
	return
}

func (h *Handler) storeDep(ctx context.Context, tx driver.Tx, dID string, hostRes, secrets map[string]string, modConfigs module.Configs, userConfigs map[string]any) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	if len(hostRes) > 0 {
		if err := h.storageHandler.CreateDepHostRes(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, hostRes, dID); err != nil {
			return err
		}
	}
	if len(secrets) > 0 {
		if err := h.storageHandler.CreateDepSecrets(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, secrets, dID); err != nil {
			return err
		}
	}
	if len(userConfigs) > 0 {
		if err := h.storageHandler.CreateDepConfigs(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, modConfigs, userConfigs, dID); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) wipeDep(ctx context.Context, tx driver.Tx, dID string) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	if err := h.storageHandler.DeleteDepHostRes(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dID); err != nil {
		return err
	}
	if err := h.storageHandler.DeleteDepSecrets(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dID); err != nil {
		return err
	}
	if err := h.storageHandler.DeleteDepConfigs(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dID); err != nil {
		return err
	}
	return nil
}

func (h *Handler) getReqModDepMap(ctx context.Context, reqMod map[string]string) (map[string]string, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	depMap := make(map[string]string)
	for mID := range reqMod {
		depList, err := h.storageHandler.ListDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepFilter{ModuleID: mID})
		if err != nil {
			return nil, err
		}
		if len(depList) == 0 {
			return nil, model.NewInternalError(fmt.Errorf("dependency '%s' not deployed", mID))
		}
		depMap[mID] = depList[0].ID
	}
	return depMap, nil
}
