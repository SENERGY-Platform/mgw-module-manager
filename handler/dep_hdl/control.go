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
	"github.com/SENERGY-Platform/mgw-go-service-base/context-hdl"
	job_hdl_lib "github.com/SENERGY-Platform/mgw-go-service-base/job-hdl/lib"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
	"net/http"
	"sort"
	"strings"
	"time"
)

func (h *Handler) Start(ctx context.Context, id string, dependencies bool) ([]string, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	if dependencies {
		depTree, err := h.storageHandler.ReadDepTree(ctxWt, id, true, true)
		if err != nil {
			return nil, err
		}
		return h.startTree(ctx, depTree)
	} else {
		dep, err := h.storageHandler.ReadDep(ctxWt, id, false, true, true)
		if err != nil {
			return nil, err
		}
		return []string{id}, h.start(ctx, dep)
	}
}

func (h *Handler) StartAll(ctx context.Context, filter lib_model.DepFilter, dependencies bool) ([]string, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	deployments, err := h.storageHandler.ListDep(ctxWt, filter, true, true, true)
	if err != nil {
		return nil, err
	}
	if dependencies {
		if err = h.storageHandler.AppendDepTree(ctxWt, deployments, true, true); err != nil {
			return nil, err
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
		deployments, err := h.storageHandler.ListDep(ctxWt2, lib_model.DepFilter{IDs: dep.DepRequiring}, false, false, false)
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
			return lib_model.NewInternalError(fmt.Errorf("required by %s", strings.Join(reqBy, ", ")))
		}
	}
	return h.stop(ctx, dep)
}

func (h *Handler) StopAll(ctx context.Context, filter lib_model.DepFilter, force bool) ([]string, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	deployments, err := h.storageHandler.ListDep(ctxWt, filter, true, true, true)
	if err != nil {
		return nil, err
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
			deps, err := h.storageHandler.ListDep(ctxWt2, lib_model.DepFilter{IDs: reqByDepIDs}, false, false, false)
			if err != nil {
				return nil, err
			}
			var reqBy []string
			for dID, dep := range deps {
				if dep.Enabled {
					reqBy = append(reqBy, fmt.Sprintf("%s (%s)", dep.Name, dID))
				}
			}
			if len(reqBy) > 0 {
				return nil, lib_model.NewInternalError(fmt.Errorf("required by %s", strings.Join(reqBy, ", ")))
			}
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

func (h *Handler) RestartAll(ctx context.Context, filter lib_model.DepFilter) ([]string, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	deployments, err := h.storageHandler.ListDep(ctxWt, filter, false, true, true)
	if err != nil {
		return nil, err
	}
	var restarted []string
	for _, dep := range deployments {
		if err = h.restart(ctx, dep); err != nil {
			return restarted, err
		}
		restarted = append(restarted, dep.ID)
	}
	return restarted, err
}

func (h *Handler) start(ctx context.Context, dep lib_model.Deployment) error {
	if err := h.loadSecrets(ctx, dep); err != nil {
		return err
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

func (h *Handler) startTree(ctx context.Context, depTree map[string]lib_model.Deployment) ([]string, error) {
	order, err := sorting.GetDepOrder(depTree)
	if err != nil {
		return nil, lib_model.NewInternalError(err)
	}
	var started []string
	for _, dID := range order {
		dep, ok := depTree[dID]
		if !ok {
			return started, lib_model.NewInternalError(fmt.Errorf("deployment '%s' does not exist", dID))
		}
		if err = h.start(ctx, dep); err != nil {
			return started, err
		}
		started = append(started, dID)
	}
	return started, nil
}

func (h *Handler) stop(ctx context.Context, dep lib_model.Deployment) error {
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

func (h *Handler) stopTree(ctx context.Context, depTree map[string]lib_model.Deployment) ([]string, error) {
	order, err := sorting.GetDepOrder(depTree)
	if err != nil {
		return nil, err
	}
	var stopped []string
	for i := len(order) - 1; i >= 0; i-- {
		dep, ok := depTree[order[i]]
		if !ok {
			return stopped, lib_model.NewInternalError(fmt.Errorf("deployment '%s' does not exist", order[i]))
		}
		if err = h.stop(ctx, dep); err != nil {
			return stopped, err
		}
		stopped = append(stopped, dep.ID)
	}
	return stopped, nil
}

func (h *Handler) restart(ctx context.Context, dep lib_model.Deployment) error {
	if err := h.stopContainers(ctx, dep.Containers); err != nil {
		return err
	}
	return h.start(ctx, dep)
}

func (h *Handler) startContainers(ctx context.Context, depContainers map[string]lib_model.DepContainer) error {
	order := getDepContainerOrder(depContainers, 1)
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, ref := range order {
		if err := h.cewClient.StartContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), depContainers[ref].ID); err != nil {
			return lib_model.NewInternalError(err)
		}
	}
	return nil
}

func (h *Handler) stopContainers(ctx context.Context, depContainers map[string]lib_model.DepContainer) error {
	order := getDepContainerOrder(depContainers, -1)
	for _, ref := range order {
		if err := h.stopContainer(ctx, depContainers[ref].ID); err != nil {
			var nfe *lib_model.NotFoundError
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
		return lib_model.NewInternalError(err)
	}
	job, err := job_hdl_lib.Await(ctx, h.cewClient, jID, time.Second, h.httpTimeout, util.Logger)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	if job.Error != nil {
		if job.Error.Code != nil && *job.Error.Code == http.StatusNotFound {
			return lib_model.NewNotFoundError(errors.New(job.Error.Message))
		}
		return lib_model.NewInternalError(errors.New(job.Error.Message))
	}
	return nil
}

func getDepContainerOrder(depContainers map[string]lib_model.DepContainer, order int8) []string {
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
