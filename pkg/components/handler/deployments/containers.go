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
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) createContainers(
	ctx context.Context,
	moduleServices map[string]models_external.ModuleLibService,
	deployment extendedDeployment,
	containerEnvironmentData containerEnvironmentDataCollection,
	cache cacheCollection,
) ([]models_handler_storage.DeploymentContainer, error) {
	var createdContainers []models_handler_storage.DeploymentContainer
	var errs []string
	for reference, service := range moduleServices {
		depContainer := deployment.Containers[reference]
		cewContainer, err := getCewContainer(
			service,
			depContainer,
			getContainerEnvVariables(service, deployment, containerEnvironmentData.Configs, cache),
			h.getContainerMounts(service, deployment, containerEnvironmentData, cache.HostResources),
			getContainerDevices(service.HostResources, deployment.UserData.HostResources, cache.HostResources),
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

func (h *Handler) removeContainers(
	ctx context.Context,
	deploymentContainers map[string]models_handler_storage.DeploymentContainer,
) error {
	var errs []string
	for _, container := range deploymentContainers {
		err := h.removeContainer(ctx, container.Id)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func (h *Handler) removeContainer(ctx context.Context, containerId string) error {
	err := h.cewClient.RemoveContainer(ctx, containerId, true)
	if err != nil {
		var notFoundErr *models_external.CEWNotFoundErr
		if !errors.As(err, &notFoundErr) {
			return err
		}
	}
	return nil
}

func getCewContainer(
	service models_external.ModuleLibService,
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
		Ports:             newCewPorts(service.Ports),
		Networks: []models_external.CewContainerNetwork{
			{
				Name:        helper_naming.ModuleContainerNetwork,
				DomainNames: []string{container.Alias, name},
			},
		},
		RunConfig: newCewRunConfig(service.RunConfig),
	}, nil
}

func getContainerEnvVariables(
	service models_external.ModuleLibService,
	deployment extendedDeployment,
	configs map[string]string,
	cache cacheCollection,
) map[string]string {
	envVariables := make(map[string]string)
	setConfigEnvVariables(envVariables, service.Configs, configs)
	setSecretValueEnvVariables(envVariables, service.SecretVars, deployment.UserData.Secrets, cache.SecretValues)
	setInternalDependencyEnvVariables(envVariables, service.SrvReferences, deployment.Containers)
	setExternalDependencyEnvVariables(envVariables, service.ExtDependencies, cache.ContainerAliases)
	envVariables[constants.EnvVariableDeploymentId] = deployment.Id
	return envVariables
}

func (h *Handler) getContainerMounts(
	service models_external.ModuleLibService,
	deployment extendedDeployment,
	containerEnvironmentData containerEnvironmentDataCollection,
	cacheHostResources map[string]models_external.HostResource,
) []models_external.CewMount {
	var mounts []models_external.CewMount
	mounts = appendIncludeMounts(mounts, service.BindMounts, deployment.DirName, h.config.WorkDirPath)
	mounts = appendTmpfsMounts(mounts, service.Tmpfs)
	mounts = appendVolumeMounts(mounts, service.Volumes, deployment.Volumes)
	mounts = appendApplicationMounts(mounts, service.HostResources, deployment.UserData.HostResources, cacheHostResources)
	mounts = appendSecretMounts(mounts, service.SecretMounts, deployment.UserData.Secrets, containerEnvironmentData.SecretMounts, h.config.HostSecretsPath)
	mounts = appendFileMounts(mounts, service.Files, deployment.FilesDirName, containerEnvironmentData.FileMounts, h.config.WorkDirPath)
	mounts = appendFileGroupMounts(mounts, service.FileGroups, deployment.FilesDirName, containerEnvironmentData.FileGroupMounts, h.config.WorkDirPath)
	return mounts
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
	serviceSecretVars map[string]models_external.ModuleLibSecretTarget,
	userDataSecrets map[string]models_handler_storage.DeploymentSecret,
	cacheSecretValues map[string]models_external.SecretValueVariant,
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
	serviceReferences map[string]models_external.ModuleLibSrvRefTarget,
	deploymentContainers map[string]models_handler_storage.DeploymentContainer,
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
	cacheContainerAliases map[string]map[string]string,
) {
	for envVarName, target := range serviceExtDependencies {
		containers, ok := cacheContainerAliases[target.ID]
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
	serviceVolumes map[string]string,
	deploymentVolumes map[string]models_handler_storage.DeploymentVolume,
) []models_external.CewMount {
	for mountPath, name := range serviceVolumes {
		volume, ok := deploymentVolumes[name]
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
	serviceHostResources map[string]models_external.ModuleLibHostResTarget,
	userDataHostResources map[string]models_handler_storage.DeploymentHostResource,
	cacheHostResources map[string]models_external.HostResource,
) []models_external.CewMount {
	for mountPath, srvHostResource := range serviceHostResources {
		tmp, ok := userDataHostResources[srvHostResource.Ref]
		if !ok {
			continue
		}
		hostResource, ok := cacheHostResources[tmp.Id]
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
	serviceSecretMounts map[string]models_external.ModuleLibSecretTarget,
	userDataSecrets map[string]models_handler_storage.DeploymentSecret,
	secretMounts map[string]models_external.SecretPathVariant,
	hostPath string,
) []models_external.CewMount {
	for mountPath, target := range serviceSecretMounts {
		selectedSecret, ok := userDataSecrets[target.Ref]
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
	deploymentFilesDirName string,
	fileMounts map[string]string,
	hostPath string,
) []models_external.CewMount {
	for mountPoint, reference := range serviceFiles {
		fileName, ok := fileMounts[reference]
		if ok {
			mounts = append(mounts, models_external.CewMount{
				Type:     models_external.CewMountTypeBind,
				Source:   path.Join(hostPath, deploymentFilesDirName, fileName),
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
	deploymentFilesDirName string,
	fileGroupMounts map[string][]fileGroupMount,
	hostPath string,
) []models_external.CewMount {
	for basePath, reference := range serviceFileGroups {
		fileMounts, ok := fileGroupMounts[reference]
		if !ok {
			continue
		}
		for _, fileMount := range fileMounts {
			mounts = append(mounts, models_external.CewMount{
				Type:     models_external.CewMountTypeBind,
				Source:   path.Join(hostPath, deploymentFilesDirName, fileMount.FileName),
				Target:   path.Join(basePath, fileMount.Path),
				ReadOnly: true,
			})
		}
	}
	return mounts
}

func getContainerDevices(
	serviceHostResources map[string]models_external.ModuleLibHostResTarget,
	userDataHostResources map[string]models_handler_storage.DeploymentHostResource,
	cacheHostResources map[string]models_external.HostResource,
) []models_external.CewDevice {
	var devices []models_external.CewDevice
	for mountPath, srvHostResource := range serviceHostResources {
		tmp, ok := userDataHostResources[srvHostResource.Ref]
		if !ok {
			continue
		}
		hostResource, ok := cacheHostResources[tmp.Id]
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

func newCewPorts(servicePorts []models_external.ModuleLibPort) (ports []models_external.CewPort) {
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

func newCewRunConfig(serviceRunConfig models_external.ModuleLibRunConfig) models_external.CewRunConfig {
	rc := models_external.CewRunConfig{
		RestartStrategy: models_external.CewRestartStrategyNever, // restarts should be handled by module-manager
		Retries:         nil,                                     // sollte von health handler verwendet werden?
		RemoveAfterRun:  false,                                   // wird das benutzt?
		PseudoTTY:       serviceRunConfig.PseudoTTY,
		Command:         serviceRunConfig.Command,
	}
	if serviceRunConfig.StopTimeout > 0 {
		rc.StopTimeout = &serviceRunConfig.StopTimeout // pointer unnötig, cew anpassen
	}
	if serviceRunConfig.StopSignal != "" {
		rc.StopSignal = &serviceRunConfig.StopSignal // pointer unnötig, cew anpassen
	}
	return rc
}
