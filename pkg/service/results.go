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

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
)

type jobResults struct {
	deploymentOperationResults map[string]models_service.JobResultDeployments
	moduleChangeResults        map[string]models_service.JobResultModulesChange
	refreshRepositoriesResults map[string]models_service.JobResult
	mu                         sync.RWMutex
}

func (r *jobResults) setDeploymentsResult(jobId string, res models_service.JobResultDeployments) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deploymentOperationResults[jobId] = res
}

func (r *jobResults) GetDeploymentsResult(jobId string) (models_service.JobResultDeployments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res, ok := r.deploymentOperationResults[jobId]
	if !ok {
		return models_service.JobResultDeployments{}, models_error.NotFoundErr
	}
	return res, nil
}

func (r *jobResults) setModuleChangeResult(jobId string, res models_service.JobResultModulesChange) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.moduleChangeResults[jobId] = res
}

func (r *jobResults) GetModuleChangeResult(jobId string) (models_service.JobResultModulesChange, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res, ok := r.moduleChangeResults[jobId]
	if !ok {
		return models_service.JobResultModulesChange{}, models_error.NotFoundErr
	}
	return res, nil
}

func (r *jobResults) setRefreshRepositoriesResult(jobId string, res models_service.JobResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refreshRepositoriesResults[jobId] = res
}

func (r *jobResults) GetRefreshRepositoriesResult(jobId string) (models_service.JobResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res, ok := r.refreshRepositoriesResults[jobId]
	if !ok {
		return models_service.JobResult{}, models_error.NotFoundErr
	}
	return res, nil
}
