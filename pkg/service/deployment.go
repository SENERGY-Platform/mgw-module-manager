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
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/configs"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/config"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployments"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/modules"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
)

func (s *Service) DeploymentRequest(ctx context.Context, moduleIds []string) ([]models_service.Module, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.jobsHandler.CurrentSlotJob(moduleJobSlotNum)
	if ok {
		return nil, errors.New("active job") // TODO
	}
	handlerModules, err := s.modulesHandler.Modules(ctx, models_handler_modules.ModuleFilter{
		Ids:          moduleIds,
		Dependencies: true,
	})
	if err != nil {
		return nil, err
	}
	handlerDeployments, err := s.deploymentsHandler.GetDeploymentsByModuleIds(ctx, models_handler_deployments.DeploymentsFilter{
		DeploymentsFilter: models_handler_database.DeploymentsFilter{
			ModuleIds: slices.Collect(maps.Keys(handlerModules)),
		},
	})
	var modules []models_service.Module
	for id, handlerModule := range handlerModules {
		_, ok := handlerDeployments[id]
		if !ok {
			modules = append(modules, getModule(handlerModule, models_handler_deployments.Deployment{}))
		}
	}
	return modules, nil
}

func (s *Service) CreateDeployments(ctx context.Context, userInputs []models_service.UserInput) (models_service.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{deploymentJobSlotNum, moduleJobSlotNum})
	if len(currentJobs) > 0 {
		return models_service.Job{}, errors.New("active jobs") // TODO
	}
	handlerModules, err := s.modulesHandler.Modules(ctx, models_handler_modules.ModuleFilter{
		Ids: helper_slices.CollectFunc(slices.Values(userInputs), func(item models_service.UserInput) string {
			return item.ModuleId
		}),
		Dependencies: true,
	})
	if err != nil {
		return models_service.Job{}, err
	}
	userInputMap, err := getUserInputs(userInputs, handlerModules)
	if err != nil {
		return models_service.Job{}, err
	}
	job, err := s.jobsHandler.CreateSlotJob(deploymentJobSlotNum, "create deployments")
	if err != nil {
		return models_service.Job{}, err
	}
	go func() {
		defer job.Done()
		jobResult := models_service.DeploymentsResult{JobId: job.Id}
		defer func() {
			if err := recover(); err != nil {
				jobResult.HasError = true
				jobResult.Error = fmt.Sprintf("panic: %v", err)
				s.jobResults.setDeploymentOperationResult(job.Id, jobResult)
			}
		}()
		jobResult.Results, err = s.deploymentsHandler.CreateDeployments(job.Context(), handlerModules, userInputMap)
		if err != nil {
			jobResult.HasError = true
			jobResult.Error = err.Error()
		}
		for _, res := range jobResult.Results {
			if res.HasError {
				jobResult.ResultsErrNum++
			}
		}
		s.jobResults.setDeploymentOperationResult(job.Id, jobResult)
	}()
	return models_service.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) UpdateDeployments(ctx context.Context, userInputs []models_service.UserInput) (models_service.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{deploymentJobSlotNum, moduleJobSlotNum})
	if len(currentJobs) > 0 {
		return models_service.Job{}, errors.New("active jobs") // TODO
	}
	handlerModules, err := s.modulesHandler.Modules(ctx, models_handler_modules.ModuleFilter{
		Ids: helper_slices.CollectFunc(slices.Values(userInputs), func(item models_service.UserInput) string {
			return item.ModuleId
		}),
	})
	if err != nil {
		return models_service.Job{}, err
	}
	userInputMap, err := getUserInputs(userInputs, handlerModules)
	if err != nil {
		return models_service.Job{}, err
	}
	job, err := s.jobsHandler.CreateSlotJob(deploymentJobSlotNum, "update deployments")
	if err != nil {
		return models_service.Job{}, err
	}
	go func() {
		defer job.Done()
		jobResult := models_service.DeploymentsResult{JobId: job.Id}
		defer func() {
			if err := recover(); err != nil {
				jobResult.HasError = true
				jobResult.Error = fmt.Sprintf("panic: %v", err)
				s.jobResults.setDeploymentOperationResult(job.Id, jobResult)
			}
		}()
		jobResult.Results, err = s.deploymentsHandler.UpdateDeployments(job.Context(), handlerModules, userInputMap)
		if err != nil {
			jobResult.HasError = true
			jobResult.Error = err.Error()
		}
		for _, res := range jobResult.Results {
			if res.HasError {
				jobResult.ResultsErrNum++
			}
		}
		s.jobResults.setDeploymentOperationResult(job.Id, jobResult)
	}()
	return models_service.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) RecreateDeployments(ctx context.Context, moduleIds []string) (models_service.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{deploymentJobSlotNum, moduleJobSlotNum})
	if len(currentJobs) > 0 {
		return models_service.Job{}, errors.New("active jobs") // TODO
	}
	handlerModules, err := s.modulesHandler.Modules(ctx, models_handler_modules.ModuleFilter{
		Ids: moduleIds,
	})
	if err != nil {
		return models_service.Job{}, err
	}
	job, err := s.jobsHandler.CreateSlotJob(deploymentJobSlotNum, "recreate deployments")
	if err != nil {
		return models_service.Job{}, err
	}
	go func() {
		defer job.Done()
		jobResult := models_service.DeploymentsResult{JobId: job.Id}
		defer func() {
			if err := recover(); err != nil {
				jobResult.HasError = true
				jobResult.Error = fmt.Sprintf("panic: %v", err)
				s.jobResults.setDeploymentOperationResult(job.Id, jobResult)
			}
		}()
		jobResult.Results, err = s.deploymentsHandler.RecreateDeployments(job.Context(), handlerModules)
		if err != nil {
			jobResult.HasError = true
			jobResult.Error = err.Error()
		}
		for _, res := range jobResult.Results {
			if res.HasError {
				jobResult.ResultsErrNum++
			}
		}
		s.jobResults.setDeploymentOperationResult(job.Id, jobResult)
	}()
	return models_service.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) DeleteDeployments(ctx context.Context, moduleIds []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.jobsHandler.CurrentSlotJob(deploymentJobSlotNum)
	if ok {
		return errors.New("active job") // TODO
	}
	return s.deploymentsHandler.DeleteDeployments(ctx, models_handler_deployments.DeploymentsFilter{
		DeploymentsFilter: models_handler_database.DeploymentsFilter{
			ModuleIds: moduleIds,
		},
	})
}

