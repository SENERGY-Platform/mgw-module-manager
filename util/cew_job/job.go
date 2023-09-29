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

package cew_job

import (
	"context"
	cew_lib "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"time"
)

func Await(ctx context.Context, cewClient cew_lib.Api, jID string, httpTimeout time.Duration) (cew_model.Job, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			c, cf := context.WithTimeout(context.Background(), httpTimeout)
			err := cewClient.CancelJob(c, jID)
			if err != nil {
				util.Logger.Error(err)
			}
			cf()
			return cew_model.Job{}, model.NewInternalError(ctx.Err())
		case <-ticker.C:
			j, err := cewClient.GetJob(ch.Add(context.WithTimeout(ctx, httpTimeout)), jID)
			if err != nil {
				return cew_model.Job{}, model.NewInternalError(err)
			}
			if j.Completed != nil {
				return j, nil
			}
		}
	}
}
