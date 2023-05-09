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
	"fmt"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
	"os"
	"path"
	"strings"
)

func (h *Handler) Delete(ctx context.Context, id string, orphans bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	d, err := h.storageHandler.ReadDep(ctxWt, id)
	if err != nil {
		return err
	}
	if len(d.DepRequiring) > 0 {
		return model.NewInternalError(fmt.Errorf("deplyoment is required by: %s", strings.Join(d.DepRequiring, ", ")))
	}
	if err = h.delete(ctx, id); err != nil {
		return err
	}
	if orphans && len(d.RequiredDep) > 0 {
		reqDep := make(map[string]*model.Deployment)
		if err = h.getReqDep(ctx, d, reqDep); err != nil {
			return err
		}
		order, err := sorting.GetDepOrder(reqDep)
		if err != nil {
			return model.NewInternalError(err)
		}
		for i := len(order) - 1; i >= 0; i-- {
			rd := reqDep[order[i]]
			if rd.Indirect && len(rd.DepRequiring) == 0 {
				if err = h.delete(ctx, rd.ID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (h *Handler) delete(ctx context.Context, dID string) error {
	if err := h.removeContainer(ctx, dID); err != nil {
		return err
	}
	if err := h.removeVolumes(ctx, dID); err != nil {
		return model.NewInternalError(err)
	}
	if err := h.rmDepDir(ctx, dID); err != nil {
		return err
	}
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	if err := h.storageHandler.DeleteDep(ctxWt, dID); err != nil {
		return err
	}
	return nil
}

func (h *Handler) rmDepDir(_ context.Context, dID string) error {
	if err := os.RemoveAll(path.Join(h.wrkSpcPath, dID)); err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) removeContainer(ctx context.Context, dID string) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	instances, err := h.storageHandler.ListInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepInstFilter{DepID: dID})
	if err != nil {
		return err
	}
	for _, instance := range instances {
		containers, err := h.storageHandler.ListInstCtr(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), instance.ID, model.CtrFilter{})
		if err != nil {
			return err
		}
		for _, ctr := range containers {
			err := h.cewClient.RemoveContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), ctr.ID)
			if err != nil {
				return model.NewInternalError(err)
			}
		}
	}
	return nil
}

func (h *Handler) removeVolumes(ctx context.Context, dID string) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	volumes, err := h.cewClient.GetVolumes(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cew_model.VolumeFilter{Labels: map[string]string{"d_id": dID}})
	if err != nil {
		return err
	}
	for _, volume := range volumes {
		err := h.cewClient.RemoveVolume(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), volume.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) getOrphans(ctx context.Context) ([]model.DepMeta, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	dms, err := h.storageHandler.ListDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepFilter{Indirect: true})
	if err != nil {
		return nil, err
	}
	var orphans []model.DepMeta
	for _, dm := range dms {
		d, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dm.ID)
		if err != nil {
			return nil, err
		}
		if len(d.DepRequiring) == 0 {
			orphans = append(orphans, dm)
		}
	}
	return orphans, nil
}