func (s *Service) EnableDeployments(ctx context.Context, moduleIds []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.jobsHandler.CurrentSlotJob(deploymentJobSlotNum)
	if ok {
		return errors.New("active job") // TODO
	}
	handlerModules, err := s.modulesHandler.Modules(ctx, models_handler_modules.ModuleFilter{
		Ids:          moduleIds,
		Dependencies: true,
	})
	if err != nil {
		return err
	}
	return s.deploymentsHandler.EnableDeployments(ctx, slices.Collect(maps.Keys(handlerModules)))
}

func (s *Service) DisableDeployments(ctx context.Context, moduleIds []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.jobsHandler.CurrentSlotJob(deploymentJobSlotNum)
	if ok {
		return errors.New("active job") // TODO
	}
	return s.deploymentsHandler.DisableDeployments(ctx, moduleIds)
}

func getUserInputs(
	userInputs []models_service.UserInput,
	handlerModules map[string]models_handler_modules.Module,
) (map[string]models_handler_deployments.UserInput, error) {
	userInputsMap := make(map[string]models_handler_deployments.UserInput)
	for _, userInput := range userInputs {
		_, ok := userInputsMap[userInput.ModuleId]
		if ok {
			return nil, errors.New("duplicate module id " + userInput.ModuleId) // TODO
		}
		handlerModule := handlerModules[userInput.ModuleId]
		configs := make(map[string]models_config.Config)
		for reference, value := range userInput.Configs {
			config, err := helper_configs.GetConfig(value, handlerModule.Configs[reference])
			if err != nil {
				return nil, err
			}
			configs[reference] = config
		}
		files := make(map[string][]byte)
		for reference, value := range userInput.Files {
			data, err := base64.StdEncoding.DecodeString(value)
			if err != nil {
				return nil, err
			}
			files[reference] = data
		}
		fileGroups := make(map[string]map[string]models_handler_deployments.FileGroupUserInput)
		for reference, items := range userInput.FileGroups {
			depItems := make(map[string]models_handler_deployments.FileGroupUserInput)
			for path, item := range items {
				data, err := base64.StdEncoding.DecodeString(item.Data)
				if err != nil {
					return nil, err
				}
				depItems[path] = models_handler_deployments.FileGroupUserInput{
					Format: item.Format,
					Data:   data,
				}
			}
			fileGroups[reference] = depItems
		}
		userInputsMap[userInput.ModuleId] = models_handler_deployments.UserInput{
			ModuleId:      userInput.ModuleId,
			HostResources: userInput.HostResources,
			Secrets:       userInput.Secrets,
			Configs:       configs,
			GlobalConfigs: userInput.GlobalConfigs,
			Files:         files,
			FileGroups:    fileGroups,
		}
	}
	return userInputsMap, nil
}
