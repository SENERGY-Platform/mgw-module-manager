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

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployments"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
)

func (s *Service) CreateAuxiliaryDeployment(
	ctx context.Context,
	serviceInput models_service.ServiceInput,
) (models_service.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{deploymentJobSlotNum, moduleJobSlotNum})
	if len(currentJobs) > 0 {
		return models_service.Job{}, errors.New("active jobs") // TODO
	}
	activeDeployment, err := s.deploymentsHandler.GetDeployment(ctx, serviceInput.DeploymentId)
	if err != nil {
		return models_service.Job{}, err
	}
	module, err := s.modulesHandler.Module(ctx, activeDeployment.ModuleId)
	if err != nil {
		return models_service.Job{}, err
	}
	dependencyDeployments, err := s.deploymentsHandler.GetReducedDeploymentsByModuleIds(ctx, models_handler_deployments.DeploymentsFilter{
		DeploymentsFilter: models_handler_database.DeploymentsFilter{
			ModuleIds: slices.Collect(maps.Keys(module.Dependencies)),
		},
	})
	job, err := s.jobsHandler.CreateJob("create auxiliary deployment")
	if err != nil {
		return models_service.Job{}, err
	}
	go func() {
		defer job.Done()
		jobResult := models_service.JobResultCreateAuxiliaryDeployment{
			JobResult: models_service.JobResult{JobId: job.Id},
		}
		defer func() {
			if err := recover(); err != nil {
				jobResult.ErrorResult = models_error.NewErrorResult(fmt.Sprintf("panic: %v", err))
				s.jobResults.setCreateAuxiliaryDeploymentResult(job.Id, jobResult)
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
		s.jobResults.setCreateAuxiliaryDeploymentResult(job.Id, jobResult)
	}()
	return models_service.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}
