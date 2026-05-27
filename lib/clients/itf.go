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

package clients

import (
	"context"

	"github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

type ClientAuxiliaryDeploymentsItf interface {
	CreateAuxiliaryDeployment(
		ctx context.Context,
		deploymentId string,
		serviceInput models.AuxiliaryDeploymentInput,
	) (models.Job, error)
	UpdateAuxiliaryDeployment(
		ctx context.Context,
		deploymentId string,
		auxDeploymentId string,
		serviceInput models.AuxiliaryDeploymentInput,
		incremental bool,
	) (models.Job, error)
	RecreateAuxiliaryDeployments(
		ctx context.Context,
		deploymentId string,
		filter models.AuxiliaryDeploymentsFilterWithState,
	) (models.Job, error)
	DeleteAuxiliaryDeployment(ctx context.Context, deploymentId, auxDeploymentId string) error
	DeleteAuxiliaryDeployments(
		ctx context.Context,
		deploymentId string,
		filter models.AuxiliaryDeploymentsFilterWithState,
		allowAll bool,
	) ([]models.AuxiliaryDeploymentBatchResult, error)
	EnableAuxiliaryDeployments(
		ctx context.Context,
		deploymentId string,
		filter models.AuxiliaryDeploymentsFilterWithState,
	) ([]string, error)
	DisableAuxiliaryDeployments(
		ctx context.Context,
		deploymentId string,
		filter models.AuxiliaryDeploymentsFilterWithState,
	) ([]string, error)
	DeleteAuxiliaryDeploymentVolume(ctx context.Context, deploymentId, reference string) error
	DeleteAuxiliaryDeploymentVolumes(
		ctx context.Context,
		deploymentId string,
		filterReferences []string,
		allowAll bool,
	) ([]models.AuxiliaryDeploymentVolumeResult, error)
	DeleteUnusedAuxiliaryDeploymentVolumes(
		ctx context.Context,
		deploymentId string,
		excludeReferences []string,
	) ([]models.AuxiliaryDeploymentVolumeResult, error)
	GetAuxiliaryDeployment(
		ctx context.Context,
		deploymentId string,
		auxDeploymentId string,
	) (models.AuxiliaryDeployment, error)
	GetAuxiliaryDeployments(
		ctx context.Context,
		deploymentId string,
		filter models.AuxiliaryDeploymentsFilterWithState,
	) (map[string]models.AuxiliaryDeployment, error)
	GetReducedAuxiliaryDeployments(
		ctx context.Context,
		deploymentId string,
		filter models.AuxiliaryDeploymentsFilterWithState,
	) (map[string]models.AuxiliaryDeploymentReduced, error)
	GetAuxiliaryDeploymentVolumes(
		ctx context.Context,
		deploymentId string,
		filterReferences []string,
	) (map[string]models.AuxiliaryDeploymentVolume, error)
	GetAuxiliaryDeploymentVolumesWithMounts(
		ctx context.Context,
		deploymentId string,
		filterReferences []string,
	) (map[string]models.AuxiliaryDeploymentVolumeWithMounts, error)

	GetCreateAuxiliaryDeploymentJobResult(ctx context.Context, jobId string) (models.AuxiliaryDeploymentCreateJobResult, error)
	GetUpdateAuxiliaryDeploymentJobResult(ctx context.Context, jobId string) (models.JobResult, error)
	GetAuxiliaryDeploymentsJobResult(ctx context.Context, jobId string) (models.AuxiliaryDeploymentJobResult, error)

	GetJobs(ctx context.Context, filterIds []string) ([]models.Job, error)
	GetJob(ctx context.Context, id string) (models.Job, error)
	CancelJobs(ctx context.Context, ids []string) error
	CancelJob(ctx context.Context, id string) error
}

type ClientDeploymentAdvertisementsItf interface {
	GetDeploymentAdvertisement(
		ctx context.Context,
		deploymentId string,
		reference string,
	) (models.DeploymentAdvertisement, error)
	GetDeploymentAdvertisementById(
		ctx context.Context,
		deploymentId string,
		id string,
	) (models.DeploymentAdvertisement, error)
	GetDeploymentAdvertisements(
		ctx context.Context,
		deploymentId string,
		filter models.DeploymentAdvertisementsFilterReduced,
	) (map[string]models.DeploymentAdvertisement, error)
	PutDeploymentAdvertisement(
		ctx context.Context,
		deploymentId string,
		reference string,
		items map[string]string,
	) (string, error)
	PutDeploymentAdvertisements(
		ctx context.Context,
		deploymentId string,
		inputs []models.DeploymentAdvertisementInput,
		incremental bool,
	) (map[string]string, error)
	DeleteDeploymentAdvertisement(ctx context.Context, deploymentId string, reference string) error
	DeleteDeploymentAdvertisements(
		ctx context.Context,
		deploymentId string,
		filter models.DeploymentAdvertisementsFilterReduced,
		allowAll bool,
	) error

	QueryDeploymentAdvertisements(
		ctx context.Context,
		filter models.DeploymentAdvertisementsFilter,
	) ([]models.DeploymentAdvertisementReduced, error)
	QueryDeploymentAdvertisement(ctx context.Context, id string) (models.DeploymentAdvertisementReduced, error)
}

type ClientHealthItf interface {
	DeploymentsHealth(ctx context.Context, filter models.DeploymentsHealthInfoFilter) (models.DeploymentsHealthInfo, error)
}
