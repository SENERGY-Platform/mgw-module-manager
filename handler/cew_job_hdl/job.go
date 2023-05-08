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

package cew_job_hdl

import (
	"context"
	"github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"time"
)

type Handler struct {
	cewClient   client.CewClient
	httpTimeout time.Duration
}

func New(cewClient client.CewClient, httpTimeout time.Duration) *Handler {
	return &Handler{
		cewClient:   cewClient,
		httpTimeout: httpTimeout,
	}
}

func (h *Handler) AwaitJob(ctx context.Context, jID string) (cew_model.Job, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			c, cf := context.WithTimeout(context.Background(), h.httpTimeout)
			err := h.cewClient.CancelJob(c, jID)
			if err != nil {
				util.Logger.Error(err)
			}
			cf()
			return cew_model.Job{}, ctx.Err()
		case <-ticker.C:
			j, err := h.cewClient.GetJob(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), jID)
			if err != nil {
				return cew_model.Job{}, err
			}
			if j.Completed != nil {
				return j, nil
			}
		}
	}
}
