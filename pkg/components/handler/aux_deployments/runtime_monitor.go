/*
 * Copyright 2026 InfAI (CC SES)
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

package aux_deployments

import (
	"context"
	"maps"
	"time"

	lib_external "github.com/SENERGY-Platform/mgw-module-manager/lib/models/external"
	helper_containers "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/containers"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) RuntimeMonitor(ctx context.Context) {
	timer := time.NewTimer(h.config.RuntimeMonitorStartupDelay)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			h.checkDeployments(ctx)
			timer.Reset(h.config.RuntimeMonitorLoopDelay)
		case <-ctx.Done():
			return
		}
	}
}

func (h *Handler) checkDeployments(ctx context.Context) {
	auxDepsByParent, cewContainersMap, err := h.getCurrentRuntimeData(ctx)
	if err != nil {
		rmLogger.ErrorContext(ctx, "get auxiliary deployments", slog_keys.Error, err)
		return
	}
	filteredAuxDepsByParent := h.runtimeMonitorJobsFilter(auxDepsByParent)
	for parentId, parent := range filteredAuxDepsByParent {
		if parent.Enabled {
			var toStart []string
			var toStop []string
			for _, auxDep := range parent.AuxiliaryDeployments {
				container, ok := cewContainersMap[auxDep.Container.Name]
				if !ok || container.State == lib_external.CewRemovingState {
					continue
				}
				if auxDep.Enabled {
					if getContainerState(container.State) < 0 {
						toStart = append(toStart, container.Name)
					}
				} else {
					if getContainerState(container.State) > 0 {
						toStop = append(toStop, container.Name)
					}
				}
			}
			if len(toStart) > 0 || len(toStop) > 0 {
				h.runtimeMonitorJobsAdd(parentId)
				go func(pId string, tSrt, tStp []string) {
					defer h.runtimeMonitorJobsRemove(pId)
					h.startContainers(ctx, tSrt)
					h.stopContainers(ctx, tStp)
				}(parentId, toStart, toStop)
			}
		} else {
			var toStop []string
			for _, auxDep := range parent.AuxiliaryDeployments {
				container, ok := cewContainersMap[auxDep.Container.Name]
				if !ok || container.State == lib_external.CewRemovingState {
					continue
				}
				if getContainerState(container.State) > 0 {
					toStop = append(toStop, container.Name)
				}
			}
			if len(toStop) > 0 {
				h.runtimeMonitorJobsAdd(parentId)
				go func(pId string, ts []string) {
					defer h.runtimeMonitorJobsRemove(pId)
					h.stopContainers(ctx, ts)
				}(parentId, toStop)
			}
		}
	}
}

func (h *Handler) getCurrentRuntimeData(ctx context.Context) (
	map[string]pkg_models.AuxiliaryDeploymentParent,
	map[string]external_models.CewContainer,
	error,
) {
	auxDepsByParent, err := h.databaseHandler.ReadAuxDeploymentsByParent(ctx)
	if err != nil {
		return nil, nil, err
	}
	tmp := make(map[string]pkg_models.AuxiliaryDeployment)
	for _, auxDeps := range auxDepsByParent {
		maps.Copy(tmp, auxDeps.AuxiliaryDeployments)
	}
	cewContainersMap, err := h.getCewContainers(ctx, tmp)
	if err != nil {
		return nil, nil, err
	}
	return auxDepsByParent, cewContainersMap, nil
}

func (h *Handler) startContainers(
	ctx context.Context,
	containerNames []string,
) {
	rmLogger.DebugContext(ctx, "start containers", slog_keys.Containers, containerNames)
	for _, name := range containerNames {
		err := h.containerEngineWrapperClient.StartContainer(ctx, name)
		if err != nil {
			rmLogger.ErrorContext(ctx, "start containers", slog_keys.Containers, containerNames, slog_keys.Error, err)
		}
	}
}

func (h *Handler) stopContainers(
	ctx context.Context,
	containerNames []string,
) {
	rmLogger.DebugContext(ctx, "stop containers", slog_keys.Containers, containerNames)
	for _, name := range containerNames {
		err := helper_containers.Stop(ctx, h.containerEngineWrapperClient, name, h.config.JobPollInterval)
		if err != nil {
			rmLogger.ErrorContext(ctx, "stop containers", slog_keys.Containers, containerNames, slog_keys.Error, err)
		}
	}
}

func (h *Handler) runtimeMonitorJobsFilter(
	auxDepsByParent map[string]pkg_models.AuxiliaryDeploymentParent,
) map[string]pkg_models.AuxiliaryDeploymentParent {
	h.runtimeMonitorJobsMu.RLock()
	defer h.runtimeMonitorJobsMu.RUnlock()
	filteredDeployments := make(map[string]pkg_models.AuxiliaryDeploymentParent)
	for deploymentId, auxDeps := range auxDepsByParent {
		_, ok := h.runtimeMonitorJobs[deploymentId]
		if !ok {
			filteredDeployments[deploymentId] = auxDeps
		}
	}
	return filteredDeployments
}

func (h *Handler) runtimeMonitorJobsAdd(id string) {
	h.runtimeMonitorJobsMu.Lock()
	defer h.runtimeMonitorJobsMu.Unlock()
	h.runtimeMonitorJobs[id] = struct{}{}
}

func (h *Handler) runtimeMonitorJobsRemove(id string) {
	h.runtimeMonitorJobsMu.Lock()
	defer h.runtimeMonitorJobsMu.Unlock()
	delete(h.runtimeMonitorJobs, id)
}

func getContainerState(state string) int {
	switch state {
	case lib_external.CewInitState:
		return -1
	case lib_external.CewStoppedState:
		return -1
	case lib_external.CewDeadState:
		return -1
	case lib_external.CewRunningState:
		return 1
	case lib_external.CewPausedState:
		return 1
	case lib_external.CewRestartingState:
		return 1
	}
	return 0
}
