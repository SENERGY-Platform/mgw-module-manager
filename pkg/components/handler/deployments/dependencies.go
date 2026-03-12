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

func newExternalDependenciesCache(deploymentWrappers map[string]*deploymentWrapper) map[string]map[string]models_handler_storage.DeploymentContainer {
	externalDependenciesCache := make(map[string]map[string]models_handler_storage.DeploymentContainer)
	for moduleId, deployment := range deploymentWrappers {
		if deployment == nil {
			continue
		}
		tmp := make(map[string]models_handler_storage.DeploymentContainer)
		for reference, container := range deployment.Containers {
			tmp[reference] = container.DeploymentContainer
		}
		externalDependenciesCache[moduleId] = tmp
	}
	return externalDependenciesCache
}

func (h *Handler) updateExternalDependenciesCache(
	ctx context.Context,
	externalDependenciesCache map[string]map[string]models_handler_storage.DeploymentContainer,
	moduleDependencies map[string]string,
) error {
	moduleIds := slices.Collect(maps.Keys(moduleDependencies))
	var idsNotInCache []string
	for _, id := range moduleIds {
		if _, ok := externalDependenciesCache[id]; !ok {
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
		tmp := make(map[string]models_handler_storage.DeploymentContainer)
		containers := deploymentsContainers[id]
		for _, container := range containers {
			tmp[container.Reference] = container
		}
		externalDependenciesCache[deployment.ModuleId] = tmp
	}
	var errs []string
	for _, id := range idsNotInCache {
		if _, ok := externalDependenciesCache[id]; !ok {
			errs = append(errs, fmt.Sprintf("dependency %v not found", id))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}
