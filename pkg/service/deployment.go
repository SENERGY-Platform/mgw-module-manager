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

	helper_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/configs"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	models_config "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/config"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_handler_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/aux_deployments"
	models_handler_database "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
	models_handler_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployments"
	models_handler_modules "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/modules"
	models_service "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
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
		jobResult := models_service.JobResultDeployments{
			JobResult: models_service.JobResult{JobId: job.Id},
		}
		defer func() {
			if err := recover(); err != nil {
				jobResult.ErrorResult = models_error.NewErrorResult(fmt.Sprintf("panic: %v", err))
				s.setDeploymentsJobResult(job.Id, jobResult)
			}
		}()
		jobResult.Results, err = s.deploymentsHandler.CreateDeployments(job.Context(), handlerModules, userInputMap)
		if err != nil {
			jobResult.ErrorResult = models_error.NewErrorResult(err.Error())
		}
		for _, res := range jobResult.Results {
			if res.HasError {
				jobResult.ResultsErrNum++
			}
		}
		s.setDeploymentsJobResult(job.Id, jobResult)
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
		jobResult := models_service.JobResultUpdateDeployments{
			JobResult: models_service.JobResult{JobId: job.Id},
		}
		defer func() {
			if err := recover(); err != nil {
				jobResult.ErrorResult = models_error.NewErrorResult(fmt.Sprintf("panic: %v", err))
				s.setUpdateDeploymentsJobResult(job.Id, jobResult)
			}
		}()
		updateDepResults, err := s.deploymentsHandler.UpdateDeployments(job.Context(), handlerModules, userInputMap)
		if err != nil {
			jobResult.ErrorResult = models_error.NewErrorResult(err.Error())
		}
		for _, updateDepResult := range updateDepResults {
			if updateDepResult.HasError {
				jobResult.ResultsErrNum++
			}
		}
		cacheDependencyDeployments := make(map[string]models_handler_deployments.DeploymentReduced)
		for _, updateDepResult := range updateDepResults {
			result := models_service.JobResultUpdateDeploymentsResult{Result: updateDepResult}
			if !updateDepResult.HasError {
				module, ok := handlerModules[updateDepResult.ModuleId]
				if ok {
					result.AuxiliaryDeployments.Results, err = s.recreateAuxDeployments(
						ctx,
						module,
						updateDepResult.Id,
						cacheDependencyDeployments,
					)
					if err != nil {
						result.AuxiliaryDeployments.ErrorResult = models_error.NewErrorResult(err.Error())
					}
					for _, res := range result.AuxiliaryDeployments.Results {
						if res.HasError {
							result.AuxiliaryDeployments.ResultsErrNum++
						}
					}
				} else {
					result.AuxiliaryDeployments.ErrorResult = models_error.NewErrorResult("missing module")
				}
			}
			jobResult.Results = append(jobResult.Results, result)
		}
		s.setUpdateDeploymentsJobResult(job.Id, jobResult)
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
		jobResult := models_service.JobResultDeployments{
			JobResult: models_service.JobResult{JobId: job.Id},
		}
		defer func() {
			if err := recover(); err != nil {
				jobResult.ErrorResult = models_error.NewErrorResult(fmt.Sprintf("panic: %v", err))
				s.setDeploymentsJobResult(job.Id, jobResult)
			}
		}()
		jobResult.Results, err = s.deploymentsHandler.RecreateDeployments(job.Context(), handlerModules)
		if err != nil {
			jobResult.ErrorResult = models_error.NewErrorResult(err.Error())
		}
		for _, res := range jobResult.Results {
			if res.HasError {
				jobResult.ResultsErrNum++
			}
		}
		s.setDeploymentsJobResult(job.Id, jobResult)
	}()
	return models_service.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) DeleteDeployments(ctx context.Context, moduleIds []string) ([]models_service.DeleteDeploymentsResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.jobsHandler.CurrentSlotJob(deploymentJobSlotNum)
	if ok {
		return nil, errors.New("active job") // TODO
	}
	deploymentIds, err := s.deploymentsHandler.GetDeploymentIds(ctx, models_handler_database.DeploymentsFilter{
		ModuleIds: moduleIds,
	})
	if err != nil {
		return nil, err
	}
	auxResults := make(map[string]models_service.DeleteAuxiliaryDeploymentResult)
	var toDelete []string
	for id := range deploymentIds {
		var auxResult models_service.DeleteAuxiliaryDeploymentResult
		auxResult.Results, auxResult.VolumeResults, err = s.deleteAuxDeployments(ctx, id)
		if err != nil {
			auxResult.ErrorResult = models_error.NewErrorResult(err.Error())
		}
		for _, res := range auxResult.Results {
			if res.HasError {
				auxResult.ResultsErrNum++
			}
		}
		for _, res := range auxResult.VolumeResults {
			if res.HasError {
				auxResult.VolumeResultsErrNum++
			}
		}
		auxResults[id] = auxResult
		if !auxResult.HasError && auxResult.ResultsErrNum+auxResult.VolumeResultsErrNum == 0 {
			toDelete = append(toDelete, id)
		}
	}
	deleteResults, err := s.deploymentsHandler.DeleteDeployments(
		ctx,
		models_handler_deployments.DeploymentsFilter{
			DeploymentsFilter: models_handler_database.DeploymentsFilter{
				Ids: toDelete,
			},
		},
		false,
	)
	deleteResultsMap := maps.Collect(helper_slices.AllFunc(deleteResults, func(item models_handler_deployments.Result) string {
		return item.Id
	}))
	var results []models_service.DeleteDeploymentsResult
	for id, moduleId := range deploymentIds {
		var errResult models_error.ErrorResult
		deleteResult, ok := deleteResultsMap[id]
		if !ok {
			errResult = models_error.NewErrorResult("not deleted")
		} else {
			errResult = deleteResult.ErrorResult
		}
		results = append(results, models_service.DeleteDeploymentsResult{
			Result: models_handler_deployments.Result{
				ModuleId:    moduleId,
				Id:          id,
				ErrorResult: errResult,
			},
			AuxiliaryDeployments: auxResults[id],
		})
	}
	if err != nil {
		return results, err
	}
	return results, nil
}

