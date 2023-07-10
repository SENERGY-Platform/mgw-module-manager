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

package dep_health_hdl

import (
	"context"
	cew_client "github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"time"
)

type Handler struct {
	cewClient   cew_client.CewClient
	httpTimeout time.Duration
}

func New(cewClient cew_client.CewClient, httpTimeout time.Duration) *Handler {
	return &Handler{
		cewClient:   cewClient,
		httpTimeout: httpTimeout,
	}
}

func (h *Handler) checkContainers(ctx context.Context, containers []model.Container) (model.HealthStatus, []model.CtrHealthInfo, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	var status model.HealthStatus
	var ctrHealthInfo []model.CtrHealthInfo
	for _, container := range containers {
		ctr, err := h.cewClient.GetContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), container.ID)
		if err != nil {
			return "", nil, model.NewInternalError(err)
		}
		if status == "" {
			if ctr.Health != nil {
				switch *ctr.Health {
				case cew_model.TransitionState:
					status = model.DepTrans
				case cew_model.UnhealthyState:
					status = model.DepUnhealthy
				}
			} else {
				switch ctr.State {
				case cew_model.InitState, cew_model.RestartingState, cew_model.RemovingState:
					status = model.DepTrans
				case cew_model.StoppedState, cew_model.DeadState, cew_model.PausedState:
					status = model.DepUnhealthy
				}
			}
		}
		ctrHealthInfo = append(ctrHealthInfo, model.CtrHealthInfo{
			ID:    container.ID,
			Ref:   container.Ref,
			State: ctr.State,
		})
	}
	if status == "" {
		status = model.DepHealthy
	}
	return status, ctrHealthInfo, nil
}

func (h *Handler) List(ctx context.Context, instances map[string]model.DepInstance) (map[string]model.DepHealthInfo, error) {
	healthInfo := make(map[string]model.DepHealthInfo)
	for dID, instance := range instances {
		status, ctrHealthInfo, err := h.checkContainers(ctx, instance.Containers)
		if err != nil {
			return nil, err
		}
		healthInfo[dID] = model.DepHealthInfo{
			Status:     status,
			Containers: ctrHealthInfo,
		}
	}
	return healthInfo, nil
}

func (h *Handler) Get(ctx context.Context, instance model.DepInstance) (model.DepHealthInfo, error) {
	status, ctrHealthInfo, err := h.checkContainers(ctx, instance.Containers)
	if err != nil {
		return model.DepHealthInfo{}, err
	}
	return model.DepHealthInfo{
		Status:     status,
		Containers: ctrHealthInfo,
	}, nil
}
