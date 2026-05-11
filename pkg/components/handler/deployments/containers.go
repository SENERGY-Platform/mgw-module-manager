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
	"context"
	"fmt"
	"path"

	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	helper_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/configs"
	helper_containers "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/containers"
	helper_errors "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/errors"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) createContainers(
	ctx context.Context,
	moduleConfigs external_models.ModuleLibConfigs,
	moduleServices map[string]external_models.ModuleLibService,
	deploymentId string,
	deploymentDirName string,
	deploymentFilesDirName string,
	userDataSecrets map[string]pkg_models.DeploymentSecret,
	userDataHostResources map[string]pkg_models.DeploymentHostResource,
	containers map[string]pkg_models.DeploymentContainerBase,
	volumes map[string]pkg_models.DeploymentVolume,
	configs map[string]pkg_models.Value,
	bindMounts bindMountDataCollection,
	cacheSecretValues map[string]external_models.SmSecretValueVariant,
	cacheDeployments map[string]deploymentsCacheItem,
	cacheHostResources map[string]external_models.HmHostResource,
) error {
	var errs []error
	for reference, service := range moduleServices {
		envVariables := make(map[string]string)
		setConfigEnvVariables(envVariables, service.Configs, configsToStrings(moduleConfigs, configs))
		setSecretValueEnvVariables(envVariables, service.SecretVars, userDataSecrets, cacheSecretValues)
		setInternalDependencyEnvVariables(envVariables, service.SrvReferences, containers)
		setExternalDependencyEnvVariables(envVariables, service.ExtDependencies, cacheDeployments)
		envVariables[constants.EnvVariableCoreId] = helper_naming.CoreId
		envVariables[constants.EnvVariableDeploymentId] = deploymentId
		var mounts []external_models.CewMount
		mounts = appendIncludeMounts(mounts, service.BindMounts, deploymentDirName, h.config.HostWorkDirPath)
		mounts = appendTmpfsMounts(mounts, service.Tmpfs)
		mounts = appendVolumeMounts(mounts, service.Volumes, volumes)
		mounts = appendApplicationMounts(mounts, service.HostResources, userDataHostResources, cacheHostResources)
		mounts = appendSecretMounts(mounts, service.SecretMounts, userDataSecrets, bindMounts.Secrets, h.config.HostSecretsPath)
		mounts = appendFileMounts(mounts, service.Files, deploymentFilesDirName, bindMounts.Files, h.config.HostWorkDirPath)
		mounts = appendFileGroupMounts(mounts, service.FileGroups, deploymentFilesDirName, bindMounts.FileGroups, h.config.HostWorkDirPath)
		storageContainer := containers[reference]
		cewContainer := getCewContainer(
			service.Image,
			service.DeviceCGroupRules,
			service.Ports,
			service.RunConfig,
			reference,
			deploymentId,
			storageContainer.Alias,
			storageContainer.Name,
			envVariables,
			mounts,
			getContainerDevices(service.HostResources, userDataHostResources, cacheHostResources),
		)
		_, err := h.containerEngineWrapperClient.CreateContainer(ctx, cewContainer)
		if err != nil {
			errs = append(errs, fmt.Errorf("'%s' %w", reference, err))
			continue
		}
	}
	if len(errs) > 0 {
		return helper_errors.Join(errs...)
	}
	return nil
}

func (h *Handler) removeContainers(
	ctx context.Context,
	deploymentContainers map[string]pkg_models.DeploymentContainerBase,
) error {
	var errs []error
	for _, container := range deploymentContainers {
		err := helper_containers.Remove(ctx, h.containerEngineWrapperClient, container.Name)
		if err != nil {
			errs = append(errs, fmt.Errorf("'%s' %w", container.Name, err))
		}
	}
	if len(errs) > 0 {
		return helper_errors.Join(errs...)
	}
	return nil
}

func (h *Handler) startContainers(
	ctx context.Context,
	deploymentContainers map[string]pkg_models.DeploymentContainerBase,
) error {
	var errs []error
	for _, container := range deploymentContainers {
		err := h.containerEngineWrapperClient.StartContainer(ctx, container.Name)
		if err != nil {
			errs = append(errs, fmt.Errorf("'%s' %w", container.Reference, err))
		}
	}
	if len(errs) > 0 {
		return helper_errors.Join(errs...)
	}
	return nil
}

