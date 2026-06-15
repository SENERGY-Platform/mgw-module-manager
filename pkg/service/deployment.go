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
	"fmt"
	"maps"
	"slices"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/configs"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
)

func (s *Service) GetDeploymentRequest(ctx context.Context, moduleIds []string) ([]lib_models.Module, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	currentJob, ok := s.jobsHandler.CurrentSlotJob(moduleJobSlotNum)
	if ok {
		return nil, lib_errors.New[lib_errors.ErrActiveJob](activeJobErrMsg(currentJob))
	}
	if len(moduleIds) == 0 {
		return nil, nil
	}
	handlerModules, err := s.modulesHandler.GetModules(
		ctx,
		pkg_models.ModulesFilterWithName{
			ModulesFilter: pkg_models.ModulesFilter{
				Ids: moduleIds,
			},
		},
		true,
	)
	if err != nil {
		return nil, err
	}
	handlerDeployments, err := s.deploymentsHandler.GetDeploymentsByModuleIds(ctx, pkg_models.DeploymentsFilterWithState{
		DeploymentsFilter: pkg_models.DeploymentsFilter{
			ModuleIds: slices.Collect(maps.Keys(handlerModules)),
		},
	})
	var modules []lib_models.Module
	for id, handlerModule := range handlerModules {
		_, ok := handlerDeployments[id]
		if !ok {
			modules = append(modules, getModule(handlerModule, pkg_models.Deployment{}))
		}
	}
	return modules, nil
}

