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

package handler_deployments

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"

	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_database "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

func (h *Handler) updateHostResourcesCache(
	ctx context.Context,
	userDataHostResources map[string]models_handler_database.DeploymentHostResource,
	cacheHostResources map[string]models_external.HostResource,
) error {
	selectedIds := helper_slices.CollectFunc(maps.Values(userDataHostResources), func(item models_handler_database.DeploymentHostResource) string {
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
	moduleHostResources map[string]models_external.ModuleLibHostResource,
	userInputHostResources map[string]string,
	deploymentID string,
) (map[string]models_handler_database.DeploymentHostResource, error) {
	hostResources := make(map[string]models_handler_database.DeploymentHostResource)
	var errs []string
	for reference, hostResource := range moduleHostResources {
		id, ok := userInputHostResources[reference]
		if !ok {
			if hostResource.Required {
				errs = append(errs, fmt.Sprintf("missing required host resource '%s'", reference))
			}
			continue
		}
		hostResources[reference] = models_handler_database.DeploymentHostResource{
			Id:           id,
			DeploymentId: deploymentID,
			Reference:    reference,
		}
	}
	return hostResources, nil
}
