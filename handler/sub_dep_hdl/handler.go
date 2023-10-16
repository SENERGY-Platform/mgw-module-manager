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

package sub_dep_hdl

import (
	"context"
	cew_lib "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"time"
)

type Handler struct {
	storageHandler handler.DepStorageHandler
	cewClient      cew_lib.Api
	dbTimeout      time.Duration
	httpTimeout    time.Duration
	managerID      string
	coreID         string
	moduleNet      string
}

func New(storageHandler handler.DepStorageHandler, cewClient cew_lib.Api, dbTimeout, httpTimeout time.Duration, managerID, moduleNet, coreID string) *Handler {
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

func (h *Handler) List(ctx context.Context, filter model.SubDepFilter, ctrInfo bool) ([]model.SubDeployment, error) {
	panic("not implemented")
}

func (h *Handler) Get(ctx context.Context, id string, ctrInfo bool) (model.SubDeployment, error) {
	panic("not implemented")
}

func (h *Handler) Create(ctx context.Context, sdReq model.SubDepBase) (string, error) {
	panic("not implemented")
}

func (h *Handler) Update(ctx context.Context, id string, sdReq model.SubDepBase) error {
	panic("not implemented")
}

func (h *Handler) Delete(ctx context.Context, id string) error {
	panic("not implemented")
}

func (h *Handler) Start(ctx context.Context, id string) error {
	panic("not implemented")
}

func (h *Handler) Stop(ctx context.Context, id string) error {
	panic("not implemented")
}
