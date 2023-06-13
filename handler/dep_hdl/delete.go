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
	dep, err := h.storageHandler.ReadDep(ctxWt, id)
	if err != nil {
		return err
	}
	if len(dep.DepRequiring) > 0 {
		return model.NewInternalError(fmt.Errorf("deplyoment is required by: %s", strings.Join(dep.DepRequiring, ", ")))
	}
	if err = h.delete(ctx, id, dep.Dir); err != nil {
		return err
	}
	if orphans && len(dep.RequiredDep) > 0 {
		reqDep := make(map[string]*model.Deployment)
		if err = h.getReqDep(ctx, dep, reqDep); err != nil {
			return err
		}
		order, err := sorting.GetDepOrder(reqDep)
		if err != nil {
			return model.NewInternalError(err)
		}
		for i := len(order) - 1; i >= 0; i-- {
			rd := reqDep[order[i]]
			if rd.Indirect && !isRequired(reqDep, rd.DepRequiring) {
				if err = h.delete(ctx, rd.ID, rd.Dir); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (h *Handler) delete(ctx context.Context, dID, inclDir string) error {
	if err := h.removeContainer(ctx, dID); err != nil {
		return err
	}
	ch := context_hdl.New()
	defer ch.CancelAll()
	volumes, err := h.cewClient.GetVolumes(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cew_model.VolumeFilter{Labels: map[string]string{"d_id": dID}})
	if err != nil {
		return err
	}
	var vols []string
	for _, v := range volumes {
		vols = append(vols, v.Name)
	}
	if err = h.removeVolumes(ctx, vols); err != nil {
		return model.NewInternalError(err)
	}
	if err = os.RemoveAll(path.Join(h.wrkSpcPath, inclDir)); err != nil {
		return model.NewInternalError(err)
	}
	if err = h.storageHandler.DeleteDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID); err != nil {
		return err
	}
	return nil
}

func (h *Handler) removeContainer(ctx context.Context, dID string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	instances, err := h.storageHandler.ListInst(ctxWt, model.DepInstFilter{DepID: dID})
	if err != nil {
		return err
	}
	for _, instance := range instances {
		err = h.removeInstance(ctx, instance.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func isRequired(reqDep map[string]*model.Deployment, depRequiring []string) bool {
	for _, dID := range depRequiring {
		if dep, ok := reqDep[dID]; !ok {
			return true
		} else {
			if !dep.Indirect {
				return true
			}
		}
	}
	return false
}
