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

package service

import (
	"context"
	"slices"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/models/constants"
	lib_external "github.com/SENERGY-Platform/mgw-module-manager/lib/models/external"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

func (s *Service) ServiceHealth(ctx context.Context) error {
	return s.databaseHandler.Ping(ctx)
}

func (s *Service) DeploymentsHealth(ctx context.Context, filter lib_models.DeploymentsHealthInfoFilter) (lib_models.DeploymentsHealthInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{deploymentJobSlotNum, moduleJobSlotNum})
	if len(currentJobs) > 0 {
		return lib_models.DeploymentsHealthInfo{}, lib_errors.New[lib_errors.ErrActiveJob](activeJobsErrMsg(currentJobs))
	}
	var moduleIds []string
	for _, id := range filter.ModuleIds {
		if !slices.Contains(filter.ExclModuleIds, id) {
			moduleIds = append(moduleIds, id)
		}
	}
	deployments, err := s.deploymentsHandler.GetReducedDeploymentsByModuleIds(ctx, pkg_models.DeploymentsFilterWithState{
		DeploymentsFilter: pkg_models.DeploymentsFilter{
			ModuleIds: moduleIds,
			Enabled:   1,
		},
	})
	if err != nil {
		return lib_models.DeploymentsHealthInfo{}, err
	}
	for _, id := range filter.ExclModuleIds {
		delete(deployments, id)
	}
	lenFilterAuxDepsOfIds := len(filter.AuxDeploymentsOfIds)
	lenFilterExclAuxDepsOfIds := len(filter.ExclAuxDeploymentsOfIds)
	auxDeployments := make(map[string]map[string]lib_models.AuxiliaryDeploymentReduced)
	for moduleId, deployment := range deployments {
		if lenFilterAuxDepsOfIds > 0 && !slices.Contains(filter.AuxDeploymentsOfIds, moduleId) || lenFilterExclAuxDepsOfIds > 0 && slices.Contains(filter.ExclAuxDeploymentsOfIds, moduleId) {
			continue
		}
		auxDeps, err := s.auxDeploymentsHandler.GetReducedDeployments(ctx, deployment.Id, lib_models.AuxiliaryDeploymentsFilterWithState{
			AuxiliaryDeploymentsFilter: lib_models.AuxiliaryDeploymentsFilter{
				Enabled: 1,
			},
		})
		if err != nil {
			return lib_models.DeploymentsHealthInfo{}, err
		}
		auxDeployments[moduleId] = auxDeps
	}
	return getDeploymentsHealthInfo(deployments, auxDeployments, filter.IncludeHealthy), nil
}

func getDeploymentsHealthInfo(
	deployments map[string]pkg_models.DeploymentReduced,
	auxDeployments map[string]map[string]lib_models.AuxiliaryDeploymentReduced,
	includeHealthy bool,
) lib_models.DeploymentsHealthInfo {
	healthInfo := lib_models.DeploymentsHealthInfo{
		TotalEnabledDeployments: len(deployments),
	}
	for id, deployment := range deployments {
		var auxDepsState int
		var auxDepsHealthInfo []lib_models.AuxiliaryDeploymentHealthInfo
		auxDeps, ok := auxDeployments[id]
		if ok {
			var notOk uint
			for _, auxDep := range auxDeps {
				if !containerOk(auxDep.Container.State, auxDep.Container.Health) {
					notOk++
				} else if !includeHealthy {
					continue
				}
				auxDepsHealthInfo = append(auxDepsHealthInfo, lib_models.AuxiliaryDeploymentHealthInfo{
					Id:        auxDep.Id,
					Reference: auxDep.Reference,
					Container: lib_models.ContainerHealthInfo{
						State:  auxDep.Container.State,
						Health: auxDep.Container.Health,
					},
				})
			}
			if notOk == 0 {
				auxDepsState = lib_constants.DeploymentStateHealthy
			} else {
				auxDepsState = lib_constants.DeploymentStateUnhealthy
			}
		}
		if !includeHealthy && deployment.State < 2 && auxDepsState < 2 {
			continue
		}
		depHealth := lib_models.DeploymentHealthInfo{
			Id:                               deployment.ModuleId,
			State:                            deployment.State,
			TotalContainers:                  len(deployment.Containers),
			AuxiliaryDeployments:             auxDepsHealthInfo,
			TotalEnabledAuxiliaryDeployments: len(auxDeps),
			AuxiliaryDeploymentsState:        auxDepsState,
		}
		for _, container := range deployment.Containers {
			if !includeHealthy && containerOk(container.State, container.Health) {
				continue
			}
			depHealth.Containers = append(depHealth.Containers, lib_models.DeploymentContainerHealthInfo{
				Reference: container.Reference,
				ContainerHealthInfo: lib_models.ContainerHealthInfo{
					State:  container.State,
					Health: container.Health,
				},
			})
		}
		healthInfo.Deployments = append(healthInfo.Deployments, depHealth)
	}
	return healthInfo
}

func containerOk(state, health string) bool {
	if health != "" {
		return health == lib_external.CewHealthyState
	}
	return state == lib_external.CewRunningState || state == lib_external.CewRestartingState || state == lib_external.CewTransitionState
}
