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
	"fmt"
	"maps"
	"slices"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
)

func (s *Service) GetAuxiliaryDeployment(
	ctx context.Context,
	deploymentId string,
	auxDeploymentId string,
) (lib_models.AuxiliaryDeployment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.auxDeploymentsHandler.GetDeployment(ctx, deploymentId, auxDeploymentId)
}

func (s *Service) GetAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter lib_models.AuxiliaryDeploymentsFilterWithState,
) (map[string]lib_models.AuxiliaryDeployment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.auxDeploymentsHandler.GetDeployments(ctx, deploymentId, filter)
}

func (s *Service) GetReducedAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter lib_models.AuxiliaryDeploymentsFilterWithState,
) (map[string]lib_models.AuxiliaryDeploymentReduced, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.auxDeploymentsHandler.GetReducedDeployments(ctx, deploymentId, filter)
}

func (s *Service) CreateAuxiliaryDeployment(
	ctx context.Context,
	serviceInput lib_models.AuxiliaryDeploymentInput,
) (lib_models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{deploymentJobSlotNum, moduleJobSlotNum})
	if len(currentJobs) > 0 {
		return lib_models.Job{}, lib_errors.New[lib_errors.ErrActiveJob](activeJobsErrMsg(currentJobs))
	}
	activeDeployment, err := s.deploymentsHandler.GetDeployment(ctx, serviceInput.DeploymentId)
	if err != nil {
		return lib_models.Job{}, err
	}
	module, err := s.modulesHandler.GetModule(ctx, activeDeployment.ModuleId)
	if err != nil {
		return lib_models.Job{}, err
	}
	dependencyDeployments, err := s.deploymentsHandler.GetReducedDeploymentsByModuleIds(ctx, pkg_models.DeploymentsFilterWithState{
		DeploymentsFilter: pkg_models.DeploymentsFilter{
			ModuleIds: slices.Collect(maps.Keys(module.Dependencies)),
		},
	})
	if err != nil {
		return lib_models.Job{}, err
	}
	job, err := s.jobsHandler.CreateJob("create auxiliary deployment")
	if err != nil {
		return lib_models.Job{}, err
	}
	go func() {
		defer func() {
			job.Done()
			logJobDone(ctx, job)
		}()
		logJobStart(ctx, job)
		jobResult := lib_models.AuxiliaryDeploymentCreateJobResult{
			JobResult: lib_models.JobResult{JobId: job.Id},
		}
		defer func() {
			if st := recover(); st != nil {
				jobResult.ErrorResult = lib_models.NewErrorResult(fmt.Sprintf("%v", st))
				s.setCreateAuxiliaryDeploymentJobResult(job.Id, jobResult)
				logger.ErrorContext(
					ctx,
					"create auxiliary deployment",
					slog_keys.JobId, job.Id,
					slog_keys.ModuleId, module.ID,
					slog_keys.DeploymentId, serviceInput.DeploymentId,
					slog_keys.Reference, serviceInput.Reference,
					slog_keys.Error, "panic",
					slog_keys.StackTrace, st,
				)
			}
		}()
		jobResult.AuxiliaryDeploymentResult, err = s.auxDeploymentsHandler.CreateDeployment(
			job.Context(),
			module,
			activeDeployment,
			dependencyDeployments,
			serviceInput.AuxiliaryDeploymentInputBase,
		)
		if err != nil {
			jobResult.ErrorResult = lib_models.NewErrorResult(err.Error())
		}
		s.setCreateAuxiliaryDeploymentJobResult(job.Id, jobResult)
	}()
	return lib_models.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) UpdateAuxiliaryDeployment(
	ctx context.Context,
	serviceInput lib_models.AuxiliaryDeploymentUpdateInput,
) (lib_models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{deploymentJobSlotNum, moduleJobSlotNum})
	if len(currentJobs) > 0 {
		return lib_models.Job{}, lib_errors.New[lib_errors.ErrActiveJob](activeJobsErrMsg(currentJobs))
	}
	activeDeployment, err := s.deploymentsHandler.GetDeployment(ctx, serviceInput.DeploymentId)
	if err != nil {
		return lib_models.Job{}, err
	}
	module, err := s.modulesHandler.GetModule(ctx, activeDeployment.ModuleId)
	if err != nil {
		return lib_models.Job{}, err
	}
	dependencyDeployments, err := s.deploymentsHandler.GetReducedDeploymentsByModuleIds(ctx, pkg_models.DeploymentsFilterWithState{
		DeploymentsFilter: pkg_models.DeploymentsFilter{
			ModuleIds: slices.Collect(maps.Keys(module.Dependencies)),
		},
	})
	if err != nil {
		return lib_models.Job{}, err
	}
	job, err := s.jobsHandler.CreateJob("update auxiliary deployment")
	if err != nil {
		return lib_models.Job{}, err
	}
	go func() {
		defer func() {
			job.Done()
			logJobDone(ctx, job)
		}()
		logJobStart(ctx, job)
		jobResult := lib_models.JobResult{JobId: job.Id}
		defer func() {
			if st := recover(); st != nil {
				jobResult.ErrorResult = lib_models.NewErrorResult(fmt.Sprintf("%v", st))
				s.setUpdateAuxiliaryDeploymentJobResult(job.Id, jobResult)
				logger.ErrorContext(
					ctx,
					"update auxiliary deployment",
					slog_keys.JobId, job.Id,
					slog_keys.ModuleId, module.ID,
					slog_keys.DeploymentId, serviceInput.DeploymentId,
					slog_keys.Reference, serviceInput.Reference,
					slog_keys.Error, "panic",
					slog_keys.StackTrace, st,
				)
			}
		}()
		err = s.auxDeploymentsHandler.UpdateDeployment(
			job.Context(),
			module,
			activeDeployment,
			dependencyDeployments,
			serviceInput.AuxDeploymentId,
			serviceInput.AuxiliaryDeploymentUpdateInputBase,
		)
		if err != nil {
			jobResult.ErrorResult = lib_models.NewErrorResult(err.Error())
		}
		s.setUpdateAuxiliaryDeploymentJobResult(job.Id, jobResult)
	}()
	return lib_models.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) RecreateAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter lib_models.AuxiliaryDeploymentsFilterWithState,
) (lib_models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{deploymentJobSlotNum, moduleJobSlotNum})
	if len(currentJobs) > 0 {
		return lib_models.Job{}, lib_errors.New[lib_errors.ErrActiveJob](activeJobsErrMsg(currentJobs))
	}
	activeDeployment, err := s.deploymentsHandler.GetDeployment(ctx, deploymentId)
	if err != nil {
		return lib_models.Job{}, err
	}
	module, err := s.modulesHandler.GetModule(ctx, activeDeployment.ModuleId)
	if err != nil {
		return lib_models.Job{}, err
	}
	dependencyDeployments, err := s.deploymentsHandler.GetReducedDeploymentsByModuleIds(ctx, pkg_models.DeploymentsFilterWithState{
		DeploymentsFilter: pkg_models.DeploymentsFilter{
			ModuleIds: slices.Collect(maps.Keys(module.Dependencies)),
		},
	})
	if err != nil {
		return lib_models.Job{}, err
	}
	job, err := s.jobsHandler.CreateJob("recreate auxiliary deployments")
	if err != nil {
		return lib_models.Job{}, err
	}
	go func() {
		defer func() {
			job.Done()
			logJobDone(ctx, job)
		}()
		logJobStart(ctx, job)
		jobResult := lib_models.AuxiliaryDeploymentJobResult{
			JobResult: lib_models.JobResult{JobId: job.Id},
		}
		defer func() {
			if st := recover(); st != nil {
				jobResult.ErrorResult = lib_models.NewErrorResult(fmt.Sprintf("%v", st))
				s.setAuxiliaryDeploymentsJobResult(job.Id, jobResult)
				logger.ErrorContext(
					ctx,
					"recreate auxiliary deployments",
					slog_keys.JobId, job.Id,
					slog_keys.ModuleId, module.ID,
					slog_keys.DeploymentId, deploymentId,
					slog_keys.Error, "panic",
					slog_keys.StackTrace, st,
				)
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
			jobResult.ErrorResult = lib_models.NewErrorResult(err.Error())
		}
		for _, res := range jobResult.Results {
			if res.HasError {
				jobResult.ResultsErrNum++
			}
		}
		s.setAuxiliaryDeploymentsJobResult(job.Id, jobResult)
	}()
	return lib_models.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) DeleteAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter lib_models.AuxiliaryDeploymentsFilterWithState,
	allowAll bool,
) (lib_models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, err := s.jobsHandler.CreateJob("delete auxiliary deployments")
	if err != nil {
		return lib_models.Job{}, err
	}
	go func() {
		defer func() {
			job.Done()
			logJobDone(ctx, job)
		}()
		logJobStart(ctx, job)
		jobResult := lib_models.AuxiliaryDeploymentJobResult{
			JobResult: lib_models.JobResult{JobId: job.Id},
		}
		defer func() {
			if st := recover(); st != nil {
				jobResult.ErrorResult = lib_models.NewErrorResult(fmt.Sprintf("%v", st))
				s.setAuxiliaryDeploymentsJobResult(job.Id, jobResult)
				logger.ErrorContext(
					ctx,
					"delete auxiliary deployments",
					slog_keys.JobId, job.Id,
					slog_keys.DeploymentId, deploymentId,
					slog_keys.Error, "panic",
					slog_keys.StackTrace, st,
				)
			}
		}()
		jobResult.Results, err = s.auxDeploymentsHandler.DeleteDeployments(
			job.Context(),
			deploymentId,
			filter,
			allowAll,
		)
		if err != nil {
			jobResult.ErrorResult = lib_models.NewErrorResult(err.Error())
		}
		for _, res := range jobResult.Results {
			if res.HasError {
				jobResult.ResultsErrNum++
			}
		}
		s.setAuxiliaryDeploymentsJobResult(job.Id, jobResult)
	}()
	return lib_models.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) EnableAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter lib_models.AuxiliaryDeploymentsFilterWithState,
) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.auxDeploymentsHandler.EnableDeployments(ctx, deploymentId, filter)
}

