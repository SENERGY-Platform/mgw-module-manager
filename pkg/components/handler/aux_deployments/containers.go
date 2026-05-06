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
	"maps"
	"path"

	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

func (h *Handler) createContainer(
	ctx context.Context,
	moduleAuxService pkg_models.ModuleLibAuxService,
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
	envVariables[pkg_models.EnvVariableCoreId] = helper_naming.CoreId
	envVariables[pkg_models.EnvVariableDeploymentId] = activeDeployment.Id
	envVariables[pkg_models.EnvVariableAuxDeploymentId] = auxDeployment.Id
	var mounts []pkg_models.CewMount
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
	auxServiceRunConfig pkg_models.ModuleLibRunConfig,
	deploymentId string,
	auxDeploymentId string,
	image string,
	containerAlias string,
	containerName string,
	runConfig lib_models.AuxiliaryDeploymentRunConfig,
	envVariables map[string]string,
	mounts []pkg_models.CewMount,
) pkg_models.CewContainer {
	return pkg_models.CewContainer{
		Name:    containerName,
		Image:   image,
		EnvVars: envVariables,
		Labels: map[string]string{
			pkg_models.LabelCoreId:                 helper_naming.CoreId,
			pkg_models.LabelManagerId:              helper_naming.ManagerId,
			pkg_models.LabelDeploymentId:           deploymentId,
			pkg_models.LabelAuxDeploymentId:        auxDeploymentId,
			pkg_models.LabelAuxDeploymentReference: auxServiceReference,
		},
		Mounts: mounts,
		Networks: []pkg_models.CewContainerNetwork{
			{
				Name:        helper_naming.ModuleContainerNetwork,
				DomainNames: []string{containerAlias, containerName},
			},
		},
		RunConfig: newCewRunConfig(auxServiceRunConfig, runConfig),
	}
}

func newCewRunConfig(
	auxServiceRunConfig pkg_models.ModuleLibRunConfig,
	runConfig lib_models.AuxiliaryDeploymentRunConfig,
) pkg_models.CewRunConfig {
	rc := pkg_models.CewRunConfig{
		RestartStrategy: pkg_models.CewRestartStrategyNever,
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
	mounts []pkg_models.CewMount,
	serviceBindMounts map[string]pkg_models.ModuleLibBindMount,
	deploymentDirName string,
	hostPath string,
) []pkg_models.CewMount {
	for mountPath, include := range serviceBindMounts {
		mounts = append(mounts, cew_model.Mount{
			Type:     pkg_models.CewMountTypeBind,
			Source:   path.Join(hostPath, deploymentDirName, include.Source),
			Target:   mountPath,
			ReadOnly: include.ReadOnly,
		})
	}
	return mounts
}

func appendTmpfsMounts(
	mounts []pkg_models.CewMount,
	serviceTmpfs map[string]pkg_models.ModuleLibTmpfsMount,
) []pkg_models.CewMount {
	for mountPath, tmpfs := range serviceTmpfs {
		mounts = append(mounts, pkg_models.CewMount{
			Type:   pkg_models.CewMountTypeTmpfs,
			Target: mountPath,
			Size:   tmpfs.Size,
			Mode:   tmpfs.Mode,
		})
	}
	return mounts
}

func appendVolumeMounts(
	mounts []pkg_models.CewMount,
	auxServiceVolumes map[string]string,
	deploymentVolumes map[string]pkg_models.DeploymentVolume,
	volumeMounts []pkg_models.AuxiliaryDeploymentVolumeMount,
) []pkg_models.CewMount {
	for mountPath, name := range auxServiceVolumes {
		volume, ok := deploymentVolumes[name]
		if ok {
			mounts = append(mounts, pkg_models.CewMount{
				Type:   pkg_models.CewMountTypeVolume,
				Source: volume.Name,
				Target: mountPath,
			})
		}
	}
	for _, mount := range volumeMounts {
		mounts = append(mounts, pkg_models.CewMount{
			Type:   pkg_models.CewMountTypeVolume,
			Source: mount.VolumeName,
			Target: mount.MountPath,
		})
	}
	return mounts
}

func setInternalDependencyEnvVariables(
	envVariables map[string]string,
	serviceReferences map[string]pkg_models.ModuleLibSrvRefTarget,
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
	serviceExtDependencies map[string]pkg_models.ModuleLibExtDependencyTarget,
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
