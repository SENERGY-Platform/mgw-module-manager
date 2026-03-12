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

package deployments

import (
	"path"

	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func newCewContainers(
	deployment *deploymentWrapper,
	cache *cacheWrapper,
	configs map[string]string,
	secretMounts map[string]models_external.SecretPathVariant,
	fileMounts map[string]string,
	fileGroupMounts map[string][]fileGroupMount,
	hostWorkDirPath string,
	hostSecretsPath string,
) []models_external.Container {
	var containers []models_external.Container
	for reference, container := range deployment.Containers {
		envVariables := make(map[string]string)
		setConfigEnvVariables(envVariables, container.Service.Configs, configs)
		setSecretValueEnvVariables(envVariables, container.Service.SecretVars, deployment.SelectedSecrets, cache.SecretValues)
		setInternalDependencyEnvVariables(envVariables, container.Service.SrvReferences, deployment.Containers)
		setExternalDependencyEnvVariables(envVariables, container.Service.ExtDependencies, cache.ExternalDependencies)
		envVariables[constants.EnvVariableDeploymentId] = deployment.Id
		var mounts []models_external.CewMount
		mounts = appendIncludeMounts(mounts, container.Service.BindMounts, hostWorkDirPath, deployment.DirName)
		mounts = appendTmpfsMounts(mounts, container.Service.Tmpfs)
		mounts = appendVolumeMounts(mounts, container.Service.Volumes, deployment.Volumes)
		mounts = appendApplicationMounts(mounts, container.Service.HostResources, deployment.SelectedHostResources, cache.HostResources)
		mounts = appendSecretMounts(mounts, container.Service.SecretMounts, deployment.SelectedSecrets, secretMounts, hostSecretsPath)
		mounts = appendFileMounts(mounts, container.Service.Files, fileMounts, hostWorkDirPath, deployment.FilesDirName)
		mounts = appendFileGroupMounts(mounts, container.Service.FileGroups, fileGroupMounts, hostWorkDirPath, deployment.FilesDirName)
		containers = append(containers, models_external.Container{
			Name:    container.Name,
			Image:   container.Service.Image,
			EnvVars: envVariables,
			Labels: map[string]string{
				constants.LabelCoreId:           helper_naming.CoreId,
				constants.LabelManagerId:        helper_naming.ManagerId,
				constants.LabelDeploymentId:     deployment.Id,
				constants.LabelServiceReference: reference,
			},
			Mounts:            mounts,
			Devices:           getDevices(container.Service.HostResources, deployment.SelectedHostResources, cache.HostResources),
			DeviceCGroupRules: container.Service.DeviceCGroupRules,
			Ports:             newCewPorts(container.Service.Ports),
			Networks: []models_external.CewContainerNetwork{
				{
					Name:        helper_naming.ModuleContainerNetwork,
					DomainNames: []string{container.Alias, container.Name},
				},
			},
			RunConfig: newCewRunConfig(container.Service.RunConfig),
		})
	}
	return containers
}

func setConfigEnvVariables(envVariables map[string]string, serviceConfigs map[string]string, configs map[string]string) {
	for envVarName, reference := range serviceConfigs {
		value, ok := configs[reference]
		if ok {
			envVariables[envVarName] = value
		}
	}
}

func setSecretValueEnvVariables(
	envVariables map[string]string,
	serviceSecretTargets map[string]models_external.ModuleSecretTarget,
	selectedSecrets map[string]models_handler_storage.DeploymentSecret,
	secretValuesCache map[string]models_external.SecretValueVariant,
) {
	for envVarName, target := range serviceSecretTargets {
		selectedSecret, ok := selectedSecrets[target.Ref]
		if !ok {
			continue
		}
		valueVariant, ok := secretValuesCache[selectedSecret.Id+target.Item]
		if !ok {
			continue
		}
		envVariables[envVarName] = valueVariant.Value
	}
}

func setInternalDependencyEnvVariables(
	envVariables map[string]string,
	internalDependencyTargets map[string]models_external.ModuleInternalDependencyTarget,
	containers map[string]containerWrapper,
) {
	for envVarName, target := range internalDependencyTargets {
		container, ok := containers[target.Ref]
		if ok {
			envVariables[envVarName] = target.FillTemplate(container.Alias)
		}
	}
}

func setExternalDependencyEnvVariables(
	envVariables map[string]string,
	externalDependencyTargets map[string]models_external.ModuleExternalDependencyTarget,
	externalDependenciesCache map[string]map[string]models_handler_storage.DeploymentContainer,
) {
	for envVarName, target := range externalDependencyTargets {
		containers, ok := externalDependenciesCache[target.ID]
		if !ok {
			continue
		}
		container, ok := containers[target.Service]
		if ok {
			envVariables[envVarName] = target.FillTemplate(container.Alias)
		}
	}
}

func appendIncludeMounts(
	mounts []models_external.CewMount,
	serviceIncludes map[string]models_external.ModuleServiceIncludeMount,
	hostPath string,
	dirName string,
) []models_external.CewMount {
	for mountPath, include := range serviceIncludes {
		mounts = append(mounts, cew_model.Mount{
			Type:     models_external.CewMountTypeBind,
			Source:   path.Join(hostPath, dirName, include.Source),
			Target:   mountPath,
			ReadOnly: include.ReadOnly,
		})
	}
	return mounts
}

func appendTmpfsMounts(
	mounts []models_external.CewMount,
	serviceTmpfsTargets map[string]models_external.ModuleServiceTmpfsMount,
) []models_external.CewMount {
	for mountPath, tmpfs := range serviceTmpfsTargets {
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
	serviceVolumes map[string]string,
	volumes map[string]models_handler_storage.DeploymentVolume,
) []models_external.CewMount {
	for mountPath, name := range serviceVolumes {
		volume, ok := volumes[name]
		if ok {
			mounts = append(mounts, models_external.CewMount{
				Type:   models_external.CewMountTypeVolume,
				Source: volume.Name,
				Target: mountPath,
			})
		}
	}
	return mounts
}

func appendApplicationMounts(
	mounts []models_external.CewMount,
	serviceHostResources map[string]models_external.ModuleServiceHostResourceTarget,
	hostResources map[string]models_handler_storage.DeploymentHostResource,
	hostResourcesCache map[string]models_external.HostResource,
) []models_external.CewMount {
	for mountPath, srvHostResource := range serviceHostResources {
		tmp, ok := hostResources[srvHostResource.Ref]
		if !ok {
			continue
		}
		hostResource, ok := hostResourcesCache[tmp.Id]
		if !ok || hostResource.Type == models_external.HostResourceTypeDevice {
			continue
		}
		mounts = append(mounts, models_external.CewMount{
			Type:     models_external.CewMountTypeBind,
			Source:   hostResource.Path,
			Target:   mountPath,
			ReadOnly: srvHostResource.ReadOnly,
		})
	}
	return mounts
}

func appendSecretMounts(
	mounts []models_external.CewMount,
	serviceSecretTargets map[string]models_external.ModuleSecretTarget,
	selectedSecrets map[string]models_handler_storage.DeploymentSecret,
	secretMounts map[string]models_external.SecretPathVariant,
	hostPath string,
) []models_external.CewMount {
	for mountPath, target := range serviceSecretTargets {
		selectedSecret, ok := selectedSecrets[target.Ref]
		if !ok {
			continue
		}
		pathVariant, ok := secretMounts[selectedSecret.Id+target.Item]
		if !ok {
			continue
		}
		mounts = append(mounts, models_external.CewMount{
			Type:     models_external.CewMountTypeBind,
			Source:   path.Join(hostPath, pathVariant.Path),
			Target:   mountPath,
			ReadOnly: true,
		})
	}
	return mounts
}

func appendFileMounts(
	mounts []models_external.CewMount,
	serviceFiles map[string]string,
	fileMounts map[string]string,
	hostPath string,
	dirName string,
) []models_external.CewMount {
	for mountPoint, reference := range serviceFiles {
		fileName, ok := fileMounts[reference]
		if ok {
			mounts = append(mounts, models_external.CewMount{
				Type:     models_external.CewMountTypeBind,
				Source:   path.Join(hostPath, dirName, fileName),
				Target:   mountPoint,
				ReadOnly: true,
			})
		}
	}
	return mounts
}

func appendFileGroupMounts(
	mounts []models_external.CewMount,
	serviceFileGroups map[string]string,
	fileGroupMounts map[string][]fileGroupMount,
	hostPath string,
	dirName string,
) []models_external.CewMount {
	for basePath, reference := range serviceFileGroups {
		fileMounts, ok := fileGroupMounts[reference]
		if !ok {
			continue
		}
		for _, fileMount := range fileMounts {
			mounts = append(mounts, models_external.CewMount{
				Type:     models_external.CewMountTypeBind,
				Source:   path.Join(hostPath, dirName, fileMount.FileName),
				Target:   path.Join(basePath, fileMount.Path),
				ReadOnly: true,
			})
		}
	}
	return mounts
}

func getDevices(
	serviceHostResources map[string]models_external.ModuleServiceHostResourceTarget,
	hostResources map[string]models_handler_storage.DeploymentHostResource,
	hostResourcesCache map[string]models_external.HostResource,
) []models_external.CewDevice {
	var devices []models_external.CewDevice
	for mountPath, srvHostResource := range serviceHostResources {
		tmp, ok := hostResources[srvHostResource.Ref]
		if !ok {
			continue
		}
		hostResource, ok := hostResourcesCache[tmp.Id]
		if !ok || hostResource.Type == models_external.HostResourceTypeApp {
			continue
		}
		devices = append(devices, models_external.CewDevice{
			Source:   hostResource.Path,
			Target:   mountPath,
			ReadOnly: srvHostResource.ReadOnly,
		})
	}
	return devices
}

func newCewPorts(servicePorts []models_external.ModuleServicePort) (ports []models_external.CewPort) {
	for _, port := range servicePorts {
		p := models_external.CewPort{
			Number:   port.Number,
			Protocol: port.Protocol,
		}
		if len(port.Bindings) > 0 {
			var bindings []models_external.CewPortBinding
			for _, n := range port.Bindings {
				bindings = append(bindings, models_external.CewPortBinding{Number: n})
			}
			p.Bindings = bindings
		}
		ports = append(ports, p)
	}
	return ports
}

func newCewRunConfig(mrc models_external.ModuleServiceRunConfig) models_external.CewRunConfig {
	rc := models_external.CewRunConfig{
		RestartStrategy: models_external.CewRestartStrategyNever, // restarts should be handled by module-manager
		Retries:         nil,                                     // sollte von health handler verwendet werden?
		RemoveAfterRun:  false,                                   // wird das benutzt?
		PseudoTTY:       mrc.PseudoTTY,
		Command:         mrc.Command,
	}
	if mrc.StopTimeout > 0 {
		rc.StopTimeout = &mrc.StopTimeout // pointer unnötig, cew anpassen
	}
	if mrc.StopSignal != "" {
		rc.StopSignal = &mrc.StopSignal // pointer unnötig, cew anpassen
	}
	return rc
}

func newVolumes(moduleVolumes map[string]struct{}, deploymentId string) map[string]models_handler_storage.DeploymentVolume {
	volumes := make(map[string]models_handler_storage.DeploymentVolume)
	for reference := range moduleVolumes {
		volumes[reference] = models_handler_storage.DeploymentVolume{
			DeploymentId: deploymentId,
			Reference:    reference,
			Name:         helper_naming.NewVolumeName(deploymentId, reference),
		}
	}
	return volumes
}
