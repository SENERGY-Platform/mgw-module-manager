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
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
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
