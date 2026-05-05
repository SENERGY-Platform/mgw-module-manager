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

package models_service

import (
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_handler_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/aux_deployments"
)

type ServiceInput struct {
	DeploymentId string `json:"deployment_id"`
	models_handler_aux_deployments.ServiceInput
}

type ServiceInputUpdate struct {
	DeploymentId    string `json:"deployment_id"`
	AuxDeploymentId string `json:"auxiliary_deployment_id"`
	models_handler_aux_deployments.UpdateServiceInput
}

type JobResultCreateAuxiliaryDeployment struct {
	JobResult
	models_handler_aux_deployments.Result
}

type JobResultAuxiliaryDeployments struct {
	JobResult
	Results       []models_handler_aux_deployments.BatchResult `json:"results"`
	ResultsErrNum int                                          `json:"results_err_num"`
}

type RecreateAuxiliaryDeploymentResult struct {
	models_error.ErrorResult
	Results       []models_handler_aux_deployments.BatchResult `json:"results"`
	ResultsErrNum int                                          `json:"results_err_num"`
}

type DeleteAuxiliaryDeploymentResult struct {
	models_error.ErrorResult
	Results             []models_handler_aux_deployments.BatchResult  `json:"results"`
	ResultsErrNum       int                                           `json:"results_err_num"`
	VolumeResults       []models_handler_aux_deployments.VolumeResult `json:"volume_results"`
	VolumeResultsErrNum int                                           `json:"volume_results_err_num"`
}
