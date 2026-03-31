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

package handler_deployments

import (
	"context"

	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

type storageHandler interface {
	CreateDeployment(
		ctx context.Context,
		deployment models_handler_storage.Deployment,
		hostResources []models_handler_storage.DeploymentHostResource,
		secrets []models_handler_storage.DeploymentSecret,
		userConfigs []models_handler_storage.DeploymentUserConfig,
		globalConfigs []models_handler_storage.DeploymentGlobalConfig,
		files []models_handler_storage.DeploymentFile,
		fileGroups []models_handler_storage.DeploymentFileGroup,
		volumes []models_handler_storage.DeploymentVolume,
		containers []models_handler_storage.DeploymentContainer,
	) error
	ReadDeployment(ctx context.Context, id string) (models_handler_storage.Deployment, error)
	ReadDeployments(
		ctx context.Context,
		filter models_handler_storage.DeploymentsFilter,
	) (map[string]models_handler_storage.Deployment, error)
	ReadDeploymentsContainers(
		ctx context.Context,
		deploymentIds []string,
	) (map[string]map[string]models_handler_storage.DeploymentContainer, error)
	ReadDeploymentsVolumes(
		ctx context.Context,
		deploymentIds []string,
	) (map[string]map[string]models_handler_storage.DeploymentVolume, error)
	ReadDeploymentsHostResources(
		ctx context.Context,
		filter models_handler_storage.DeploymentsHostResourcesFilter,
	) (map[string]map[string]models_handler_storage.DeploymentHostResource, error)
	ReadDeploymentsSecrets(
		ctx context.Context,
		filter models_handler_storage.DeploymentsSecretsFilter,
	) (map[string]map[string]models_handler_storage.DeploymentSecret, error)
	ReadDeploymentsConfigs(
		ctx context.Context,
		deploymentIds []string,
	) (map[string]map[string]models_handler_storage.DeploymentUserConfig, error)
	ReadDeploymentsGlobalConfigs(
		ctx context.Context,
		filter models_handler_storage.DeploymentGlobalConfigsFilter,
	) (map[string]map[string]models_handler_storage.DeploymentGlobalConfig, error)
	ReadDeploymentsFiles(
		ctx context.Context,
		deploymentIds []string,
	) (map[string]map[string]models_handler_storage.DeploymentFile, error)
	ReadDeploymentsFileGroups(
		ctx context.Context,
		deploymentIds []string,
	) (map[string]map[string]models_handler_storage.DeploymentFileGroup, error)
	ReadGlobalConfigs(ctx context.Context, ids []string) (map[string]models_handler_storage.GlobalConfig, error)
	UpdateDeploymentsEnabledState(ctx context.Context, deploymentIds []string, state bool) error
	UpdateDeploymentContainerNames(ctx context.Context, containers []models_handler_storage.DeploymentContainer) error
	UpdateDeployment(
		ctx context.Context,
		deployment models_handler_storage.Deployment,
		hostResources []models_handler_storage.DeploymentHostResource,
		secrets []models_handler_storage.DeploymentSecret,
		userConfigs []models_handler_storage.DeploymentUserConfig,
		globalConfigs []models_handler_storage.DeploymentGlobalConfig,
		files []models_handler_storage.DeploymentFile,
		fileGroups []models_handler_storage.DeploymentFileGroup,
		volumes []models_handler_storage.DeploymentVolume,
		containers []models_handler_storage.DeploymentContainer,
	) (err error)
	DeleteDeployment(ctx context.Context, id string) error
}

type containerEngineWrapperClient interface {
	GetContainers(ctx context.Context, filter models_external.ContainersFilter) ([]models_external.Container, error)
	CreateContainer(ctx context.Context, container models_external.Container) (id string, err error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string) (jobId string, err error)
	RestartContainer(ctx context.Context, id string) (jobId string, err error)
	RemoveContainer(ctx context.Context, id string, force bool) error
	GetImage(ctx context.Context, id string) (models_external.Image, error)
	AddImage(ctx context.Context, img string) (jobId string, err error)
	GetVolumes(ctx context.Context, filter models_external.VolumesFilter) ([]models_external.Volume, error)
	CreateVolume(ctx context.Context, vol models_external.Volume) (string, error)
	RemoveVolume(ctx context.Context, id string, force bool) error
	GetJob(ctx context.Context, id string) (models_external.Job, error)
	CancelJob(ctx context.Context, id string) error
}

type hostManagerClient interface {
	GetHostResource(ctx context.Context, id string) (models_external.HostResource, error)
}

type secretManagerClient interface {
	InitPathVariant(ctx context.Context, secretRequest models_external.SecretVariantRequest) (variant models_external.SecretPathVariant, err error, errCode int)
	LoadPathVariant(ctx context.Context, secretRequest models_external.SecretVariantRequest) (err error, errCode int)
	UnloadPathVariant(ctx context.Context, secretRequest models_external.SecretVariantRequest) (err error, errCode int)
	CleanPathVariants(ctx context.Context, ref string) (err error, errCode int)
	GetValueVariant(ctx context.Context, secretRequest models_external.SecretVariantRequest) (variant models_external.SecretValueVariant, err error, errCode int)
}

type coreManagerClient interface {
	SetEndpoints(ctx context.Context, endpoints []models_external.CmEndpointBase) (string, error)
	RemoveEndpoints(ctx context.Context, filter models_external.CmEndpointFiler, restrictStd bool) (string, error)
	GetJob(ctx context.Context, id string) (models_external.Job, error)
	CancelJob(ctx context.Context, id string) error
}
