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

package deployments

import (
	"context"
	"errors"
	"strings"

	helper_job "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/job"
	helper_url "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/url"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
)

func (h *Handler) ensureContainerImages(ctx context.Context, module models_handler_module.Module) error {
	imageNames := make(map[string]struct{})
	for _, service := range module.Services {
		imageNames[service.Image] = struct{}{}
	}
	var errs []string
	for imageName := range imageNames {
		err := h.ensureContainerImage(ctx, imageName)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func (h *Handler) ensureContainerImage(ctx context.Context, imageName string) error {
	_, err := h.cewClient.GetImage(ctx, helper_url.EscapePath(imageName, h.config.PathEscapeDepth))
	if err != nil {
		var notFoundErr *models_external.CEWNotFoundErr
		if !errors.As(err, &notFoundErr) {
			return err
		}
	} else {
		return nil
	}
	jobId, err := h.cewClient.AddImage(ctx, imageName)
	if err != nil {
		return err
	}
	job, err := helper_job.Await(ctx, h.cewClient, jobId, h.config.JobPollInterval)
	if err != nil {
		return err
	}
	if job.Error != nil {
		return errors.New(job.Error.Message)
	}
	return nil
}
