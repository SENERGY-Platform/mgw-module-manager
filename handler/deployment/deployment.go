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
	"github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"time"
)

type Handler struct {
	storageHandler handler.DepStorageHandler
	cfgVltHandler  handler.CfgValidationHandler
	cewClient      client.CewClient
	stgHdlTimeout  time.Duration
}

func NewHandler(storageHandler handler.DepStorageHandler, cfgVltHandler handler.CfgValidationHandler, cewClient client.CewClient, storageHandlerTimeout time.Duration) *Handler {
	return &Handler{
		storageHandler: storageHandler,
		cfgVltHandler:  cfgVltHandler,
		cewClient:      cewClient,
		stgHdlTimeout:  storageHandlerTimeout,
	}
}

func (h *Handler) List(ctx context.Context) ([]model.DepMeta, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.stgHdlTimeout)
	defer cf()
	return h.storageHandler.ListDep(ctxWt)
}

func (h *Handler) Get(ctx context.Context, id string) (*model.Deployment, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.stgHdlTimeout)
	defer cf()
	return h.storageHandler.ReadDep(ctxWt, id)
}

func (h *Handler) Create(ctx context.Context, m *module.Module, name *string, hostRes map[string]string, secrets map[string]string, configs map[string]any) (string, error) {
	d, err := genDeployment(m, name, hostRes, secrets, configs)
	if err != nil {
		return "", model.NewInvalidInputError(err)
	}
	if err = h.validateConfigs(d.Configs, m.Configs); err != nil {
		return "", err
	}
	d.Created = time.Now().UTC()
	ctxWt, cf := context.WithTimeout(ctx, h.stgHdlTimeout)
	defer cf()
	tx, id, err := h.storageHandler.CreateDep(ctxWt, d)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	err = tx.Commit()
	if err != nil {
		return "", model.NewInternalError(err)
	}
	return id, nil
}

func (h *Handler) Delete(ctx context.Context, id string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.stgHdlTimeout)
	defer cf()
	return h.storageHandler.DeleteDep(ctxWt, id)
}

func (h *Handler) Update(ctx context.Context, m *module.Module, id string, name *string, hostRes map[string]string, secrets map[string]string, configs map[string]any) error {
	d, err := genDeployment(m, name, hostRes, secrets, configs)
	if err != nil {
		return model.NewInvalidInputError(err)
	}
	if err = h.validateConfigs(d.Configs, m.Configs); err != nil {
		return err
	}
	d.ID = id
	d.Updated = time.Now().UTC()
	ctxWt, cf := context.WithTimeout(ctx, h.stgHdlTimeout)
	defer cf()
	tx, err := h.storageHandler.UpdateDep(ctxWt, d)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = tx.Commit()
	if err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) Deploy(ctx context.Context, m *module.Module, mPath string, d *model.Deployment) error {

	return nil
}

func (h *Handler) Start(ctx context.Context, id string) error {
	panic("not implemented")
}

func (h *Handler) Stop(ctx context.Context, id string) error {
	panic("not implemented")
}

func (h *Handler) validateConfigs(dCs map[string]model.DepConfig, mCs module.Configs) error {
	for ref, dC := range dCs {
		mC := mCs[ref]
		if err := h.cfgVltHandler.ValidateValue(mC.Type, mC.TypeOpt, dC.Value, mC.IsSlice, mC.DataType); err != nil {
			return model.NewInvalidInputError(err)
		}
		if mC.Options != nil && !mC.OptExt {
			if err := h.cfgVltHandler.ValidateValInOpt(mC.Options, dC.Value, mC.IsSlice, mC.DataType); err != nil {
				return model.NewInvalidInputError(err)
			}
		}
	}
	return nil
}
