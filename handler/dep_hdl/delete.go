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
	"errors"
	"fmt"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/naming_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
	"os"
	"path"
	"strings"
)

func (h *Handler) Delete(ctx context.Context, id string, force bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	dep, err := h.storageHandler.ReadDep(ctxWt, id, !force, true, true)
	if err != nil {
		return err
	}
	if !force {
		if dep.Enabled {
			return model.NewInvalidInputError(errors.New("deployment is enabled"))
		}
		if len(dep.DepRequiring) > 0 {
			ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
			defer cf2()
			deps, err := h.storageHandler.ListDep(ctxWt2, model.DepFilter{IDs: dep.DepRequiring}, false, false, false)
			if err != nil {
				return err
			}
			var reqBy []string
			for dID, d := range deps {
				reqBy = append(reqBy, fmt.Sprintf("%s (%s)", d.Name, dID))
			}
			return model.NewInternalError(fmt.Errorf("required by: %s", strings.Join(reqBy, ", ")))
		}
	}
	return h.delete(ctx, dep, force)
}

func (h *Handler) DeleteAll(ctx context.Context, filter model.DepFilter, force bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	deployments, err := h.storageHandler.ListDep(ctxWt, filter, true, true, true)
	if err != nil {
		return err
	}
	if !force {
		var enabled []string
		var reqByDepIDs []string
		for dID, dep := range deployments {
			if dep.Enabled {
				enabled = append(enabled, fmt.Sprintf("%s (%s)", dep.Name, dID))
			}
			if len(dep.DepRequiring) > 0 {
				for _, id := range dep.DepRequiring {
					if _, ok := deployments[id]; !ok {
						reqByDepIDs = append(reqByDepIDs, id)
					}
				}
			}
		}
		var errMsg string
		if len(enabled) > 0 {
			errMsg = "enabled deployments: " + strings.Join(enabled, ", ") + " "
		}
		if len(reqByDepIDs) > 0 {
			ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
			defer cf2()
			deployments, err = h.storageHandler.ListDep(ctxWt2, model.DepFilter{IDs: reqByDepIDs}, false, false, false)
			if err != nil {
				return err
			}
			var reqBy []string
			for dID, dep := range deployments {
				if dep.Enabled {
					reqBy = append(reqBy, fmt.Sprintf("%s (%s)", dep.Name, dID))
				}
			}
			errMsg += "required by: " + strings.Join(reqBy, ", ")
		}
		if len(enabled) > 0 || len(reqByDepIDs) > 0 {
			return model.NewInternalError(errors.New(errMsg))
		}
	}
	order, err := sorting.GetDepOrder(deployments)
	if err != nil {
		return err
	}
	for i := len(order) - 1; i >= 0; i-- {
		dep, ok := deployments[order[i]]
		if !ok {
			return model.NewInternalError(fmt.Errorf("deployment '%s' does not exist", order[i]))
		}
		if err = h.delete(ctx, dep, force); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) delete(ctx context.Context, dep model.Deployment, force bool) error {
	if err := h.removeContainers(ctx, dep.Containers, force); err != nil {
		return err
	}
	if dep.Enabled {
		if err := h.unloadSecrets(ctx, dep.ID); err != nil {
			return err
		}
	}
	ch := context_hdl.New()
	defer ch.CancelAll()
	volumes, err := h.cewClient.GetVolumes(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cew_model.VolumeFilter{Labels: map[string]string{naming_hdl.DeploymentIDLabel: dep.ID}})
	if err != nil {
		return err
	}
	var vols []string
	for _, v := range volumes {
		vols = append(vols, v.Name)
	}
	if err = h.removeVolumes(ctx, vols, force); err != nil {
		return model.NewInternalError(err)
	}
	if err = os.RemoveAll(path.Join(h.wrkSpcPath, dep.Dir)); err != nil {
		if !os.IsNotExist(err) {
			return model.NewInternalError(err)
		}
	}
	return h.storageHandler.DeleteDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), nil, dep.ID)
}

func (h *Handler) removeContainers(ctx context.Context, depContainers map[string]model.DepContainer, force bool) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, ctr := range depContainers {
		if ctr.ID != "" {
			err := h.cewClient.RemoveContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), ctr.ID, force)
			if err != nil {
				var nfe *cew_model.NotFoundError
				if !errors.As(err, &nfe) {
					return model.NewInternalError(err)
				}
			}
		}
	}
	return nil
}

func (h *Handler) removeVolumes(ctx context.Context, volumes []string, force bool) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, name := range volumes {
		err := h.cewClient.RemoveVolume(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), name, force)
		if err != nil {
			var nfe *cew_model.NotFoundError
			if !errors.As(err, &nfe) {
				return err
			}
		}
	}
	return nil
}
