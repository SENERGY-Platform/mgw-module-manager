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

package dep_hdl

import (
	"context"
	"database/sql/driver"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
	"time"
)

func (h *Handler) getCurrentInst(ctx context.Context, dID string) (model.DepInstance, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	instances, err := h.storageHandler.ListInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepInstFilter{DepID: dID})
	if err != nil {
		return model.DepInstance{}, err
	}
	if len(instances) != 1 {
		return model.DepInstance{}, model.NewInternalError(fmt.Errorf("invalid number of instances: %d", len(instances)))
	}
	return h.storageHandler.ReadInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), instances[0].ID)
}

func (h *Handler) createInstance(ctx context.Context, tx driver.Tx, mod *module.Module, dID, depDirPth string, stringValues, hostRes, secrets, reqModDepMap, volumes map[string]string) (string, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	iID, err := h.storageHandler.CreateInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dID, time.Now().UTC())
	if err != nil {
		return "", err
	}
	order, err := sorting.GetSrvOrder(mod.Services)
	if err != nil {
		return "", model.NewInternalError(err)
	}
	for i := 0; i < len(order); i++ {
		ref := order[i]
		srv := mod.Services[ref]
		envVars, err := getEnvVars(srv, stringValues, reqModDepMap, dID, iID)
		if err != nil {
			return "", model.NewInternalError(err)
		}
		container := getContainer(srv, ref, getSrvName(iID, ref), dID, iID, envVars, getMounts(srv, volumes, hostRes, secrets, depDirPth), getPorts(srv.Ports))
		cID, err := h.cewClient.CreateContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), container)
		if err != nil {
			return "", model.NewInternalError(err)
		}
		err = h.storageHandler.CreateInstCtr(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, iID, cID, order[i], uint(i))
		if err != nil {
			return "", err
		}
	}
	return iID, nil
}

func (h *Handler) startInstance(ctx context.Context, iID string) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	containers, err := h.storageHandler.ListInstCtr(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), iID, model.CtrFilter{SortOrder: model.Ascending})
	if err != nil {
		return err
	}
	for _, ctr := range containers {
		err = h.cewClient.StartContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), ctr.ID)
		if err != nil {
			return model.NewInternalError(err)
		}
	}
	return nil
}

func (h *Handler) stopInstance(ctx context.Context, iID string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	containers, err := h.storageHandler.ListInstCtr(ctxWt, iID, model.CtrFilter{SortOrder: model.Descending})
	if err != nil {
		return err
	}
	for _, ctr := range containers {
		if err = h.stopContainer(ctx, ctr.ID); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) stopContainer(ctx context.Context, cID string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	jID, err := h.cewClient.StopContainer(ctxWt, cID)
	if err != nil {
		return model.NewInternalError(err)
	}
	job, err := h.cewJobHandler.AwaitJob(ctx, jID)
	if err != nil {
		return err
	}
	if job.Error != nil {
		return model.NewInternalError(fmt.Errorf("%v", job.Error))
	}
	return nil
}
