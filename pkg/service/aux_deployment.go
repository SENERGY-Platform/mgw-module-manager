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
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/SENERGY-Platform/mgw-module-manager/lib/models/aux_deployments"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/lib/models/results"
	models_service2 "github.com/SENERGY-Platform/mgw-module-manager/lib/models/service"
	models_handler_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/aux_deployments"
	models_handler_database "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
	models_handler_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployments"
	models_handler_modules "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/modules"
)

func (s *Service) GetAuxiliaryDeployment(
	ctx context.Context,
	deploymentId string,
	auxDeploymentId string,
) (models_handler_aux_deployments.AuxiliaryDeployment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.auxDeploymentsHandler.GetDeployment(ctx, deploymentId, auxDeploymentId)
}

func (s *Service) GetAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models_handler_aux_deployments.AuxiliaryDeploymentsFilterWithState,
) (map[string]models_handler_aux_deployments.AuxiliaryDeployment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.auxDeploymentsHandler.GetDeployments(ctx, deploymentId, filter)
}

func (s *Service) GetReducedAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models_handler_aux_deployments.AuxiliaryDeploymentsFilterWithState,
) (map[string]models_handler_aux_deployments.AuxiliaryDeploymentReduced, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.auxDeploymentsHandler.GetReducedDeployments(ctx, deploymentId, filter)
}

