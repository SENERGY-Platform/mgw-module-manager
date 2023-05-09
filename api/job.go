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

package api

import (
	"context"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

func (a *Api) GetJobs(_ context.Context, filter model.JobFilter) ([]model.Job, error) {
	return a.jobHandler.List(filter), nil
}

func (a *Api) GetJob(_ context.Context, id string) (model.Job, error) {
	return a.jobHandler.Get(id)
}

func (a *Api) CancelJob(_ context.Context, id string) error {
	return a.jobHandler.Cancel(id)
}
