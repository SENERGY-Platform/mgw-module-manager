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
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"time"
)

type Handler struct {
	storageHandler handler.DepStorageHandler
	cfgVltHandler  handler.CfgValidationHandler
	cewClient      client.CewClient
	dbTimeout      time.Duration
	httpTimeout    time.Duration
}

func NewHandler(storageHandler handler.DepStorageHandler, cfgVltHandler handler.CfgValidationHandler, cewClient client.CewClient, dbTimeout time.Duration, httpTimeout time.Duration) *Handler {
	return &Handler{
		storageHandler: storageHandler,
		cfgVltHandler:  cfgVltHandler,
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

func (h *Handler) Create(ctx context.Context, m *module.Module, name *string, hostRes map[string]string, secrets map[string]string, configs map[string]any) (string, error) {
	dbCtx, dbCf := context.WithTimeout(ctx, h.dbTimeout)
	defer dbCf()
	deployment, rad, sad, err := genDeployment(m, name, hostRes, secrets, configs)
	if err != nil {
		return "", model.NewInvalidInputError(err)
	}
	if len(rad) > 0 || len(sad) > 0 {
		return "", model.NewInternalError(errors.New("auto resource discovery not implemented"))
	}
	if err = h.validateConfigs(deployment.Configs, m.Configs); err != nil {
		return "", err
	}
	dName := m.Name
	if name != nil {
		dName = *name
	}
	timestamp := time.Now().UTC()
	tx, err := h.storageHandler.BeginTransaction(dbCtx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	dID, err := h.storageHandler.CreateDep(dbCtx, tx, m.ID, dName, dRs, dSs, dCs, timestamp)
	if err != nil {
		return "", err
	}
	iID, err := h.storageHandler.CreateInst(dbCtx, tx, dID, timestamp)
	if err != nil {
		return "", err
	}

	for v := range m.Volumes {
		vName, err := h.createVolume(ctx, dID, iID, v)
		if err != nil {
			return "", err
		}
	}

	err = tx.Commit()
	if err != nil {
		return "", model.NewInternalError(err)
	}
	return dID, nil
}

func (h *Handler) Delete(ctx context.Context, id string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	return h.storageHandler.DeleteDep(ctxWt, id)
}

func (h *Handler) Update(ctx context.Context, m *module.Module, id string, name *string, hostRes map[string]string, secrets map[string]string, configs map[string]any) error {
	panic("not implemented")
	//d, rad, sad, err := genDeployment(m, name, hostRes, secrets, configs)
	//if err != nil {
	//	return model.NewInvalidInputError(err)
	//}
	//if len(rad) > 0 || len(sad) > 0 {
	//	return model.NewInternalError(errors.New("auto resource discovery not implemented"))
	//}
	//if err = h.validateConfigs(d.Configs, m.Configs); err != nil {
	//	return err
	//}
	//d.ID = id
	//d.Updated = time.Now().UTC()
	//ctxWt, cf := context.WithTimeout(ctx, h.stgHdlTimeout)
	//defer cf()
	//tx, err := h.storageHandler.UpdateDep(ctxWt, d)
	//if err != nil {
	//	return err
	//}
	//defer tx.Rollback()
	//err = tx.Commit()
	//if err != nil {
	//	return model.NewInternalError(err)
	//}
	//return nil
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

func (h *Handler) createVolume(ctx context.Context, dID, iID, v string) (string, error) {
	httpCtx, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	vName, err := h.cewClient.CreateVolume(httpCtx, cew_model.Volume{
		Name:   genVolumeName(iID, v),
		Labels: map[string]string{"d_id": dID, "i_id": iID},
	})
	if err != nil {
		return "", model.NewInternalError(err)
	}
	return vName, nil
}
