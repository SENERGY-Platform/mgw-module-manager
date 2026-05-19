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
	"fmt"
	"time"

	helper_containers "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/containers"
	helper_errors "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/errors"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) ensureContainerImages(ctx context.Context, moduleServices map[string]external_models.ModuleLibService) error {
	imageNames := make(map[string]struct{})
	for _, service := range moduleServices {
		imageNames[service.Image] = struct{}{}
	}
	var errs []error
	for imageName := range imageNames {
		err := helper_containers.EnsureImage(
			ctx,
			h.containerEngineWrapperClient,
			imageName,
			false,
			h.config.PathEscapeDepth,
			time.Duration(h.config.JobPollInterval),
		)
		if err != nil {
			errs = append(errs, fmt.Errorf("'%s' %w", imageName, err))
		}
	}
	if len(errs) > 0 {
		return helper_errors.Join(errs...)
	}
	return nil
}
