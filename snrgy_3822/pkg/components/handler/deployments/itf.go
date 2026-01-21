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
	"time"

	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

type storageHandler interface {
	CreateDeployment(
		ctx context.Context,
		deployment models_handler_storage.Deployment,
		hostResources []models_handler_storage.DeploymentHostResource,
		secrets []models_handler_storage.DeploymentSecret,
		configs []models_handler_storage.DeploymentConfig,
		globalConfigs []models_handler_storage.DeploymentGlobalConfig,
	) (err error)
	ReadDeployment(ctx context.Context, id string) (models_handler_storage.Deployment, error)
	ReadDeployments(ctx context.Context, filter models_handler_storage.DeploymentsFilter) (map[string]models_handler_storage.Deployment, error)
	ReadDeploymentContainers(ctx context.Context, deploymentId string) ([]models_handler_storage.DeploymentContainer, error)
	ReadDeploymentsContainers(ctx context.Context, deploymentIds []string) (map[string][]models_handler_storage.DeploymentContainer, error)
	ReadDeploymentHostResources(ctx context.Context, deploymentId string) ([]models_handler_storage.DeploymentHostResource, error)
	ReadDeploymentsHostResources(ctx context.Context, filter models_handler_storage.DeploymentsHostResourcesFilter) (map[string][]models_handler_storage.DeploymentHostResource, error)
	ReadDeploymentSecrets(ctx context.Context, deploymentId string) ([]models_handler_storage.DeploymentSecret, error)
	ReadDeploymentsSecrets(ctx context.Context, filter models_handler_storage.DeploymentsSecretsFilter) (map[string][]models_handler_storage.DeploymentSecret, error)
	ReadDeploymentConfigs(ctx context.Context, deploymentId string) ([]models_handler_storage.DeploymentConfig, error)
	ReadDeploymentsConfigs(ctx context.Context, deploymentIds []string) (map[string][]models_handler_storage.DeploymentConfig, error)
	ReadDeploymentGlobalConfigs(ctx context.Context, deploymentId string) ([]models_handler_storage.DeploymentGlobalConfig, error)
	ReadDeploymentsGlobalConfigs(ctx context.Context, filter models_handler_storage.DeploymentGlobalConfigsFilter) (map[string][]models_handler_storage.DeploymentGlobalConfig, error)
	UpdateDeploymentsEnabledState(ctx context.Context, deployments map[string]bool, timestamp time.Time) (err error)
	UpdateDeploymentEnabledState(ctx context.Context, id string, enabled bool, timestamp time.Time) error
	UpdateDeploymentName(ctx context.Context, id, name string, timestamp time.Time) error
	UpdateDeploymentResourcesAndConfigs(
		ctx context.Context,
		deploymentId string,
		hostResources []models_handler_storage.DeploymentHostResource,
		secrets []models_handler_storage.DeploymentSecret,
		configs []models_handler_storage.DeploymentConfig,
		globalConfigs []models_handler_storage.DeploymentGlobalConfig,
	) (err error)
	UpdateDeployment(
		ctx context.Context,
		deployment models_handler_storage.Deployment,
		hostResources []models_handler_storage.DeploymentHostResource,
		secrets []models_handler_storage.DeploymentSecret,
		configs []models_handler_storage.DeploymentConfig,
		globalConfigs []models_handler_storage.DeploymentGlobalConfig,
	) (err error)
	DeleteDeployment(ctx context.Context, id string) error
	DeleteDeployments(ctx context.Context, ids []string) error
	ReadGlobalConfigs(ctx context.Context, ids []string) (map[string]models_handler_storage.GlobalConfig, error)
}

type containerEngineWrapperClient interface {
	GetContainers(ctx context.Context, filter models_external.ContainersFilter) ([]models_external.Container, error)
	GetContainer(ctx context.Context, id string) (models_external.Container, error)
	CreateContainer(ctx context.Context, container models_external.Container) (id string, err error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string) (jobId string, err error)
	RestartContainer(ctx context.Context, id string) (jobId string, err error)
	RemoveContainer(ctx context.Context, id string, force bool) error
	GetImage(ctx context.Context, id string) (models_external.Image, error)
	AddImage(ctx context.Context, img string) (jobId string, err error)
	GetImages(ctx context.Context, filter models_external.ImagesFilter) ([]models_external.Image, error)
	GetVolumes(ctx context.Context, filter models_external.VolumesFilter) ([]models_external.Volume, error)
	GetVolume(ctx context.Context, id string) (models_external.Volume, error)
	CreateVolume(ctx context.Context, vol models_external.Volume) (string, error)
	RemoveVolume(ctx context.Context, id string, force bool) error
	GetJob(ctx context.Context, id string) (models_external.Job, error)
	CancelJob(ctx context.Context, id string) error
}

type hostManagerClient interface {
	GetHostResource(ctx context.Context, id string) (models_external.HostResource, error)
}
