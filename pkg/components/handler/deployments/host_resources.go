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
	"fmt"
	"maps"
	"strings"

	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) updateHostResourcesCache(
	ctx context.Context,
	userDataHostResources map[string]pkg_models.DeploymentHostResource,
	cacheHostResources map[string]external_models.HmHostResource,
) error {
	selectedIds := helper_slices.CollectFunc(maps.Values(userDataHostResources), func(item pkg_models.DeploymentHostResource) string {
		return item.Id
	})
	var idsNotInCache []string
	for _, id := range selectedIds {
		if _, ok := cacheHostResources[id]; ok {
			idsNotInCache = append(idsNotInCache, id)
		}
	}
	if len(idsNotInCache) == 0 {
		return nil
	}
	var errs []string
	for _, id := range idsNotInCache {
		hostResource, err := h.hostManagerClient.GetHostResource(ctx, id)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		cacheHostResources[hostResource.ID] = hostResource
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func getSelectedHostResources(
	moduleHostResources map[string]external_models.ModuleLibHostResource,
	userInputHostResources map[string]string,
	deploymentID string,
) (map[string]pkg_models.DeploymentHostResource, error) {
	hostResources := make(map[string]pkg_models.DeploymentHostResource)
	var errs []string
	for reference, hostResource := range moduleHostResources {
		id, ok := userInputHostResources[reference]
		if !ok {
			if hostResource.Required {
				errs = append(errs, fmt.Sprintf("missing required host resource '%s'", reference))
			}
			continue
		}
		hostResources[reference] = pkg_models.DeploymentHostResource{
			Id:           id,
			DeploymentId: deploymentID,
			Reference:    reference,
		}
	}
	return hostResources, nil
}
