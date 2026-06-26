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

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

type databaseHandler interface {
	ReadAuxiliaryDeployment(
		ctx context.Context,
		deploymentId string,
		auxDeploymentId string,
	) (pkg_models.AuxiliaryDeployment, error)
	ReadAuxiliaryDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models.AuxiliaryDeploymentsFilter,
	) (map[string]pkg_models.AuxiliaryDeployment, error)
	ReadAuxiliaryDeploymentLabels(ctx context.Context, auxiliaryDeploymentId string) (map[string]string, error)
	ReadAuxiliaryDeploymentsLabels(ctx context.Context, auxDeploymentsIds []string) (map[string]map[string]string, error)
	ReadAuxiliaryDeploymentConfigs(ctx context.Context, auxiliaryDeploymentId string) (map[string]string, error)
	ReadAuxiliaryDeploymentsConfigs(ctx context.Context, auxDeploymentsIds []string) (map[string]map[string]string, error)
	ReadAuxiliaryDeploymentVolumeMounts(
		ctx context.Context,
		auxiliaryDeploymentId string,
	) ([]pkg_models.AuxiliaryDeploymentVolumeMount, error)
	ReadAuxiliaryDeploymentsVolumeMounts(
		ctx context.Context,
		auxiliaryDeploymentsIds []string,
	) (map[string][]pkg_models.AuxiliaryDeploymentVolumeMount, error)
	ReadAuxiliaryDeploymentVolumes(
		ctx context.Context,
		deploymentId string,
		refFilter []string,
	) (map[string]lib_models.AuxiliaryDeploymentVolume, error)
	ReadAuxiliaryDeploymentVolumesWithMounts(
		ctx context.Context,
		deploymentId string,
		refFilter []string,
	) (map[string]lib_models.AuxiliaryDeploymentVolumeWithMounts, error)
	ReadAuxDeploymentsByParent(ctx context.Context) (
		map[string]pkg_models.AuxiliaryDeploymentParent,
		error,
	)
	CreateAuxiliaryDeploymentVolumes(
		ctx context.Context,
		deploymentId string,
		volumes []lib_models.AuxiliaryDeploymentVolume,
	) error
	CreateAuxiliaryDeployment(
		ctx context.Context,
		auxiliaryDeployment pkg_models.AuxiliaryDeployment,
		labels map[string]string,
		configs map[string]string,
		volumeMounts []pkg_models.AuxiliaryDeploymentVolumeMount,
	) error
	UpdateAuxiliaryDeployment(
		ctx context.Context,
		auxiliaryDeployment pkg_models.AuxiliaryDeployment,
		labels map[string]string,
		configs map[string]string,
		volumeMounts []pkg_models.AuxiliaryDeploymentVolumeMount,
	) error
	UpdateAuxiliaryDeploymentContainerName(ctx context.Context, auxDeploymentId, name string) error
	UpdateAuxiliaryDeploymentsEnabledState(ctx context.Context, auxDeploymentIds []string, state bool) error
	DeleteAuxiliaryDeployment(ctx context.Context, auxDeploymentId string) error
	DeleteAuxiliaryDeployments(ctx context.Context, auxiliaryDeploymentsIds []string) error
	DeleteAuxiliaryDeploymentVolumes(ctx context.Context, deploymentId string, references []string) error
}

type containerEngineWrapperClient interface {
	GetContainer(ctx context.Context, id string) (external_models.CewContainer, error)
	GetContainers(ctx context.Context, filter external_models.CewContainersFilter) ([]external_models.CewContainer, error)
	CreateContainer(ctx context.Context, container external_models.CewContainer) (id string, err error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string) (jobId string, err error)
	RestartContainer(ctx context.Context, id string) (jobId string, err error)
	RemoveContainer(ctx context.Context, id string, force bool) error
	GetImage(ctx context.Context, id string) (external_models.CewImage, error)
	AddImage(ctx context.Context, img string) (jobId string, err error)
	GetVolumes(ctx context.Context, filter external_models.CewVolumesFilter) ([]external_models.CewVolume, error)
	CreateVolume(ctx context.Context, vol external_models.CewVolume) (string, error)
	RemoveVolume(ctx context.Context, id string, force bool) error
	GetJob(ctx context.Context, id string) (external_models.JobLibJob, error)
	CancelJob(ctx context.Context, id string) error
}