func (s *Service) EnableDeployments(ctx context.Context, moduleIds []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.jobsHandler.CurrentSlotJob(deploymentJobSlotNum)
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
	return s.deploymentsHandler.EnableDeployments(ctx, slices.Collect(maps.Keys(handlerModules)))
}

func (s *Service) DisableDeployments(ctx context.Context, moduleIds []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.jobsHandler.CurrentSlotJob(deploymentJobSlotNum)
	if ok {
		return nil, errors.New("active job") // TODO
	}
	return s.deploymentsHandler.DisableDeployments(ctx, moduleIds)
}

func (s *Service) deleteAuxDeployments(
	ctx context.Context,
	deploymentId string,
) ([]models_handler_aux_deployments.BatchResult, []models_handler_aux_deployments.VolumeResult, error) {
	results, err := s.auxDeploymentsHandler.DeleteDeployments(
		ctx,
		deploymentId,
		models_handler_aux_deployments.AuxiliaryDeploymentsFilter{},
		true,
	)
	if err != nil {
		return results, nil, err
	}
	volResults, err := s.auxDeploymentsHandler.DeleteVolumes(ctx, deploymentId, nil, true)
	if err != nil {
		return results, volResults, err
	}
	return results, volResults, nil
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
		configs := make(map[string]models_config.Value)
		for reference, itfValue := range userInput.Configs {
			value, err := helper_configs.GetValueWithValidation(itfValue, handlerModule.Configs[reference])
			if err != nil {
				return nil, err
			}
			configs[reference] = value
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