func (h *Handler) stopContainers(
	ctx context.Context,
	deploymentContainers map[string]pkg_models.DeploymentContainerBase,
) error {
	var errs []error
	for _, container := range deploymentContainers {
		err := helper_containers.Stop(ctx, h.containerEngineWrapperClient, container.Name, h.config.JobPollInterval)
		if err != nil {
			errs = append(errs, fmt.Errorf("'%s' %w", container.Reference, err))
		}
	}
	if len(errs) > 0 {
		return helper_errors.Join(errs...)
	}
	return nil
}

func getCewContainer(
	serviceImage string,
	serviceDeviceCGroupRules []string,
	servicePorts []external_models.ModuleLibPort,
	serviceRunConfig external_models.ModuleLibRunConfig,
	serviceReference string,
	deploymentId string,
	containerAlias string,
	containerName string,
	envVariables map[string]string,
	mounts []external_models.CewMount,
	devices []external_models.CewDevice,
) external_models.CewContainer {
	return external_models.CewContainer{
		Name:    containerName,
		Image:   serviceImage,
		EnvVars: envVariables,
		Labels: map[string]string{
			constants.LabelCoreId:           helper_naming.CoreId,
			constants.LabelManagerId:        helper_naming.ManagerId,
			constants.LabelDeploymentId:     deploymentId,
			constants.LabelServiceReference: serviceReference,
		},
		Mounts:            mounts,
		Devices:           devices,
		DeviceCGroupRules: serviceDeviceCGroupRules,
		Ports:             newCewPorts(servicePorts),
		Networks: []external_models.CewContainerNetwork{
			{
				Name:        helper_naming.ModuleContainerNetwork,
				DomainNames: []string{containerAlias, containerName},
			},
		},
		RunConfig: newCewRunConfig(serviceRunConfig),
	}
}

func setConfigEnvVariables(
	envVariables map[string]string,
	serviceConfigs map[string]string,
	configs map[string]string,
) {
	for envVarName, reference := range serviceConfigs {
		value, ok := configs[reference]
		if ok {
			envVariables[envVarName] = value
		}
	}
}

func setSecretValueEnvVariables(
	envVariables map[string]string,
	serviceSecretVars map[string]external_models.ModuleLibSecretTarget,
	userDataSecrets map[string]pkg_models.DeploymentSecret,
	cacheSecretValues map[string]external_models.SmSecretValueVariant,
) {
	for envVarName, target := range serviceSecretVars {
		selectedSecret, ok := userDataSecrets[target.Ref]
		if !ok {
			continue
		}
		valueVariant, ok := cacheSecretValues[selectedSecret.Id+target.Item]
		if !ok {
			continue
		}
		envVariables[envVarName] = valueVariant.Value
	}
}

func setInternalDependencyEnvVariables(
	envVariables map[string]string,
	serviceReferences map[string]external_models.ModuleLibSrvRefTarget,
	deploymentContainers map[string]pkg_models.DeploymentContainerBase,
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
	cacheDeployments map[string]deploymentsCacheItem,
) {
	for envVarName, target := range serviceExtDependencies {
		item, ok := cacheDeployments[target.ID]
		if !ok {
			continue
		}
		container, ok := item.Containers[target.Service]
		if ok {
			envVariables[envVarName] = target.FillTemplate(container.Alias)
		}
	}
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
	serviceVolumes map[string]string,
	deploymentVolumes map[string]pkg_models.DeploymentVolume,
) []external_models.CewMount {
	for mountPath, name := range serviceVolumes {
		volume, ok := deploymentVolumes[name]
		if ok {
			mounts = append(mounts, external_models.CewMount{
				Type:   external_models.CewMountTypeVolume,
				Source: volume.Name,
				Target: mountPath,
			})
		}
	}
	return mounts
}

func appendApplicationMounts(
	mounts []external_models.CewMount,
	serviceHostResources map[string]external_models.ModuleLibHostResTarget,
	userDataHostResources map[string]pkg_models.DeploymentHostResource,
	cacheHostResources map[string]external_models.HmHostResource,
) []external_models.CewMount {
	for mountPath, srvHostResource := range serviceHostResources {
		tmp, ok := userDataHostResources[srvHostResource.Ref]
		if !ok {
			continue
		}
		hostResource, ok := cacheHostResources[tmp.Id]
		if !ok || hostResource.Type == external_models.HmHostResourceTypeDevice {
			continue
		}
		mounts = append(mounts, external_models.CewMount{
			Type:     external_models.CewMountTypeBind,
			Source:   hostResource.Path,
			Target:   mountPath,
			ReadOnly: srvHostResource.ReadOnly,
		})
	}
	return mounts
}

