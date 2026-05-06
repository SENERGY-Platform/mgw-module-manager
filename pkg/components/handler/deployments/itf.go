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

package deployments

import (
	"context"

	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

type databaseHandler interface {
	CreateDeployment(
		ctx context.Context,
		deployment pkg_models.DeploymentBase,
		hostResources []pkg_models.DeploymentHostResource,
		secrets []pkg_models.DeploymentSecret,
		userConfigs []pkg_models.DeploymentUserConfig,
		globalConfigs []pkg_models.DeploymentGlobalConfig,
		files []pkg_models.DeploymentFile,
		fileGroups []pkg_models.DeploymentFileGroup,
		volumes []pkg_models.DeploymentVolume,
		containers []pkg_models.DeploymentContainerBase,
	) error
	ReadDeployments(
		ctx context.Context,
		filter pkg_models.DeploymentsFilter,
	) (map[string]pkg_models.DeploymentBase, error)
	ReadDeploymentsContainers(
		ctx context.Context,
		deploymentIds []string,
	) (map[string]map[string]pkg_models.DeploymentContainerBase, error)
	ReadDeploymentsVolumes(
		ctx context.Context,
		deploymentIds []string,
	) (map[string]map[string]pkg_models.DeploymentVolume, error)
	ReadDeploymentsHostResources(
		ctx context.Context,
		filter pkg_models.DeploymentsHostResourcesFilter,
	) (map[string]map[string]pkg_models.DeploymentHostResource, error)
	ReadDeploymentsSecrets(
		ctx context.Context,
		filter pkg_models.DeploymentsSecretsFilter,
	) (map[string]map[string]pkg_models.DeploymentSecret, error)
	ReadDeploymentsConfigs(
		ctx context.Context,
		deploymentIds []string,
	) (map[string]map[string]pkg_models.DeploymentUserConfig, error)
	ReadDeploymentsGlobalConfigs(
		ctx context.Context,
		filter pkg_models.DeploymentGlobalConfigsFilter,
	) (map[string]map[string]pkg_models.DeploymentGlobalConfig, error)
	ReadDeploymentsFiles(
		ctx context.Context,
		deploymentIds []string,
	) (map[string]map[string]pkg_models.DeploymentFile, error)
	ReadDeploymentsFileGroups(
		ctx context.Context,
		deploymentIds []string,
	) (map[string]map[string]pkg_models.DeploymentFileGroup, error)
	ReadGlobalConfigs(ctx context.Context, ids []string) (map[string]pkg_models.Config, error)
	UpdateDeploymentsEnabledState(ctx context.Context, deploymentIds []string, state bool) error
	UpdateDeploymentContainerNames(ctx context.Context, containers []pkg_models.DeploymentContainerBase) error
	UpdateDeployment(
		ctx context.Context,
		deployment pkg_models.DeploymentBase,
		hostResources []pkg_models.DeploymentHostResource,
		secrets []pkg_models.DeploymentSecret,
		userConfigs []pkg_models.DeploymentUserConfig,
		globalConfigs []pkg_models.DeploymentGlobalConfig,
		files []pkg_models.DeploymentFile,
		fileGroups []pkg_models.DeploymentFileGroup,
		volumes []pkg_models.DeploymentVolume,
		containers []pkg_models.DeploymentContainerBase,
	) (err error)
	DeleteDeployment(ctx context.Context, id string) error
}

type containerEngineWrapperClient interface {
	GetContainers(ctx context.Context, filter pkg_models.CewContainersFilter) ([]pkg_models.CewContainer, error)
	CreateContainer(ctx context.Context, container pkg_models.CewContainer) (id string, err error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string) (jobId string, err error)
	RestartContainer(ctx context.Context, id string) (jobId string, err error)
	RemoveContainer(ctx context.Context, id string, force bool) error
	GetImage(ctx context.Context, id string) (pkg_models.CewImage, error)
	AddImage(ctx context.Context, img string) (jobId string, err error)
	GetVolumes(ctx context.Context, filter pkg_models.CewVolumesFilter) ([]pkg_models.CewVolume, error)
	CreateVolume(ctx context.Context, vol pkg_models.CewVolume) (string, error)
	RemoveVolume(ctx context.Context, id string, force bool) error
	GetJob(ctx context.Context, id string) (pkg_models.JobLibJob, error)
	CancelJob(ctx context.Context, id string) error
}

type hostManagerClient interface {
	GetHostResource(ctx context.Context, id string) (pkg_models.HmHostResource, error)
}

type secretManagerClient interface {
	InitPathVariant(ctx context.Context, secretRequest pkg_models.SmSecretVariantRequest) (variant pkg_models.SmSecretPathVariant, err error, errCode int)
	LoadPathVariant(ctx context.Context, secretRequest pkg_models.SmSecretVariantRequest) (err error, errCode int)
	UnloadPathVariant(ctx context.Context, secretRequest pkg_models.SmSecretVariantRequest) (err error, errCode int)
	CleanPathVariants(ctx context.Context, ref string) (err error, errCode int)
	GetValueVariant(ctx context.Context, secretRequest pkg_models.SmSecretVariantRequest) (variant pkg_models.SmSecretValueVariant, err error, errCode int)
}

type coreManagerClient interface {
	SetEndpoints(ctx context.Context, endpoints []pkg_models.CmEndpointBase) (string, error)
	RemoveEndpoints(ctx context.Context, filter pkg_models.CmEndpointFiler, restrictStd bool) (string, error)
	GetJob(ctx context.Context, id string) (pkg_models.JobLibJob, error)
	CancelJob(ctx context.Context, id string) error
}
