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
	"slices"
	"strings"

	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) updateDependenciesCache(
	ctx context.Context,
	dependenciesCache map[string]deploymentWrapper,
	moduleDependencies map[string]string,
) error {
	moduleIds := slices.Collect(maps.Keys(moduleDependencies))
	var idsNotInCache []string
	for _, id := range moduleIds {
		if _, ok := dependenciesCache[id]; !ok {
			idsNotInCache = append(idsNotInCache, id)
		}
	}
	if len(idsNotInCache) == 0 {
		return nil
	}
	deployments, err := h.storageHdl.ReadDeployments(ctx, models_handler_storage.DeploymentsFilter{ModuleIds: idsNotInCache})
	if err != nil {
		return err
	}
	deploymentsContainers, err := h.storageHdl.ReadDeploymentsContainers(ctx, slices.Collect(maps.Keys(deployments)))
	if err != nil {
		return err
	}
	for id, deployment := range deployments {
		containers := make(map[string]containerWrapper)
		deploymentContainers := deploymentsContainers[id]
		for _, container := range deploymentContainers {
			containers[container.Reference] = containerWrapper{
				DeploymentContainer: container,
			}
		}
		dependenciesCache[deployment.ModuleId] = deploymentWrapper{
			Deployment: deployment,
			Containers: containers,
		}
	}
	var errs []string
	for _, id := range idsNotInCache {
		if _, ok := dependenciesCache[id]; !ok {
			errs = append(errs, fmt.Sprintf("dependency %v not found", id))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}
