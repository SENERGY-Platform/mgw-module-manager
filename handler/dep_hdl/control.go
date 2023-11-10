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
	job_hdl_lib "github.com/SENERGY-Platform/go-service-base/job-hdl/lib"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
	"net/http"
	"sort"
	"strings"
	"time"
)

func (h *Handler) Start(ctx context.Context, id string, dependencies bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	if dependencies {
		depTree, err := h.storageHandler.ReadDepTree(ctxWt, id, true, true)
		if err != nil {
			return err
		}
		return h.startTree(ctx, depTree)
	} else {
		dep, err := h.storageHandler.ReadDep(ctxWt, id, false, true, true)
		if err != nil {
			return err
		}
		return h.start(ctx, dep)
	}
}

func (h *Handler) StartAll(ctx context.Context, filter model.DepFilter, dependencies bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	deployments, err := h.storageHandler.ListDep(ctxWt, filter, true, true, true)
	if err != nil {
		return err
	}
	if dependencies {
		if err = h.storageHandler.AppendDepTree(ctxWt, deployments, true, true); err != nil {
			return err
		}
	}
	return h.startTree(ctx, deployments)
}

func (h *Handler) Stop(ctx context.Context, id string, force bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	dep, err := h.storageHandler.ReadDep(ctxWt, id, !force, true, true)
	if err != nil {
		return err
	}
	if dep.Enabled && !force && len(dep.DepRequiring) > 0 {
		ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
		defer cf2()
		deployments, err := h.storageHandler.ListDep(ctxWt2, model.DepFilter{IDs: dep.DepRequiring}, false, false, false)
		if err != nil {
			return err
		}
		var reqBy []string
		for dID, d := range deployments {
			if d.Enabled {
				reqBy = append(reqBy, fmt.Sprintf("%s (%s)", d.Name, dID))
			}
		}
		if len(reqBy) > 0 {
			return model.NewInternalError(fmt.Errorf("required by: %s", strings.Join(reqBy, ", ")))
		}
	}
	return h.stop(ctx, dep)
}

func (h *Handler) StopAll(ctx context.Context, filter model.DepFilter, force bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	deployments, err := h.storageHandler.ListDep(ctxWt, filter, true, true, true)
	if err != nil {
		return err
	}
	if !force {
		var reqByDepIDs []string
		for _, dep := range deployments {
			if dep.Enabled && len(dep.DepRequiring) > 0 {
				for _, dID := range dep.DepRequiring {
					if _, ok := deployments[dID]; !ok {
						reqByDepIDs = append(reqByDepIDs, dID)
					}
				}
			}
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
			return model.NewInternalError(fmt.Errorf("required by: %s", strings.Join(reqBy, ", ")))
		}
	}
	return h.stopTree(ctx, deployments)
}

func (h *Handler) Restart(ctx context.Context, id string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	dep, err := h.storageHandler.ReadDep(ctxWt, id, false, true, true)
	if err != nil {
		return err
	}
	return h.restart(ctx, dep)
}

func (h *Handler) RestartAll(ctx context.Context, filter model.DepFilter) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	deployments, err := h.storageHandler.ListDep(ctxWt, filter, false, true, true)
	if err != nil {
		return err
	}
	for _, dep := range deployments {
		if err = h.restart(ctx, dep); err != nil {
			return err
		}
	}
	return err
}

func (h *Handler) start(ctx context.Context, dep model.Deployment) error {
	if !dep.Enabled {
		if err := h.loadSecrets(ctx, dep); err != nil {
			return err
		}
	}
	if err := h.startContainers(ctx, dep.Containers); err != nil {
		return err
	}
	if !dep.Enabled {
		dep.Enabled = true
		ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
		defer cf()
		return h.storageHandler.UpdateDep(ctxWt, nil, dep.DepBase)
	}
	return nil
}

func (h *Handler) startTree(ctx context.Context, depTree map[string]model.Deployment) error {
	order, err := sorting.GetDepOrder(depTree)
	if err != nil {
		return model.NewInternalError(err)
	}
	for _, dID := range order {
		dep, ok := depTree[dID]
		if !ok {
			return model.NewInternalError(fmt.Errorf("deployment '%s' does not exist", dID))
		}
		if err = h.start(ctx, dep); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) stop(ctx context.Context, dep model.Deployment) error {
	if err := h.stopContainers(ctx, dep.Containers); err != nil {
		return err
	}
	if dep.Enabled {
		if err := h.unloadSecrets(ctx, dep.ID); err != nil {
			return err
		}
		dep.Enabled = false
		ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
		defer cf()
		if err := h.storageHandler.UpdateDep(ctxWt, nil, dep.DepBase); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) stopTree(ctx context.Context, depTree map[string]model.Deployment) error {
	order, err := sorting.GetDepOrder(depTree)
	if err != nil {
		return err
	}
	for i := len(order) - 1; i >= 0; i-- {
		dep, ok := depTree[order[i]]
		if !ok {
			return model.NewInternalError(fmt.Errorf("deployment '%s' does not exist", order[i]))
		}
		if err = h.stop(ctx, dep); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) restart(ctx context.Context, dep model.Deployment) error {
	if err := h.stopContainers(ctx, dep.Containers); err != nil {
		return err
	}
	return h.start(ctx, dep)
}

func (h *Handler) startContainers(ctx context.Context, depContainers map[string]model.DepContainer) error {
	order := getDepContainerOrder(depContainers, 1)
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, ref := range order {
		if err := h.cewClient.StartContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), depContainers[ref].ID); err != nil {
			return model.NewInternalError(err)
		}
	}
	return nil
}

func (h *Handler) stopContainers(ctx context.Context, depContainers map[string]model.DepContainer) error {
	order := getDepContainerOrder(depContainers, -1)
	for _, ref := range order {
		if err := h.stopContainer(ctx, depContainers[ref].ID); err != nil {
			var nfe *model.NotFoundError
			if !errors.As(err, &nfe) {
				return err
			}
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
	job, err := job_hdl_lib.Await(ctx, h.cewClient, jID, time.Second, h.httpTimeout, util.Logger)
	if err != nil {
		return model.NewInternalError(err)
	}
	if job.Error != nil {
		if job.Error.Code != nil && *job.Error.Code == http.StatusNotFound {
			return model.NewNotFoundError(errors.New(job.Error.Message))
		}
		return model.NewInternalError(errors.New(job.Error.Message))
	}
	return nil
}

func getDepContainerOrder(depContainers map[string]model.DepContainer, order int8) []string {
	keys := make([]string, 0, len(depContainers))
	for key := range depContainers {
		keys = append(keys, key)
	}
	if order > 0 {
		sort.SliceStable(keys, func(i, j int) bool {
			return depContainers[keys[i]].Order < depContainers[keys[j]].Order
		})
	}
	if order < 0 {
		sort.SliceStable(keys, func(i, j int) bool {
			return depContainers[keys[i]].Order > depContainers[keys[j]].Order
		})
	}
	return keys
}
