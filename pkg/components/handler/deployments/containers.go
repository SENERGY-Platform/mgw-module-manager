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
	"errors"
	"path"
	"strings"

	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) createContainers(
	ctx context.Context,
	module models_handler_module.Module,
	deployment extendedDeployment,
	userData userDataCollection,
	containerData containerDataCollection,
	cache cacheCollection,
) ([]models_handler_storage.DeploymentContainer, error) {
	var createdContainers []models_handler_storage.DeploymentContainer
	var errs []string
	for reference, service := range module.Services {
		depContainer := deployment.Containers[reference]
		cewContainer, err := getCewContainer(
			service,
			depContainer,
			getContainerEnvVariables(service, deployment, userData, containerData, cache),
			h.getContainerMounts(service, deployment, userData, containerData, cache),
			getContainerDevices(service, userData, cache),
		)
		id, err := h.cewClient.CreateContainer(ctx, cewContainer)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		depContainer.Id = id
		createdContainers = append(createdContainers, depContainer)
	}

	if len(errs) > 0 {
		return createdContainers, errors.New(strings.Join(errs, "\n"))
	}
	return createdContainers, nil
}

func getCewContainer(
	service models_external.ModuleService,
	container models_handler_storage.DeploymentContainer,
	envVariables map[string]string,
	mounts []models_external.CewMount,
	devices []models_external.CewDevice,
) (models_external.Container, error) {
	name, err := helper_naming.NewContainerName("dep")
	if err != nil {
		return models_external.Container{}, err
	}
	return models_external.Container{
		Name:    name,
		Image:   service.Image,
		EnvVars: envVariables,
		Labels: map[string]string{
			constants.LabelCoreId:           helper_naming.CoreId,
			constants.LabelManagerId:        helper_naming.ManagerId,
			constants.LabelDeploymentId:     container.DeploymentId,
			constants.LabelServiceReference: container.Reference,
		},
		Mounts:            mounts,
		Devices:           devices,
		DeviceCGroupRules: service.DeviceCGroupRules,
		Ports:             newCewPorts(service),
		Networks: []models_external.CewContainerNetwork{
			{
				Name:        helper_naming.ModuleContainerNetwork,
				DomainNames: []string{container.Alias, name},
			},
		},
		RunConfig: newCewRunConfig(service),
	}, nil
}

func getContainerEnvVariables(
	service models_external.ModuleService,
	deployment extendedDeployment,
	userData userDataCollection,
	containerData containerDataCollection,
	cache cacheCollection,
) map[string]string {
	envVariables := make(map[string]string)
	setConfigEnvVariables(envVariables, service, containerData)
	setSecretValueEnvVariables(envVariables, service, userData, cache)
	setInternalDependencyEnvVariables(envVariables, service, deployment)
	setExternalDependencyEnvVariables(envVariables, service, cache)
	envVariables[constants.EnvVariableDeploymentId] = deployment.Id
	return envVariables
}

func (h *Handler) getContainerMounts(
	service models_external.ModuleService,
	deployment extendedDeployment,
	userData userDataCollection,
	containerData containerDataCollection,
	cache cacheCollection,
) []models_external.CewMount {
	var mounts []models_external.CewMount
	mounts = appendIncludeMounts(mounts, service, deployment, h.config.WorkDirPath)
	mounts = appendTmpfsMounts(mounts, service)
	mounts = appendVolumeMounts(mounts, service, deployment)
	mounts = appendApplicationMounts(mounts, service, userData, cache)
	mounts = appendSecretMounts(mounts, service, userData, containerData, h.config.HostSecretsPath)
	mounts = appendFileMounts(mounts, service, deployment, containerData, h.config.WorkDirPath)
	mounts = appendFileGroupMounts(mounts, service, deployment, containerData, h.config.WorkDirPath)
	return mounts
}

func setConfigEnvVariables(
	envVariables map[string]string,
	service models_external.ModuleService,
	containerData containerDataCollection,
) {
	for envVarName, reference := range service.Configs {
		value, ok := containerData.Configs[reference]
		if ok {
			envVariables[envVarName] = value
		}
	}
}

func setSecretValueEnvVariables(
	envVariables map[string]string,
	service models_external.ModuleService,
	userData userDataCollection,
	cache cacheCollection,
) {
	for envVarName, target := range service.SecretVars {
		selectedSecret, ok := userData.Secrets[target.Ref]
		if !ok {
			continue
		}
		valueVariant, ok := cache.SecretValues[selectedSecret.Id+target.Item]
		if !ok {
			continue
		}
		envVariables[envVarName] = valueVariant.Value
	}
}

func setInternalDependencyEnvVariables(
	envVariables map[string]string,
	service models_external.ModuleService,
	deployment extendedDeployment,
) {
	for envVarName, target := range service.SrvReferences {
		container, ok := deployment.Containers[target.Ref]
		if ok {
			envVariables[envVarName] = target.FillTemplate(container.Alias)
		}
	}
}

