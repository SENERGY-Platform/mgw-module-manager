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
	"sync"

	lib_service "github.com/SENERGY-Platform/mgw-module-manager/lib/models/service"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
)

type jobResults struct {
	deployments         map[string]lib_service.JobResultDeployments
	deploymentsUpdate   map[string]lib_service.JobResultUpdateDeployments
	moduleChange        map[string]lib_service.JobResultModulesChange
	refreshRepositories map[string]lib_service.JobResult
	auxDeploymentCreate map[string]lib_service.JobResultCreateAuxiliaryDeployment
	auxDeploymentUpdate map[string]lib_service.JobResult
	auxDeployment       map[string]lib_service.JobResultAuxiliaryDeployments
	mu                  sync.RWMutex
}

func (s *Service) setDeploymentsJobResult(jobId string, res lib_service.JobResultDeployments) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.deployments[jobId] = res
}

func (s *Service) GetDeploymentsJobResult(jobId string) (lib_service.JobResultDeployments, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.deployments[jobId]
	if !ok {
		return lib_service.JobResultDeployments{}, models_error.NotFoundErr
	}
	return res, nil
}

func (s *Service) setUpdateDeploymentsJobResult(jobId string, res lib_service.JobResultUpdateDeployments) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.deploymentsUpdate[jobId] = res
}

func (s *Service) GetUpdateDeploymentsJobResult(jobId string) (lib_service.JobResultUpdateDeployments, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.deploymentsUpdate[jobId]
	if !ok {
		return lib_service.JobResultUpdateDeployments{}, models_error.NotFoundErr
	}
	return res, nil
}

func (s *Service) setModuleChangeJobResult(jobId string, res lib_service.JobResultModulesChange) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.moduleChange[jobId] = res
}

func (s *Service) GetModuleChangeJobResult(jobId string) (lib_service.JobResultModulesChange, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.moduleChange[jobId]
	if !ok {
		return lib_service.JobResultModulesChange{}, models_error.NotFoundErr
	}
	return res, nil
}

func (s *Service) setRefreshRepositoriesJobResult(jobId string, res lib_service.JobResult) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.refreshRepositories[jobId] = res
}

func (s *Service) GetRefreshRepositoriesJobResult(jobId string) (lib_service.JobResult, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.refreshRepositories[jobId]
	if !ok {
		return lib_service.JobResult{}, models_error.NotFoundErr
	}
	return res, nil
}

func (s *Service) setCreateAuxiliaryDeploymentJobResult(jobId string, res lib_service.JobResultCreateAuxiliaryDeployment) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.auxDeploymentCreate[jobId] = res
}

func (s *Service) GetCreateAuxiliaryDeploymentJobResult(jobId string) (lib_service.JobResultCreateAuxiliaryDeployment, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.auxDeploymentCreate[jobId]
	if !ok {
		return lib_service.JobResultCreateAuxiliaryDeployment{}, models_error.NotFoundErr
	}
	return res, nil
}

func (s *Service) setUpdateAuxiliaryDeploymentJobResult(jobId string, res lib_service.JobResult) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.auxDeploymentUpdate[jobId] = res
}

func (s *Service) GetUpdateAuxiliaryDeploymentJobResult(jobId string) (lib_service.JobResult, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.auxDeploymentUpdate[jobId]
	if !ok {
		return lib_service.JobResult{}, models_error.NotFoundErr
	}
	return res, nil
}

func (s *Service) setAuxiliaryDeploymentsJobResult(jobId string, res lib_service.JobResultAuxiliaryDeployments) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.auxDeployment[jobId] = res
}

func (s *Service) GetAuxiliaryDeploymentsJobResult(jobId string) (lib_service.JobResultAuxiliaryDeployments, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.auxDeployment[jobId]
	if !ok {
		return lib_service.JobResultAuxiliaryDeployments{}, models_error.NotFoundErr
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
