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

package aux_deployments

import (
	"context"

	lib_models_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/lib/models/aux_deployments"
	models_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/aux_deployments"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

type databaseHandler interface {
	ReadAuxiliaryDeployment(
		ctx context.Context,
		deploymentId string,
		auxDeploymentId string,
	) (models_aux_deployments.AuxiliaryDeployment, error)
	ReadAuxiliaryDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models_aux_deployments.AuxiliaryDeploymentsFilter,
	) (map[string]models_aux_deployments.AuxiliaryDeployment, error)
	ReadAuxiliaryDeploymentLabels(ctx context.Context, auxiliaryDeploymentId string) (map[string]string, error)
	ReadAuxiliaryDeploymentsLabels(ctx context.Context, auxDeploymentsIds []string) (map[string]map[string]string, error)
	ReadAuxiliaryDeploymentConfigs(ctx context.Context, auxiliaryDeploymentId string) (map[string]string, error)
	ReadAuxiliaryDeploymentsConfigs(ctx context.Context, auxDeploymentsIds []string) (map[string]map[string]string, error)
	ReadAuxiliaryDeploymentVolumeMounts(
		ctx context.Context,
		auxiliaryDeploymentId string,
	) ([]models_aux_deployments.AuxiliaryDeploymentVolumeMount, error)
	ReadAuxiliaryDeploymentsVolumeMounts(
		ctx context.Context,
		auxiliaryDeploymentsIds []string,
	) (map[string][]models_aux_deployments.AuxiliaryDeploymentVolumeMount, error)
	ReadAuxiliaryDeploymentVolumes(
		ctx context.Context,
		deploymentId string,
		refFilter []string,
	) (map[string]lib_models_aux_deployments.AuxiliaryDeploymentVolume, error)
	ReadAuxiliaryDeploymentVolumesWithMounts(
		ctx context.Context,
		deploymentId string,
		refFilter []string,
	) (map[string]lib_models_aux_deployments.AuxiliaryDeploymentVolumeWithMounts, error)
	ReadAuxDeploymentsByParent(ctx context.Context) (
		map[string]models_aux_deployments.AuxiliaryDeploymentParent,
		error,
	)
	CreateAuxiliaryDeploymentVolumes(
		ctx context.Context,
		deploymentId string,
		volumes []lib_models_aux_deployments.AuxiliaryDeploymentVolume,
	) error
	CreateAuxiliaryDeployment(
		ctx context.Context,
		auxiliaryDeployment models_aux_deployments.AuxiliaryDeployment,
		labels map[string]string,
		configs map[string]string,
		volumeMounts []models_aux_deployments.AuxiliaryDeploymentVolumeMount,
	) error
	UpdateAuxiliaryDeployment(
		ctx context.Context,
		auxiliaryDeployment models_aux_deployments.AuxiliaryDeployment,
		labels map[string]string,
		configs map[string]string,
		volumeMounts []models_aux_deployments.AuxiliaryDeploymentVolumeMount,
	) error
	UpdateAuxiliaryDeploymentContainerName(ctx context.Context, auxDeploymentId, name string) error
	UpdateAuxiliaryDeploymentsEnabledState(ctx context.Context, auxDeploymentIds []string, state bool) error
	DeleteAuxiliaryDeployment(ctx context.Context, auxDeploymentId string) error
	DeleteAuxiliaryDeployments(ctx context.Context, auxiliaryDeploymentsIds []string) error
	DeleteAuxiliaryDeploymentVolumes(ctx context.Context, deploymentId string, references []string) error
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