func (s *Service) CreateAuxiliaryDeployment(
	ctx context.Context,
	serviceInput models_service2.ServiceInput,
) (models_service2.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{deploymentJobSlotNum, moduleJobSlotNum})
	if len(currentJobs) > 0 {
		return models_service2.Job{}, errors.New("active jobs") // TODO
	}
	activeDeployment, err := s.deploymentsHandler.GetDeployment(ctx, serviceInput.DeploymentId)
	if err != nil {
		return models_service2.Job{}, err
	}
	module, err := s.modulesHandler.Module(ctx, activeDeployment.ModuleId)
	if err != nil {
		return models_service2.Job{}, err
	}
	dependencyDeployments, err := s.deploymentsHandler.GetReducedDeploymentsByModuleIds(ctx, models_handler_deployments.DeploymentsFilter{
		DeploymentsFilter: models_handler_database.DeploymentsFilter{
			ModuleIds: slices.Collect(maps.Keys(module.Dependencies)),
		},
	})
	if err != nil {
		return models_service2.Job{}, err
	}
	job, err := s.jobsHandler.CreateJob("create auxiliary deployment")
	if err != nil {
		return models_service2.Job{}, err
	}
	go func() {
		defer job.Done()
		jobResult := models_service2.JobResultCreateAuxiliaryDeployment{
			JobResult: models_service2.JobResult{JobId: job.Id},
		}
		defer func() {
			if err := recover(); err != nil {
				jobResult.ErrorResult = models_error.NewErrorResult(fmt.Sprintf("panic: %v", err))
				s.setCreateAuxiliaryDeploymentJobResult(job.Id, jobResult)
			}
		}()
		jobResult.Result, err = s.auxDeploymentsHandler.CreateDeployment(
			job.Context(),
			module,
			activeDeployment,
			dependencyDeployments,
			serviceInput.ServiceInput,
		)
		if err != nil {
			jobResult.ErrorResult = models_error.NewErrorResult(err.Error())
		}
		s.setCreateAuxiliaryDeploymentJobResult(job.Id, jobResult)
	}()
	return models_service2.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) UpdateAuxiliaryDeployment(
	ctx context.Context,
	serviceInput models_service2.ServiceInputUpdate,
) (models_service2.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{deploymentJobSlotNum, moduleJobSlotNum})
	if len(currentJobs) > 0 {
		return models_service2.Job{}, errors.New("active jobs") // TODO
	}
	activeDeployment, err := s.deploymentsHandler.GetDeployment(ctx, serviceInput.DeploymentId)
	if err != nil {
		return models_service2.Job{}, err
	}
	module, err := s.modulesHandler.Module(ctx, activeDeployment.ModuleId)
	if err != nil {
		return models_service2.Job{}, err
	}
	dependencyDeployments, err := s.deploymentsHandler.GetReducedDeploymentsByModuleIds(ctx, models_handler_deployments.DeploymentsFilter{
		DeploymentsFilter: models_handler_database.DeploymentsFilter{
			ModuleIds: slices.Collect(maps.Keys(module.Dependencies)),
		},
	})
	if err != nil {
		return models_service2.Job{}, err
	}
	job, err := s.jobsHandler.CreateJob("update auxiliary deployment")
	if err != nil {
		return models_service2.Job{}, err
	}
	go func() {
		defer job.Done()
		jobResult := models_service2.JobResult{JobId: job.Id}
		defer func() {
			if err := recover(); err != nil {
				jobResult.ErrorResult = models_error.NewErrorResult(fmt.Sprintf("panic: %v", err))
				s.setUpdateAuxiliaryDeploymentJobResult(job.Id, jobResult)
			}
		}()
		err = s.auxDeploymentsHandler.UpdateDeployment(
			job.Context(),
			module,
			activeDeployment,
			dependencyDeployments,
			serviceInput.AuxDeploymentId,
			serviceInput.UpdateServiceInput,
		)
		if err != nil {
			jobResult.ErrorResult = models_error.NewErrorResult(err.Error())
		}
		s.setUpdateAuxiliaryDeploymentJobResult(job.Id, jobResult)
	}()
	return models_service2.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) RecreateAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models_handler_aux_deployments.AuxiliaryDeploymentsFilterWithState,
) (models_service2.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{deploymentJobSlotNum, moduleJobSlotNum})
	if len(currentJobs) > 0 {
		return models_service2.Job{}, errors.New("active jobs") // TODO
	}
	activeDeployment, err := s.deploymentsHandler.GetDeployment(ctx, deploymentId)
	if err != nil {
		return models_service2.Job{}, err
	}
	module, err := s.modulesHandler.Module(ctx, activeDeployment.ModuleId)
	if err != nil {
		return models_service2.Job{}, err
	}
	dependencyDeployments, err := s.deploymentsHandler.GetReducedDeploymentsByModuleIds(ctx, models_handler_deployments.DeploymentsFilter{
		DeploymentsFilter: models_handler_database.DeploymentsFilter{
			ModuleIds: slices.Collect(maps.Keys(module.Dependencies)),
		},
	})
	if err != nil {
		return models_service2.Job{}, err
	}
	job, err := s.jobsHandler.CreateJob("recreate auxiliary deployments")
	if err != nil {
		return models_service2.Job{}, err
	}
	go func() {
		defer job.Done()
		jobResult := models_service2.JobResultAuxiliaryDeployments{
			JobResult: models_service2.JobResult{JobId: job.Id},
		}
		defer func() {
			if err := recover(); err != nil {
				jobResult.ErrorResult = models_error.NewErrorResult(fmt.Sprintf("panic: %v", err))
				s.setAuxiliaryDeploymentsJobResult(job.Id, jobResult)
			}
		}()
		jobResult.Results, err = s.auxDeploymentsHandler.RecreateDeployments(
			job.Context(),
			module,
			activeDeployment,
			dependencyDeployments,
			filter,
		)
		if err != nil {
			jobResult.ErrorResult = models_error.NewErrorResult(err.Error())
		}
		for _, res := range jobResult.Results {
			if res.HasError {
				jobResult.ResultsErrNum++
			}
		}
		s.setAuxiliaryDeploymentsJobResult(job.Id, jobResult)
	}()
	return models_service2.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) DeleteAuxiliaryDeployments(
	_ context.Context,
	deploymentId string,
	filter models_handler_aux_deployments.AuxiliaryDeploymentsFilterWithState,
	allowAll bool,
) (models_service2.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, err := s.jobsHandler.CreateJob("delete auxiliary deployments")
	if err != nil {
		return models_service2.Job{}, err
	}
	go func() {
		defer job.Done()
		jobResult := models_service2.JobResultAuxiliaryDeployments{
			JobResult: models_service2.JobResult{JobId: job.Id},
		}
		defer func() {
			if err := recover(); err != nil {
				jobResult.ErrorResult = models_error.NewErrorResult(fmt.Sprintf("panic: %v", err))
				s.setAuxiliaryDeploymentsJobResult(job.Id, jobResult)
			}
		}()
		jobResult.Results, err = s.auxDeploymentsHandler.DeleteDeployments(
			job.Context(),
			deploymentId,
			filter,
			allowAll,
		)
		if err != nil {
			jobResult.ErrorResult = models_error.NewErrorResult(err.Error())
		}
		for _, res := range jobResult.Results {
			if res.HasError {
				jobResult.ResultsErrNum++
			}
		}
		s.setAuxiliaryDeploymentsJobResult(job.Id, jobResult)
	}()
	return models_service2.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) EnableAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models_handler_aux_deployments.AuxiliaryDeploymentsFilterWithState,
) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.auxDeploymentsHandler.EnableDeployments(ctx, deploymentId, filter)
}