func appendSecretMounts(
	mounts []external_models.CewMount,
	serviceSecretMounts map[string]external_models.ModuleLibSecretTarget,
	userDataSecrets map[string]pkg_models.DeploymentSecret,
	secretMounts map[string]external_models.SmSecretPathVariant,
	hostPath string,
) []external_models.CewMount {
	for mountPath, target := range serviceSecretMounts {
		selectedSecret, ok := userDataSecrets[target.Ref]
		if !ok {
			continue
		}
		pathVariant, ok := secretMounts[selectedSecret.Id+target.Item]
		if !ok {
			continue
		}
		mounts = append(mounts, external_models.CewMount{
			Type:     external_models.CewMountTypeBind,
			Source:   path.Join(hostPath, pathVariant.Path),
			Target:   mountPath,
			ReadOnly: true,
		})
	}
	return mounts
}

func appendFileMounts(
	mounts []external_models.CewMount,
	serviceFiles map[string]string,
	deploymentFilesDirName string,
	fileMounts map[string]string,
	hostPath string,
) []external_models.CewMount {
	for mountPoint, reference := range serviceFiles {
		fileName, ok := fileMounts[reference]
		if ok {
			mounts = append(mounts, external_models.CewMount{
				Type:     external_models.CewMountTypeBind,
				Source:   path.Join(hostPath, deploymentFilesDirName, fileName),
				Target:   mountPoint,
				ReadOnly: true,
			})
		}
	}
	return mounts
}

func appendFileGroupMounts(
	mounts []external_models.CewMount,
	serviceFileGroups map[string]string,
	deploymentFilesDirName string,
	fileGroupMounts map[string][]fileGroupMount,
	hostPath string,
) []external_models.CewMount {
	for basePath, reference := range serviceFileGroups {
		fileMounts, ok := fileGroupMounts[reference]
		if !ok {
			continue
		}
		for _, fileMount := range fileMounts {
			mounts = append(mounts, external_models.CewMount{
				Type:     external_models.CewMountTypeBind,
				Source:   path.Join(hostPath, deploymentFilesDirName, fileMount.FileName),
				Target:   path.Join(basePath, fileMount.Path),
				ReadOnly: true,
			})
		}
	}
	return mounts
}

func getContainerDevices(
	serviceHostResources map[string]external_models.ModuleLibHostResTarget,
	userDataHostResources map[string]pkg_models.DeploymentHostResource,
	cacheHostResources map[string]external_models.HmHostResource,
) []external_models.CewDevice {
	var devices []external_models.CewDevice
	for mountPath, srvHostResource := range serviceHostResources {
		tmp, ok := userDataHostResources[srvHostResource.Ref]
		if !ok {
			continue
		}
		hostResource, ok := cacheHostResources[tmp.Id]
		if !ok || hostResource.Type == external_models.HmHostResourceTypeApp {
			continue
		}
		devices = append(devices, external_models.CewDevice{
			Source:   hostResource.Path,
			Target:   mountPath,
			ReadOnly: srvHostResource.ReadOnly,
		})
	}
	return devices
}

func newCewPorts(servicePorts []external_models.ModuleLibPort) (ports []external_models.CewPort) {
	for _, port := range servicePorts {
		p := external_models.CewPort{
			Number:   port.Number,
			Protocol: port.Protocol,
		}
		if len(port.Bindings) > 0 {
			var bindings []external_models.CewPortBinding
			for _, n := range port.Bindings {
				bindings = append(bindings, external_models.CewPortBinding{Number: n})
			}
			p.Bindings = bindings
		}
		ports = append(ports, p)
	}
	return ports
}

func newCewRunConfig(serviceRunConfig external_models.ModuleLibRunConfig) external_models.CewRunConfig {
	rc := external_models.CewRunConfig{
		RestartStrategy: external_models.CewRestartStrategyNever,
		PseudoTTY:       serviceRunConfig.PseudoTTY,
		Command:         serviceRunConfig.Command,
	}
	if serviceRunConfig.StopTimeout > 0 {
		rc.StopTimeout = &serviceRunConfig.StopTimeout // unnecessary pointer, change cew
	}
	if serviceRunConfig.StopSignal != "" {
		rc.StopSignal = &serviceRunConfig.StopSignal // unnecessary pointer, change cew
	}
	return rc
}

func configsToStrings(
	moduleConfigs external_models.ModuleLibConfigs,
	configs map[string]pkg_models.Value,
) map[string]string {
	configValues := make(map[string]string)
	for reference, config := range configs {
		if config.IsSlice {
			moduleConfig := moduleConfigs[reference]
			configValues[reference] = helper_configs.SliceValueToString(config, moduleConfig.Delimiter)
		} else {
			configValues[reference] = helper_configs.ValueToString(config)
		}
	}
	return configValues
}