func (s *Service) CreateDeployments(ctx context.Context, userInputs []lib_models.DeploymentUserInput) (lib_models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{deploymentJobSlotNum, moduleJobSlotNum})
	if len(currentJobs) > 0 {
		return lib_models.Job{}, lib_errors.New[lib_errors.ErrActiveJob](activeJobsErrMsg(currentJobs))
	}
	if len(userInputs) == 0 {
		return lib_models.Job{}, lib_errors.New[lib_errors.ErrInvalidInput]("no input provided")
	}
	handlerModules, err := s.modulesHandler.GetModules(
		ctx,
		pkg_models.ModulesFilterWithName{
			ModulesFilter: pkg_models.ModulesFilter{
				Ids: helper_slices.CollectFunc(slices.Values(userInputs), func(item lib_models.DeploymentUserInput) string {
					return item.ModuleId
				}),
			},
		},
		true,
	)
	if err != nil {
		return lib_models.Job{}, err
	}
	userInputMap, err := getUserInputs(userInputs, handlerModules)
	if err != nil {
		return lib_models.Job{}, err
	}
	job, err := s.jobsHandler.CreateSlotJob(deploymentJobSlotNum, "create deployments")
	if err != nil {
		return lib_models.Job{}, err
	}
	go func() {
		defer func() {
			job.Done()
			logJobDone(ctx, job)
		}()
		logJobStart(ctx, job)
		jobResult := lib_models.DeploymentJobResult{
			JobResult: lib_models.JobResult{JobId: job.Id},
		}
		defer func() {
			if st := recover(); st != nil {
				jobResult.ErrorResult = lib_models.NewErrorResult(fmt.Sprintf("%v", st))
				s.setDeploymentsJobResult(job.Id, jobResult)
				logger.ErrorContext(
					ctx,
					"create deployments",
					slog_keys.JobId, job.Id,
					slog_keys.Error, "panic",
					slog_keys.StackTrace, st,
				)
			}
		}()
		jobResult.Results, err = s.deploymentsHandler.CreateDeployments(job.Context(), handlerModules, userInputMap)
		if err != nil {
			jobResult.ErrorResult = lib_models.NewErrorResult(err.Error())
		}
		for _, res := range jobResult.Results {
			if res.HasError {
				jobResult.ResultsErrNum++
			}
		}
		s.setDeploymentsJobResult(job.Id, jobResult)
	}()
	return lib_models.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) UpdateDeployments(ctx context.Context, userInputs []lib_models.DeploymentUserInput) (lib_models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{deploymentJobSlotNum, moduleJobSlotNum})
	if len(currentJobs) > 0 {
		return lib_models.Job{}, lib_errors.New[lib_errors.ErrActiveJob](activeJobsErrMsg(currentJobs))
	}
	handlerModules, err := s.modulesHandler.GetModules(
		ctx,
		pkg_models.ModulesFilterWithName{
			ModulesFilter: pkg_models.ModulesFilter{
				Ids: helper_slices.CollectFunc(slices.Values(userInputs), func(item lib_models.DeploymentUserInput) string {
					return item.ModuleId
				}),
			},
		},
		false,
	)
	if err != nil {
		return lib_models.Job{}, err
	}
	userInputMap, err := getUserInputs(userInputs, handlerModules)
	if err != nil {
		return lib_models.Job{}, err
	}
	job, err := s.jobsHandler.CreateSlotJob(deploymentJobSlotNum, "update deployments")
	if err != nil {
		return lib_models.Job{}, err
	}
	go func() {
		defer func() {
			job.Done()
			logJobDone(ctx, job)
		}()
		logJobStart(ctx, job)
		jobResult := lib_models.DeploymentUpdateJobResult{
			JobResult: lib_models.JobResult{JobId: job.Id},
		}
		defer func() {
			if st := recover(); st != nil {
				jobResult.ErrorResult = lib_models.NewErrorResult(fmt.Sprintf("%v", st))
				s.setUpdateDeploymentsJobResult(job.Id, jobResult)
				logger.ErrorContext(
					ctx,
					"update deployments",
					slog_keys.JobId, job.Id,
					slog_keys.Error, "panic",
					slog_keys.StackTrace, st,
				)
			}
		}()
		updateDepResults, err := s.deploymentsHandler.UpdateDeployments(job.Context(), handlerModules, userInputMap)
		if err != nil {
			jobResult.ErrorResult = lib_models.NewErrorResult(err.Error())
		}
		for _, updateDepResult := range updateDepResults {
			if updateDepResult.HasError {
				jobResult.ResultsErrNum++
			}
		}
		cacheDependencyDeployments := make(map[string]pkg_models.DeploymentReduced)
		for _, updateDepResult := range updateDepResults {
			result := lib_models.DeploymentUpdateResult{DeploymentResult: updateDepResult}
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
						result.AuxiliaryDeployments.ErrorResult = lib_models.NewErrorResult(err.Error())
					}
					for _, res := range result.AuxiliaryDeployments.Results {
						if res.HasError {
							result.AuxiliaryDeployments.ResultsErrNum++
						}
					}
				} else {
					result.AuxiliaryDeployments.ErrorResult = lib_models.NewErrorResult("missing module")
				}
			}
			jobResult.Results = append(jobResult.Results, result)
		}
		s.setUpdateDeploymentsJobResult(job.Id, jobResult)
	}()
	return lib_models.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) RecreateDeployments(ctx context.Context, moduleIds []string) (lib_models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{deploymentJobSlotNum, moduleJobSlotNum})
	if len(currentJobs) > 0 {
		return lib_models.Job{}, lib_errors.New[lib_errors.ErrActiveJob](activeJobsErrMsg(currentJobs))
	}
	handlerModules, err := s.modulesHandler.GetModules(
		ctx,
		pkg_models.ModulesFilterWithName{
			ModulesFilter: pkg_models.ModulesFilter{
				Ids: moduleIds,
			},
		},
		false,
	)
	if err != nil {
		return lib_models.Job{}, err
	}
	job, err := s.jobsHandler.CreateSlotJob(deploymentJobSlotNum, "recreate deployments")
	if err != nil {
		return lib_models.Job{}, err
	}
	go func() {
		defer func() {
			job.Done()
			logJobDone(ctx, job)
		}()
		logJobStart(ctx, job)
		jobResult := lib_models.DeploymentJobResult{
			JobResult: lib_models.JobResult{JobId: job.Id},
		}
		defer func() {
			if st := recover(); st != nil {
				jobResult.ErrorResult = lib_models.NewErrorResult(fmt.Sprintf("%v", st))
				s.setDeploymentsJobResult(job.Id, jobResult)
				logger.ErrorContext(
					ctx,
					"recreate deployments",
					slog_keys.JobId, job.Id,
					slog_keys.Error, "panic",
					slog_keys.StackTrace, st,
				)
			}
		}()
		jobResult.Results, err = s.deploymentsHandler.RecreateDeployments(job.Context(), handlerModules)
		if err != nil {
			jobResult.ErrorResult = lib_models.NewErrorResult(err.Error())
		}
		for _, res := range jobResult.Results {
			if res.HasError {
				jobResult.ResultsErrNum++
			}
		}
		s.setDeploymentsJobResult(job.Id, jobResult)
	}()
	return lib_models.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) DeleteDeployments(ctx context.Context, moduleIds []string, allowAll bool) (lib_models.Job, error) {
	if !allowAll && len(moduleIds) == 0 {
		return lib_models.Job{}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJob, ok := s.jobsHandler.CurrentSlotJob(deploymentJobSlotNum)
	if ok {
		return lib_models.Job{}, lib_errors.New[lib_errors.ErrActiveJob](activeJobErrMsg(currentJob))
	}
	if allowAll {
		logger.WarnContext(ctx, "delete deployments", slog_keys.Filter, moduleIds, slog_keys.AllowAll, allowAll)
	}
	deploymentIds, err := s.deploymentsHandler.GetDeploymentIds(ctx, pkg_models.DeploymentsFilter{
		ModuleIds: moduleIds,
	})
	if err != nil {
		return lib_models.Job{}, err
	}
	job, err := s.jobsHandler.CreateSlotJob(deploymentJobSlotNum, "delete deployments")
	if err != nil {
		return lib_models.Job{}, err
	}
	go func() {
		defer func() {
			job.Done()
			logJobDone(ctx, job)
		}()
		logJobStart(ctx, job)
		jobResult := lib_models.DeploymentDeleteJobResult{
			JobResult: lib_models.JobResult{JobId: job.Id},
		}
		defer func() {
			if st := recover(); st != nil {
				jobResult.ErrorResult = lib_models.NewErrorResult(fmt.Sprintf("%v", st))
				s.setDeleteDeploymentsJobResult(job.Id, jobResult)
				logger.ErrorContext(
					ctx,
					"delete deployments",
					slog_keys.JobId, job.Id,
					slog_keys.Error, "panic",
					slog_keys.StackTrace, st,
				)
			}
		}()
		auxResults := make(map[string]lib_models.AuxiliaryDeploymentDeleteResult)
		var toDelete []string
		for id := range deploymentIds {
			var auxResult lib_models.AuxiliaryDeploymentDeleteResult
			auxResult.Results, auxResult.VolumeResults, err = s.deleteAuxDeployments(ctx, id)
			if err != nil {
				auxResult.ErrorResult = lib_models.NewErrorResult(err.Error())
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
			pkg_models.DeploymentsFilterWithState{
				DeploymentsFilter: pkg_models.DeploymentsFilter{
					Ids: toDelete,
				},
			},
			false,
		)
		if err != nil {
			jobResult.ErrorResult = lib_models.NewErrorResult(err.Error())
		}
		deleteResultsMap := maps.Collect(helper_slices.AllFunc(deleteResults, func(item lib_models.DeploymentResult) string {
			return item.Id
		}))
		for id, moduleId := range deploymentIds {
			var errResult lib_models.ErrorResult
			deleteResult, ok := deleteResultsMap[id]
			if !ok {
				errResult = lib_models.NewErrorResult("not deleted")
			} else {
				errResult = deleteResult.ErrorResult
			}
			jobResult.Results = append(jobResult.Results, lib_models.DeploymentDeleteResult{
				DeploymentResult: lib_models.DeploymentResult{
					ModuleId:    moduleId,
					Id:          id,
					ErrorResult: errResult,
				},
				AuxiliaryDeployments: auxResults[id],
			})
		}
		for _, res := range jobResult.Results {
			if res.HasError {
				jobResult.ResultsErrNum++
			}
		}
		s.setDeleteDeploymentsJobResult(job.Id, jobResult)
	}()
	return lib_models.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) EnableDeployments(ctx context.Context, moduleIds []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJob, ok := s.jobsHandler.CurrentSlotJob(deploymentJobSlotNum)
	if ok {
		return nil, lib_errors.New[lib_errors.ErrActiveJob](activeJobErrMsg(currentJob))
	}
	handlerModules, err := s.modulesHandler.GetModules(
		ctx,
		pkg_models.ModulesFilterWithName{
			ModulesFilter: pkg_models.ModulesFilter{
				Ids: moduleIds,
			},
		},
		true,
	)
	if err != nil {
		return nil, err
	}
	return s.deploymentsHandler.EnableDeployments(ctx, slices.Collect(maps.Keys(handlerModules)))
}

func (s *Service) DisableDeployments(ctx context.Context, moduleIds []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJob, ok := s.jobsHandler.CurrentSlotJob(deploymentJobSlotNum)
	if ok {
		return nil, lib_errors.New[lib_errors.ErrActiveJob](activeJobErrMsg(currentJob))
	}
	return s.deploymentsHandler.DisableDeployments(ctx, moduleIds)
}

func (s *Service) deleteAuxDeployments(
	ctx context.Context,
	deploymentId string,
) ([]lib_models.AuxiliaryDeploymentBatchResult, []lib_models.AuxiliaryDeploymentVolumeResult, error) {
	results, err := s.auxDeploymentsHandler.DeleteDeployments(
		ctx,
		deploymentId,
		lib_models.AuxiliaryDeploymentsFilterWithState{},
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
	userInputs []lib_models.DeploymentUserInput,
	handlerModules map[string]pkg_models.Module,
) (map[string]pkg_models.DeploymentUserInput, error) {
	userInputsMap := make(map[string]pkg_models.DeploymentUserInput)
	for _, userInput := range userInputs {
		_, ok := userInputsMap[userInput.ModuleId]
		if ok {
			return nil, lib_errors.New[lib_errors.ErrInvalidInput]("duplicate entry: " + userInput.ModuleId)
		}
		handlerModule := handlerModules[userInput.ModuleId]
		configs := make(map[string]pkg_models.Value)
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
		fileGroups := make(map[string]map[string]pkg_models.DeploymentFileGroupUserInput)
		for reference, items := range userInput.FileGroups {
			depItems := make(map[string]pkg_models.DeploymentFileGroupUserInput)
			for path, item := range items {
				data, err := base64.StdEncoding.DecodeString(item.Data)
				if err != nil {
					return nil, err
				}
				depItems[path] = pkg_models.DeploymentFileGroupUserInput{
					Format: item.Format,
					Data:   data,
				}
			}
			fileGroups[reference] = depItems
		}
		userInputsMap[userInput.ModuleId] = pkg_models.DeploymentUserInput{
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