func (s *Service) DisableAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter lib_models.AuxiliaryDeploymentsFilterWithState,
) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.auxDeploymentsHandler.DisableDeployments(ctx, deploymentId, filter)
}

func (s *Service) GetAuxiliaryDeploymentVolumes(
	ctx context.Context,
	deploymentId string,
	filterReferences []string,
) (map[string]lib_models.AuxiliaryDeploymentVolume, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.auxDeploymentsHandler.GetVolumes(ctx, deploymentId, filterReferences)
}

func (s *Service) GetAuxiliaryDeploymentVolumesWithMounts(
	ctx context.Context,
	deploymentId string,
	filterReferences []string,
) (map[string]lib_models.AuxiliaryDeploymentVolumeWithMounts, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.auxDeploymentsHandler.GetVolumesWithMounts(ctx, deploymentId, filterReferences)
}

func (s *Service) DeleteAuxiliaryDeploymentVolumes(
	ctx context.Context,
	deploymentId string,
	filterReferences []string,
	allowAll bool,
) ([]lib_models.AuxiliaryDeploymentVolumeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.auxDeploymentsHandler.DeleteVolumes(ctx, deploymentId, filterReferences, allowAll)
}

func (s *Service) DeleteUnusedAuxiliaryDeploymentVolumes(
	ctx context.Context,
	deploymentId string,
	excludeReferences []string,
) ([]lib_models.AuxiliaryDeploymentVolumeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.auxDeploymentsHandler.DeleteUnusedVolumes(ctx, deploymentId, excludeReferences)
}

func (s *Service) recreateAuxDeployments(
	ctx context.Context,
	module pkg_models.Module,
	deploymentId string,
	cacheDependencyDeployments map[string]pkg_models.DeploymentReduced,
) ([]lib_models.AuxiliaryDeploymentBatchResult, error) {
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
		dependencyDeployments, err := s.deploymentsHandler.GetReducedDeploymentsByModuleIds(ctx, pkg_models.DeploymentsFilterWithState{
			DeploymentsFilter: pkg_models.DeploymentsFilter{
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
		lib_models.AuxiliaryDeploymentsFilterWithState{
			AuxiliaryDeploymentsFilter: lib_models.AuxiliaryDeploymentsFilter{
				Recreate: 1,
			},
		},
	)
}
