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

package containers

import (
	"context"

	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

type containerEngineWrapperClient interface {
	StopContainer(ctx context.Context, id string) (jobId string, err error)
	RestartContainer(ctx context.Context, id string) (jobId string, err error)
	RemoveContainer(ctx context.Context, id string, force bool) error
	GetImage(ctx context.Context, id string) (external_models.CewImage, error)
	AddImage(ctx context.Context, img string) (jobId string, err error)
	RemoveVolume(ctx context.Context, id string, force bool) error
	GetJob(ctx context.Context, id string) (external_models.JobLibJob, error)
	CancelJob(ctx context.Context, id string) error
}