func (s *Service) DisableAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models_handler_aux_deployments.AuxiliaryDeploymentsFilterWithState,
) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.auxDeploymentsHandler.DisableDeployments(ctx, deploymentId, filter)
}

func (s *Service) GetAuxiliaryDeploymentVolumes(
	ctx context.Context,
	deploymentId string,
	filterReferences []string,
) (map[string]models_handler_database.AuxiliaryDeploymentVolume, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.auxDeploymentsHandler.GetVolumes(ctx, deploymentId, filterReferences)
}

func (s *Service) GetAuxiliaryDeploymentVolumesWithMounts(
	ctx context.Context,
	deploymentId string,
	filterReferences []string,
) (map[string]models_handler_database.AuxiliaryDeploymentVolumeWithMounts, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.auxDeploymentsHandler.GetVolumesWithMounts(ctx, deploymentId, filterReferences)
}

func (s *Service) DeleteAuxiliaryDeploymentVolumes(
	ctx context.Context,
	deploymentId string,
	filterReferences []string,
	allowAll bool,
) ([]aux_deployments.VolumeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.auxDeploymentsHandler.DeleteVolumes(ctx, deploymentId, filterReferences, allowAll)
}

func (s *Service) DeleteUnusedAuxiliaryDeploymentVolumes(
	ctx context.Context,
	deploymentId string,
	excludeReferences []string,
) ([]aux_deployments.VolumeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.auxDeploymentsHandler.DeleteUnusedVolumes(ctx, deploymentId, excludeReferences)
}

func (s *Service) recreateAuxDeployments(
	ctx context.Context,
	module models_handler_modules.Module,
	deploymentId string,
	cacheDependencyDeployments map[string]models_handler_deployments.DeploymentReduced,
) ([]aux_deployments.BatchResult, error) {
	activeDeployment, err := s.deploymentsHandler.GetDeployment(ctx, deploymentId)
	if err != nil {
		return nil, err
	}
	var idsNotInCache []string
	for id := range module.Dependencies {
		_, ok := cacheDependencyDeployments[id]
		if !ok {
			idsNotInCache = append(idsNotInCache, id)
		}
	}
	if len(idsNotInCache) > 0 {
		dependencyDeployments, err := s.deploymentsHandler.GetReducedDeploymentsByModuleIds(ctx, models_handler_deployments.DeploymentsFilter{
			DeploymentsFilter: models_handler_database.DeploymentsFilter{
				ModuleIds: idsNotInCache,
			},
		})
		if err != nil {
			return nil, err
		}
		maps.Copy(cacheDependencyDeployments, dependencyDeployments)
	}
	return s.auxDeploymentsHandler.RecreateDeployments(
		ctx,
		module,
		activeDeployment,
		cacheDependencyDeployments,
		models_handler_aux_deployments.AuxiliaryDeploymentsFilterWithState{
			AuxiliaryDeploymentsFilter: models_handler_aux_deployments.AuxiliaryDeploymentsFilter{
				Recreate: 1,
			},
		},
	)
}
