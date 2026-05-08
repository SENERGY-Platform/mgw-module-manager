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

	helper_errors "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/errors"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

func (h *Handler) updateDeploymentsCache(
	ctx context.Context,
	moduleDependencies map[string]string,
	cacheDeployments map[string]deploymentsCacheItem,
) error {
	var idsNotInCache []string
	for moduleId := range moduleDependencies {
		if _, ok := cacheDeployments[moduleId]; !ok {
			idsNotInCache = append(idsNotInCache, moduleId)
		}
	}
	if len(idsNotInCache) == 0 {
		return nil
	}
	deployments, err := h.databaseHandler.ReadDeployments(ctx, pkg_models.DeploymentsFilter{ModuleIds: idsNotInCache})
	if err != nil {
		return err
	}
	deploymentsContainers, err := h.databaseHandler.ReadDeploymentsContainers(ctx, slices.Collect(maps.Keys(deployments)))
	if err != nil {
		return err
	}
	for id, deployment := range deployments {
		containers := make(map[string]containerCacheItem)
		for _, container := range deploymentsContainers[id] {
			containers[container.Reference] = containerCacheItem{
				Name:  container.Name,
				Alias: container.Alias,
			}
		}
		cacheDeployments[deployment.ModuleId] = deploymentsCacheItem{
			DeploymentId: id,
			Containers:   containers,
		}
	}
	var errs []error
	for _, id := range idsNotInCache {
		if _, ok := cacheDeployments[id]; !ok {
			errs = append(errs, errors.New(fmt.Sprintf("'%s' not found", id)))
		}
	}
	if len(errs) > 0 {
		return helper_errors.Join(errs...)
	}
	return nil
}
