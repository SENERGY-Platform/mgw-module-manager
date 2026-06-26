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
	"errors"
	"maps"
	"path"

	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_containers "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/containers"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) createContainer(
	ctx context.Context,
	moduleAuxService external_models.ModuleLibAuxService,
	auxServiceReference string,
	activeDeployment pkg_models.Deployment,
	dependencies map[string]pkg_models.DeploymentReduced,
	auxDeployment pkg_models.AuxiliaryDeployment,
	configs map[string]string,
	volumeMounts []pkg_models.AuxiliaryDeploymentVolumeMount,
) error {
	envVariables := make(map[string]string)
	maps.Copy(envVariables, configs)
	setInternalDependencyEnvVariables(envVariables, moduleAuxService.SrvReferences, activeDeployment.Containers)
	setExternalDependencyEnvVariables(envVariables, moduleAuxService.ExtDependencies, dependencies)
	envVariables[constants.EnvVariableCoreId] = helper_naming.CoreId
	envVariables[constants.EnvVariableDeploymentId] = activeDeployment.Id
	envVariables[constants.EnvVariableAuxDeploymentId] = auxDeployment.Id
	var mounts []external_models.CewMount
	mounts = appendIncludeMounts(mounts, moduleAuxService.BindMounts, activeDeployment.DirName, h.config.HostDeploymentsPath)
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

func (h *Handler) stopContainer(ctx context.Context, containerName string) error {
	_, err := h.containerEngineWrapperClient.GetContainer(ctx, containerName)
	if err != nil {
		var notFoundErr *external_models.CewNotFoundErr
		if !errors.As(err, &notFoundErr) {
			return err
		}
		logger.ErrorContext(ctx, "stop container", slog_keys.Name, containerName, slog_keys.Error, "not found")
		return nil
	}
	return helper_containers.Stop(
		ctx,
		h.containerEngineWrapperClient,
		containerName,
		h.config.JobPollInterval,
	)
}

func getCewContainer(
	auxServiceReference string,
	auxServiceRunConfig external_models.ModuleLibRunConfig,
	deploymentId string,
	auxDeploymentId string,
	image string,
	containerAlias string,
	containerName string,
	runConfig lib_models.AuxiliaryDeploymentRunConfig,
	envVariables map[string]string,
	mounts []external_models.CewMount,
) external_models.CewContainer {
	return external_models.CewContainer{
		Name:    containerName,
		Image:   image,
		EnvVars: envVariables,
		Labels: map[string]string{
			constants.LabelCoreId:                 helper_naming.CoreId,
			constants.LabelManagerId:              helper_naming.ManagerId,
			constants.LabelDeploymentId:           deploymentId,
			constants.LabelAuxDeploymentId:        auxDeploymentId,
			constants.LabelAuxDeploymentReference: auxServiceReference,
		},
		Mounts: mounts,
		Networks: []external_models.CewContainerNetwork{
			{
				Name:        helper_naming.ModuleContainerNetwork,
				DomainNames: []string{containerAlias, containerName},
			},
		},
		RunConfig: newCewRunConfig(auxServiceRunConfig, runConfig),
	}
}

func newCewRunConfig(
	auxServiceRunConfig external_models.ModuleLibRunConfig,
	runConfig lib_models.AuxiliaryDeploymentRunConfig,
) external_models.CewRunConfig {
	rc := external_models.CewRunConfig{
		RestartStrategy: external_models.CewRestartStrategyNever,
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
	mounts []external_models.CewMount,
	serviceBindMounts map[string]external_models.ModuleLibBindMount,
	deploymentDirName string,
	hostPath string,
) []external_models.CewMount {
	for mountPath, include := range serviceBindMounts {
		mounts = append(mounts, cew_model.Mount{
			Type:     external_models.CewMountTypeBind,
			Source:   path.Join(hostPath, deploymentDirName, include.Source),
			Target:   mountPath,
			ReadOnly: include.ReadOnly,
		})
	}
	return mounts
}

func appendTmpfsMounts(
	mounts []external_models.CewMount,
	serviceTmpfs map[string]external_models.ModuleLibTmpfsMount,
) []external_models.CewMount {
	for mountPath, tmpfs := range serviceTmpfs {
		mounts = append(mounts, external_models.CewMount{
			Type:   external_models.CewMountTypeTmpfs,
			Target: mountPath,
			Size:   tmpfs.Size,
			Mode:   tmpfs.Mode,
		})
	}
	return mounts
}

func appendVolumeMounts(
	mounts []external_models.CewMount,
	auxServiceVolumes map[string]string,
	deploymentVolumes map[string]pkg_models.DeploymentVolume,
	volumeMounts []pkg_models.AuxiliaryDeploymentVolumeMount,
) []external_models.CewMount {
	for mountPath, name := range auxServiceVolumes {
		volume, ok := deploymentVolumes[name]
		if ok {
			mounts = append(mounts, external_models.CewMount{
				Type:   external_models.CewMountTypeVolume,
				Source: volume.Name,
				Target: mountPath,
			})
		}
	}
	for _, mount := range volumeMounts {
		mounts = append(mounts, external_models.CewMount{
			Type:   external_models.CewMountTypeVolume,
			Source: mount.VolumeName,
			Target: mount.MountPath,
		})
	}
	return mounts
}

func setInternalDependencyEnvVariables(
	envVariables map[string]string,
	serviceReferences map[string]external_models.ModuleLibSrvRefTarget,
	deploymentContainers map[string]pkg_models.DeploymentContainer,
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
	serviceExtDependencies map[string]external_models.ModuleLibExtDependencyTarget,
	deployments map[string]pkg_models.DeploymentReduced,
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
