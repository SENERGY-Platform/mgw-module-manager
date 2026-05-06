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
	"errors"
	"time"

	helper_job "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/job"
	helper_url "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/url"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

func Stop(ctx context.Context, client containerEngineWrapperClient, containerId string, jobPollInterval time.Duration) error {
	jobId, err := client.StopContainer(ctx, containerId)
	if err != nil {
		return err
	}
	job, err := helper_job.Await(ctx, client, jobId, jobPollInterval)
	if err != nil {
		return err
	}
	if job.Error != nil {
		return errors.New(job.Error.Message)
	}
	return nil
}

func Restart(ctx context.Context, client containerEngineWrapperClient, containerId string, jobPollInterval time.Duration) error {
	jobId, err := client.RestartContainer(ctx, containerId)
	if err != nil {
		return err
	}
	job, err := helper_job.Await(ctx, client, jobId, jobPollInterval)
	if err != nil {
		return err
	}
	if job.Error != nil {
		return errors.New(job.Error.Message)
	}
	return nil
}

func Remove(ctx context.Context, client containerEngineWrapperClient, containerId string) error {
	err := client.RemoveContainer(ctx, containerId, true)
	if err != nil {
		var notFoundErr *pkg_models.CewNotFoundErr
		if !errors.As(err, &notFoundErr) {
			return err
		}
	}
	return nil
}

func EnsureImage(
	ctx context.Context,
	client containerEngineWrapperClient,
	imageName string,
	forcePull bool,
	pathEscapeDepth int,
	jobPollInterval time.Duration,
) error {
	if !forcePull {
		_, err := client.GetImage(ctx, helper_url.EscapePath(imageName, pathEscapeDepth))
		if err != nil {
			var notFoundErr *pkg_models.CewNotFoundErr
			if !errors.As(err, &notFoundErr) {
				return err
			}
		} else {
			return nil
		}
	}
	jobId, err := client.AddImage(ctx, imageName)
	if err != nil {
		return err
	}
	job, err := helper_job.Await(ctx, client, jobId, jobPollInterval)
	if err != nil {
		return err
	}
	if job.Error != nil {
		return errors.New(job.Error.Message)
	}
	return nil
}

func RemoveVolume(ctx context.Context, client containerEngineWrapperClient, name string) error {
	err := client.RemoveVolume(ctx, name, false)
	if err != nil {
		var notFoundErr *pkg_models.CewNotFoundErr
		if !errors.As(err, &notFoundErr) {
			return err
		}
	}
	return nil
}
