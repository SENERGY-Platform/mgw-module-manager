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

package handler_aux_deployments

import (
	"context"
	"maps"
	"path"

	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	lib_models_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/lib/models/aux_deployments"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/aux_deployments"
	models_constants "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	models_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/deployments"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) createContainer(
	ctx context.Context,
	moduleAuxService models_external.ModuleLibAuxService,
	auxServiceReference string,
	activeDeployment models_deployments.Deployment,
	dependencies map[string]models_deployments.DeploymentReduced,
	auxDeployment aux_deployments.AuxiliaryDeployment,
	configs map[string]string,
	volumeMounts []aux_deployments.AuxiliaryDeploymentVolumeMount,
) error {
	envVariables := make(map[string]string)
	maps.Copy(envVariables, configs)
	setInternalDependencyEnvVariables(envVariables, moduleAuxService.SrvReferences, activeDeployment.Containers)
	setExternalDependencyEnvVariables(envVariables, moduleAuxService.ExtDependencies, dependencies)
	envVariables[models_constants.EnvVariableCoreId] = helper_naming.CoreId
	envVariables[models_constants.EnvVariableDeploymentId] = activeDeployment.Id
	envVariables[models_constants.EnvVariableAuxDeploymentId] = auxDeployment.Id
	var mounts []models_external.CewMount
	mounts = appendIncludeMounts(mounts, moduleAuxService.BindMounts, activeDeployment.DirName, h.config.HostWorkDirPath)
	mounts = appendTmpfsMounts(mounts, moduleAuxService.Tmpfs)
	mounts = appendVolumeMounts(mounts, moduleAuxService.Volumes, activeDeployment.Volumes, volumeMounts)
	cewContainer := getCewContainer(
		auxServiceReference,
		moduleAuxService.RunConfig,
		activeDeployment.Id,
		auxDeployment.Id,
		auxDeployment.Image,
		auxDeployment.Container.Alias,
		auxDeployment.Container.Name,
		auxDeployment.RunConfig,
		envVariables,
		mounts,
	)
	_, err := h.containerEngineWrapperClient.CreateContainer(ctx, cewContainer)
	if err != nil {
		return err
	}
	return nil
}

func getCewContainer(
	auxServiceReference string,
	auxServiceRunConfig models_external.ModuleLibRunConfig,
	deploymentId string,
	auxDeploymentId string,
	image string,
	containerAlias string,
	containerName string,
	runConfig lib_models_aux_deployments.AuxiliaryDeploymentRunConfig,
	envVariables map[string]string,
	mounts []models_external.CewMount,
) models_external.Container {
	return models_external.Container{
		Name:    containerName,
		Image:   image,
		EnvVars: envVariables,
		Labels: map[string]string{
			models_constants.LabelCoreId:                 helper_naming.CoreId,
			models_constants.LabelManagerId:              helper_naming.ManagerId,
			models_constants.LabelDeploymentId:           deploymentId,
			models_constants.LabelAuxDeploymentId:        auxDeploymentId,
			models_constants.LabelAuxDeploymentReference: auxServiceReference,
		},
		Mounts: mounts,
		Networks: []models_external.CewContainerNetwork{
			{
				Name:        helper_naming.ModuleContainerNetwork,
				DomainNames: []string{containerAlias, containerName},
			},
		},
		RunConfig: newCewRunConfig(auxServiceRunConfig, runConfig),
	}
}

func newCewRunConfig(
	auxServiceRunConfig models_external.ModuleLibRunConfig,
	runConfig lib_models_aux_deployments.AuxiliaryDeploymentRunConfig,
) models_external.CewRunConfig {
	rc := models_external.CewRunConfig{
		RestartStrategy: models_external.CewRestartStrategyNever,
		PseudoTTY:       runConfig.PseudoTTY,
		Command:         runConfig.Command,
	}
	if auxServiceRunConfig.StopTimeout > 0 {
		rc.StopTimeout = &auxServiceRunConfig.StopTimeout // unnecessary pointer, change cew
	}
	if auxServiceRunConfig.StopSignal != "" {
		rc.StopSignal = &auxServiceRunConfig.StopSignal // unnecessary pointer, change cew
	}
	return rc
}

func appendIncludeMounts(
	mounts []models_external.CewMount,
	serviceBindMounts map[string]models_external.ModuleLibBindMount,
	deploymentDirName string,
	hostPath string,
) []models_external.CewMount {
	for mountPath, include := range serviceBindMounts {
		mounts = append(mounts, cew_model.Mount{
			Type:     models_external.CewMountTypeBind,
			Source:   path.Join(hostPath, deploymentDirName, include.Source),
			Target:   mountPath,
			ReadOnly: include.ReadOnly,
		})
	}
	return mounts
}

func appendTmpfsMounts(
	mounts []models_external.CewMount,
	serviceTmpfs map[string]models_external.ModuleLibTmpfsMount,
) []models_external.CewMount {
	for mountPath, tmpfs := range serviceTmpfs {
		mounts = append(mounts, models_external.CewMount{
			Type:   models_external.CewMountTypeTmpfs,
			Target: mountPath,
			Size:   tmpfs.Size,
			Mode:   tmpfs.Mode,
		})
	}
	return mounts
}

func appendVolumeMounts(
	mounts []models_external.CewMount,
	auxServiceVolumes map[string]string,
	deploymentVolumes map[string]models_deployments.DeploymentVolume,
	volumeMounts []aux_deployments.AuxiliaryDeploymentVolumeMount,
) []models_external.CewMount {
	for mountPath, name := range auxServiceVolumes {
		volume, ok := deploymentVolumes[name]
		if ok {
			mounts = append(mounts, models_external.CewMount{
				Type:   models_external.CewMountTypeVolume,
				Source: volume.Name,
				Target: mountPath,
			})
		}
	}
	for _, mount := range volumeMounts {
		mounts = append(mounts, models_external.CewMount{
			Type:   models_external.CewMountTypeVolume,
			Source: mount.VolumeName,
			Target: mount.MountPath,
		})
	}
	return mounts
}

func setInternalDependencyEnvVariables(
	envVariables map[string]string,
	serviceReferences map[string]models_external.ModuleLibSrvRefTarget,
	deploymentContainers map[string]models_deployments.Container,
) {
	for envVarName, target := range serviceReferences {
		container, ok := deploymentContainers[target.Ref]
		if ok {
			envVariables[envVarName] = target.FillTemplate(container.Alias)
		}
	}
}

func setExternalDependencyEnvVariables(
	envVariables map[string]string,
	serviceExtDependencies map[string]models_external.ModuleLibExtDependencyTarget,
	deployments map[string]models_deployments.DeploymentReduced,
) {
	for envVarName, target := range serviceExtDependencies {
		item, ok := deployments[target.ID]
		if !ok {
			continue
		}
		container, ok := item.Containers[target.Service]
		if ok {
			envVariables[envVarName] = target.FillTemplate(container.Alias)
		}
	}
}