func setExternalDependencyEnvVariables(
	envVariables map[string]string,
	service models_external.ModuleService,
	cache cacheCollection,
) {
	for envVarName, target := range service.ExtDependencies {
		containers, ok := cache.ContainerAliases[target.ID]
		if !ok {
			continue
		}
		alias, ok := containers[target.Service]
		if ok {
			envVariables[envVarName] = target.FillTemplate(alias)
		}
	}
}

func appendIncludeMounts(
	mounts []models_external.CewMount,
	service models_external.ModuleService,
	deployment extendedDeployment,
	hostPath string,
) []models_external.CewMount {
	for mountPath, include := range service.BindMounts {
		mounts = append(mounts, cew_model.Mount{
			Type:     models_external.CewMountTypeBind,
			Source:   path.Join(hostPath, deployment.DirName, include.Source),
			Target:   mountPath,
			ReadOnly: include.ReadOnly,
		})
	}
	return mounts
}

func appendTmpfsMounts(
	mounts []models_external.CewMount,
	service models_external.ModuleService,
) []models_external.CewMount {
	for mountPath, tmpfs := range service.Tmpfs {
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
	service models_external.ModuleService,
	deployment extendedDeployment,
) []models_external.CewMount {
	for mountPath, name := range service.Volumes {
		volume, ok := deployment.Volumes[name]
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
	service models_external.ModuleService,
	userData userDataCollection,
	cache cacheCollection,
) []models_external.CewMount {
	for mountPath, srvHostResource := range service.HostResources {
		tmp, ok := userData.HostResources[srvHostResource.Ref]
		if !ok {
			continue
		}
		hostResource, ok := cache.HostResources[tmp.Id]
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
	service models_external.ModuleService,
	userData userDataCollection,
	containerData containerDataCollection,
	hostPath string,
) []models_external.CewMount {
	for mountPath, target := range service.SecretMounts {
		selectedSecret, ok := userData.Secrets[target.Ref]
		if !ok {
			continue
		}
		pathVariant, ok := containerData.SecretMounts[selectedSecret.Id+target.Item]
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
	service models_external.ModuleService,
	deployment extendedDeployment,
	containerData containerDataCollection,
	hostPath string,
) []models_external.CewMount {
	for mountPoint, reference := range service.Files {
		fileName, ok := containerData.FileMounts[reference]
		if ok {
			mounts = append(mounts, models_external.CewMount{
				Type:     models_external.CewMountTypeBind,
				Source:   path.Join(hostPath, deployment.FilesDirName, fileName),
				Target:   mountPoint,
				ReadOnly: true,
			})
		}
	}
	return mounts
}

func appendFileGroupMounts(
	mounts []models_external.CewMount,
	service models_external.ModuleService,
	deployment extendedDeployment,
	containerData containerDataCollection,
	hostPath string,
) []models_external.CewMount {
	for basePath, reference := range service.FileGroups {
		fileMounts, ok := containerData.FileGroupMounts[reference]
		if !ok {
			continue
		}
		for _, fileMount := range fileMounts {
			mounts = append(mounts, models_external.CewMount{
				Type:     models_external.CewMountTypeBind,
				Source:   path.Join(hostPath, deployment.FilesDirName, fileMount.FileName),
				Target:   path.Join(basePath, fileMount.Path),
				ReadOnly: true,
			})
		}
	}
	return mounts
}

func getContainerDevices(
	service models_external.ModuleService,
	userData userDataCollection,
	cache cacheCollection,
) []models_external.CewDevice {
	var devices []models_external.CewDevice
	for mountPath, srvHostResource := range service.HostResources {
		tmp, ok := userData.HostResources[srvHostResource.Ref]
		if !ok {
			continue
		}
		hostResource, ok := cache.HostResources[tmp.Id]
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

func newCewPorts(service models_external.ModuleService) (ports []models_external.CewPort) {
	for _, port := range service.Ports {
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

func newCewRunConfig(service models_external.ModuleService) models_external.CewRunConfig {
	rc := models_external.CewRunConfig{
		RestartStrategy: models_external.CewRestartStrategyNever, // restarts should be handled by module-manager
		Retries:         nil,                                     // sollte von health handler verwendet werden?
		RemoveAfterRun:  false,                                   // wird das benutzt?
		PseudoTTY:       service.RunConfig.PseudoTTY,
		Command:         service.RunConfig.Command,
	}
	if service.RunConfig.StopTimeout > 0 {
		rc.StopTimeout = &service.RunConfig.StopTimeout // pointer unnötig, cew anpassen
	}
	if service.RunConfig.StopSignal != "" {
		rc.StopSignal = &service.RunConfig.StopSignal // pointer unnötig, cew anpassen
	}
	return rc
}
