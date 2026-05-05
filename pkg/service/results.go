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

	models_service2 "github.com/SENERGY-Platform/mgw-module-manager/lib/models/service"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
)

type jobResults struct {
	deployments         map[string]models_service2.JobResultDeployments
	deploymentsUpdate   map[string]models_service2.JobResultUpdateDeployments
	moduleChange        map[string]models_service2.JobResultModulesChange
	refreshRepositories map[string]models_service2.JobResult
	auxDeploymentCreate map[string]models_service2.JobResultCreateAuxiliaryDeployment
	auxDeploymentUpdate map[string]models_service2.JobResult
	auxDeployment       map[string]models_service2.JobResultAuxiliaryDeployments
	mu                  sync.RWMutex
}

func (s *Service) setDeploymentsJobResult(jobId string, res models_service2.JobResultDeployments) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.deployments[jobId] = res
}

func (s *Service) GetDeploymentsJobResult(jobId string) (models_service2.JobResultDeployments, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.deployments[jobId]
	if !ok {
		return models_service2.JobResultDeployments{}, models_error.NotFoundErr
	}
	return res, nil
}

func (s *Service) setUpdateDeploymentsJobResult(jobId string, res models_service2.JobResultUpdateDeployments) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.deploymentsUpdate[jobId] = res
}

func (s *Service) GetUpdateDeploymentsJobResult(jobId string) (models_service2.JobResultUpdateDeployments, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.deploymentsUpdate[jobId]
	if !ok {
		return models_service2.JobResultUpdateDeployments{}, models_error.NotFoundErr
	}
	return res, nil
}

func (s *Service) setModuleChangeJobResult(jobId string, res models_service2.JobResultModulesChange) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.moduleChange[jobId] = res
}

func (s *Service) GetModuleChangeJobResult(jobId string) (models_service2.JobResultModulesChange, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.moduleChange[jobId]
	if !ok {
		return models_service2.JobResultModulesChange{}, models_error.NotFoundErr
	}
	return res, nil
}

func (s *Service) setRefreshRepositoriesJobResult(jobId string, res models_service2.JobResult) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.refreshRepositories[jobId] = res
}

func (s *Service) GetRefreshRepositoriesJobResult(jobId string) (models_service2.JobResult, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.refreshRepositories[jobId]
	if !ok {
		return models_service2.JobResult{}, models_error.NotFoundErr
	}
	return res, nil
}

func (s *Service) setCreateAuxiliaryDeploymentJobResult(jobId string, res models_service2.JobResultCreateAuxiliaryDeployment) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.auxDeploymentCreate[jobId] = res
}

func (s *Service) GetCreateAuxiliaryDeploymentJobResult(jobId string) (models_service2.JobResultCreateAuxiliaryDeployment, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.auxDeploymentCreate[jobId]
	if !ok {
		return models_service2.JobResultCreateAuxiliaryDeployment{}, models_error.NotFoundErr
	}
	return res, nil
}

func (s *Service) setUpdateAuxiliaryDeploymentJobResult(jobId string, res models_service2.JobResult) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.auxDeploymentUpdate[jobId] = res
}

func (s *Service) GetUpdateAuxiliaryDeploymentJobResult(jobId string) (models_service2.JobResult, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.auxDeploymentUpdate[jobId]
	if !ok {
		return models_service2.JobResult{}, models_error.NotFoundErr
	}
	return res, nil
}

func (s *Service) setAuxiliaryDeploymentsJobResult(jobId string, res models_service2.JobResultAuxiliaryDeployments) {
	s.jobResults.mu.Lock()
	defer s.jobResults.mu.Unlock()
	s.jobResults.auxDeployment[jobId] = res
}

func (s *Service) GetAuxiliaryDeploymentsJobResult(jobId string) (models_service2.JobResultAuxiliaryDeployments, error) {
	s.jobResults.mu.RLock()
	defer s.jobResults.mu.RUnlock()
	res, ok := s.jobResults.auxDeployment[jobId]
	if !ok {
		return models_service2.JobResultAuxiliaryDeployments{}, models_error.NotFoundErr
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
