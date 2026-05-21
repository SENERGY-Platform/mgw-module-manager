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
	"sync"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

type jobResults struct {
	deployments         map[string]lib_models.DeploymentJobResult
	deploymentsUpdate   map[string]lib_models.DeploymentUpdateJobResult
	moduleChange        map[string]lib_models.ModulesChangeJobResult
	refreshRepositories map[string]lib_models.JobResult
	auxDeploymentCreate map[string]lib_models.AuxiliaryDeploymentCreateJobResult
	auxDeploymentUpdate map[string]lib_models.JobResult
	auxDeployment       map[string]lib_models.AuxiliaryDeploymentJobResult
	mu                  sync.RWMutex
}

func (s *Service) setDeploymentsJobResult(jobId string, res lib_models.DeploymentJobResult) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.deployments[jobId] = res
}

func (s *Service) GetDeploymentsJobResult(_ context.Context, jobId string) (lib_models.DeploymentJobResult, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.deployments[jobId]
	if !ok {
		return lib_models.DeploymentJobResult{}, lib_errors.New[lib_errors.ErrNotFound]("")
	}
	return res, nil
}

func (s *Service) setUpdateDeploymentsJobResult(jobId string, res lib_models.DeploymentUpdateJobResult) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.deploymentsUpdate[jobId] = res
}

func (s *Service) GetUpdateDeploymentsJobResult(_ context.Context, jobId string) (lib_models.DeploymentUpdateJobResult, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.deploymentsUpdate[jobId]
	if !ok {
		return lib_models.DeploymentUpdateJobResult{}, lib_errors.New[lib_errors.ErrNotFound]("")
	}
	return res, nil
}

func (s *Service) setModuleChangeJobResult(jobId string, res lib_models.ModulesChangeJobResult) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.moduleChange[jobId] = res
}

func (s *Service) GetModuleChangeJobResult(_ context.Context, jobId string) (lib_models.ModulesChangeJobResult, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.moduleChange[jobId]
	if !ok {
		return lib_models.ModulesChangeJobResult{}, lib_errors.New[lib_errors.ErrNotFound]("")
	}
	return res, nil
}

func (s *Service) setRefreshRepositoriesJobResult(jobId string, res lib_models.JobResult) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.refreshRepositories[jobId] = res
}

func (s *Service) GetRefreshRepositoriesJobResult(_ context.Context, jobId string) (lib_models.JobResult, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.refreshRepositories[jobId]
	if !ok {
		return lib_models.JobResult{}, lib_errors.New[lib_errors.ErrNotFound]("")
	}
	return res, nil
}

func (s *Service) setCreateAuxiliaryDeploymentJobResult(jobId string, res lib_models.AuxiliaryDeploymentCreateJobResult) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.auxDeploymentCreate[jobId] = res
}

func (s *Service) GetCreateAuxiliaryDeploymentJobResult(_ context.Context, jobId string) (lib_models.AuxiliaryDeploymentCreateJobResult, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.auxDeploymentCreate[jobId]
	if !ok {
		return lib_models.AuxiliaryDeploymentCreateJobResult{}, lib_errors.New[lib_errors.ErrNotFound]("")
	}
	return res, nil
}

func (s *Service) setUpdateAuxiliaryDeploymentJobResult(jobId string, res lib_models.JobResult) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.auxDeploymentUpdate[jobId] = res
}

func (s *Service) GetUpdateAuxiliaryDeploymentJobResult(_ context.Context, jobId string) (lib_models.JobResult, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.auxDeploymentUpdate[jobId]
	if !ok {
		return lib_models.JobResult{}, lib_errors.New[lib_errors.ErrNotFound]("")
	}
	return res, nil
}

func (s *Service) setAuxiliaryDeploymentsJobResult(jobId string, res lib_models.AuxiliaryDeploymentJobResult) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.auxDeployment[jobId] = res
}

func (s *Service) GetAuxiliaryDeploymentsJobResult(_ context.Context, jobId string) (lib_models.AuxiliaryDeploymentJobResult, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.auxDeployment[jobId]
	if !ok {
		return lib_models.AuxiliaryDeploymentJobResult{}, lib_errors.New[lib_errors.ErrNotFound]("")
	}
	return res, nil
}

func (s *Service) DeleteJobResults(jobIds []string) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	for _, id := range jobIds {
		delete(s.jobResults.deployments, id)
		delete(s.jobResults.deploymentsUpdate, id)
		delete(s.jobResults.moduleChange, id)
		delete(s.jobResults.refreshRepositories, id)
		delete(s.jobResults.auxDeploymentCreate, id)
		delete(s.jobResults.auxDeploymentUpdate, id)
		delete(s.jobResults.auxDeployment, id)
	}
}
