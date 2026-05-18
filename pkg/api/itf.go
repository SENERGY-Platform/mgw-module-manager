/*
 * Copyright 2025 InfAI (CC SES)
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

package api

import (
	"context"

	srv_info_hdl "github.com/SENERGY-Platform/go-service-base/srv-info-hdl"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

type serviceItf interface {
	GetModules(ctx context.Context, filter lib_models.ModulesFilter) ([]lib_models.ModuleReduced, error)
	GetModule(ctx context.Context, id string) (lib_models.Module, error)
	GetModulesChangeRequest(ctx context.Context) (lib_models.ModulesChangeRequest, error)
	CreateModulesChangeRequest(
		ctx context.Context,
		reqItems []lib_models.ChangeRequestItem,
	) (lib_models.ModulesChangeRequest, error)
	ExecModulesChangeRequest(ctx context.Context) (lib_models.Job, error)
	CancelModulesChangeRequest(ctx context.Context) error
	GetModulesAvailableUpdates(ctx context.Context) (int, error)
	CreateModulesUpdateAllChangeRequest(ctx context.Context) (lib_models.ModulesChangeRequest, error)

	RefreshRepositories(ctx context.Context) (lib_models.Job, error)
	RepoModules(ctx context.Context, filter lib_models.RepoModulesFilter) ([]lib_models.RepoModule, error)

	CreateGlobalConfig(ctx context.Context, input lib_models.GlobalConfigInput) (string, error)
	GetGlobalConfig(ctx context.Context, id string) (lib_models.GlobalConfig, error)
	GetGlobalConfigs(ctx context.Context, ids []string) (map[string]lib_models.GlobalConfig, error)
	UpdateGlobalConfig(ctx context.Context, config lib_models.GlobalConfig) error
	DeleteGlobalConfig(ctx context.Context, id string) error
	DeleteGlobalConfigs(ctx context.Context, ids []string, allowAll bool) error

	GetDeploymentRequest(ctx context.Context, moduleIds []string) ([]lib_models.Module, error)
	CreateDeployments(ctx context.Context, userInputs []lib_models.DeploymentUserInput) (lib_models.Job, error)
	UpdateDeployments(ctx context.Context, userInputs []lib_models.DeploymentUserInput) (lib_models.Job, error)
	RecreateDeployments(ctx context.Context, moduleIds []string) (lib_models.Job, error)
	DeleteDeployments(ctx context.Context, moduleIds []string) ([]lib_models.DeploymentDeleteResult, error)
	EnableDeployments(ctx context.Context, moduleIds []string) ([]string, error)
	DisableDeployments(ctx context.Context, moduleIds []string) ([]string, error)

	QueryDeploymentAdvertisements(
		ctx context.Context,
		filter lib_models.DeploymentAdvertisementsFilter,
	) ([]lib_models.DeploymentAdvertisementReduced, error)
	QueryDeploymentAdvertisement(ctx context.Context, id string) (lib_models.DeploymentAdvertisementReduced, error)
	GetDeploymentAdvertisement(
		ctx context.Context,
		deploymentId string,
		reference string,
	) (lib_models.DeploymentAdvertisement, error)
	GetDeploymentAdvertisementById(
		ctx context.Context,
		id string,
	) (lib_models.DeploymentAdvertisement, error)
	GetDeploymentAdvertisements(
		ctx context.Context,
		deploymentId string,
		filter lib_models.DeploymentAdvertisementsFilterReduced,
	) (map[string]lib_models.DeploymentAdvertisement, error)
	PutDeploymentAdvertisement(
		ctx context.Context,
		deploymentId string,
		reference string,
		items map[string]string,
	) (string, error)
	PutDeploymentAdvertisements(
		ctx context.Context,
		deploymentId string,
		inputs []lib_models.DeploymentAdvertisementInput,
		incremental bool,
	) (map[string]string, error)
	DeleteDeploymentAdvertisements(
		ctx context.Context,
		deploymentId string,
		filter lib_models.DeploymentAdvertisementsFilterReduced,
		allowAll bool,
	) error

	GetAuxiliaryDeployment(
		ctx context.Context,
		deploymentId string,
		auxDeploymentId string,
	) (lib_models.AuxiliaryDeployment, error)
	GetAuxiliaryDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models.AuxiliaryDeploymentsFilterWithState,
	) (map[string]lib_models.AuxiliaryDeployment, error)
	GetReducedAuxiliaryDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models.AuxiliaryDeploymentsFilterWithState,
	) (map[string]lib_models.AuxiliaryDeploymentReduced, error)
	CreateAuxiliaryDeployment(
		ctx context.Context,
		serviceInput lib_models.AuxiliaryDeploymentInput,
	) (lib_models.Job, error)
	UpdateAuxiliaryDeployment(
		ctx context.Context,
		serviceInput lib_models.AuxiliaryDeploymentUpdateInput,
	) (lib_models.Job, error)
	RecreateAuxiliaryDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models.AuxiliaryDeploymentsFilterWithState,
	) (lib_models.Job, error)
	DeleteAuxiliaryDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models.AuxiliaryDeploymentsFilterWithState,
		allowAll bool,
	) (lib_models.Job, error)
	EnableAuxiliaryDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models.AuxiliaryDeploymentsFilterWithState,
	) ([]string, error)
	DisableAuxiliaryDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models.AuxiliaryDeploymentsFilterWithState,
	) ([]string, error)
	GetAuxiliaryDeploymentVolumes(
		ctx context.Context,
		deploymentId string,
		filterReferences []string,
	) (map[string]lib_models.AuxiliaryDeploymentVolume, error)
	GetAuxiliaryDeploymentVolumesWithMounts(
		ctx context.Context,
		deploymentId string,
		filterReferences []string,
	) (map[string]lib_models.AuxiliaryDeploymentVolumeWithMounts, error)
	DeleteAuxiliaryDeploymentVolumes(
		ctx context.Context,
		deploymentId string,
		filterReferences []string,
		allowAll bool,
	) ([]lib_models.AuxiliaryDeploymentVolumeResult, error)
	DeleteUnusedAuxiliaryDeploymentVolumes(
		ctx context.Context,
		deploymentId string,
		excludeReferences []string,
	) ([]lib_models.AuxiliaryDeploymentVolumeResult, error)

	GetDeploymentsJobResult(jobId string) (lib_models.DeploymentJobResult, error)
	GetUpdateDeploymentsJobResult(jobId string) (lib_models.DeploymentUpdateJobResult, error)
	GetModuleChangeJobResult(jobId string) (lib_models.ModulesChangeJobResult, error)
	GetRefreshRepositoriesJobResult(jobId string) (lib_models.JobResult, error)
	GetCreateAuxiliaryDeploymentJobResult(jobId string) (lib_models.AuxiliaryDeploymentCreateJobResult, error)
	GetUpdateAuxiliaryDeploymentJobResult(jobId string) (lib_models.JobResult, error)
	GetAuxiliaryDeploymentsJobResult(jobId string) (lib_models.AuxiliaryDeploymentJobResult, error)

	GetJobs(ctx context.Context, filterIds []string) ([]lib_models.Job, error)
	GetJob(ctx context.Context, id string) (lib_models.Job, error)
	CancelJobs(ctx context.Context, ids []string) error
	CancelJob(ctx context.Context, id string) error

	ServiceHealth(ctx context.Context) error
	DeploymentsHealth(ctx context.Context, filter lib_models.DeploymentsHealthInfoFilter) (lib_models.DeploymentsHealthInfo, error)
}

type infoHandler interface {
	ServiceInfo() srv_info_hdl.ServiceInfo
	Version() string
	Name() string
}
