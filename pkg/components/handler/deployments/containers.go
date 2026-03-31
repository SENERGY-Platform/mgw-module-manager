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

package handler_deployments

import (
	"context"
	"errors"
	"path"
	"strings"

	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/configs"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/job"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/config"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

func (h *Handler) createContainers(
	ctx context.Context,
	moduleConfigs models_external.ModuleLibConfigs,
	moduleServices map[string]models_external.ModuleLibService,
	deploymentId string,
	deploymentDirName string,
	deploymentFilesDirName string,
	userDataSecrets map[string]models_handler_database.DeploymentSecret,
	userDataHostResources map[string]models_handler_database.DeploymentHostResource,
	containers map[string]models_handler_database.DeploymentContainer,
	volumes map[string]models_handler_database.DeploymentVolume,
	configs map[string]models_config.Config,
	bindMounts bindMountDataCollection,
	cacheSecretValues map[string]models_external.SecretValueVariant,
	cacheDeployments map[string]deploymentsCacheItem,
	cacheHostResources map[string]models_external.HostResource,
) error {
	var errs []string
	for reference, service := range moduleServices {
		envVariables := make(map[string]string)
		setConfigEnvVariables(envVariables, service.Configs, helper_configs.ConfigsToStrings(moduleConfigs, configs))
		setSecretValueEnvVariables(envVariables, service.SecretVars, userDataSecrets, cacheSecretValues)
		setInternalDependencyEnvVariables(envVariables, service.SrvReferences, containers)
		setExternalDependencyEnvVariables(envVariables, service.ExtDependencies, cacheDeployments)
		envVariables[models_constants.EnvVariableDeploymentId] = deploymentId
		var mounts []models_external.CewMount
		mounts = appendIncludeMounts(mounts, service.BindMounts, deploymentDirName, h.config.WorkDirPath)
		mounts = appendTmpfsMounts(mounts, service.Tmpfs)
		mounts = appendVolumeMounts(mounts, service.Volumes, volumes)
		mounts = appendApplicationMounts(mounts, service.HostResources, userDataHostResources, cacheHostResources)
		mounts = appendSecretMounts(mounts, service.SecretMounts, userDataSecrets, bindMounts.Secrets, h.config.HostSecretsPath)
		mounts = appendFileMounts(mounts, service.Files, deploymentFilesDirName, bindMounts.Files, h.config.WorkDirPath)
		mounts = appendFileGroupMounts(mounts, service.FileGroups, deploymentFilesDirName, bindMounts.FileGroups, h.config.WorkDirPath)
		storageContainer := containers[reference]
		cewContainer, err := getCewContainer(
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
		_, err = h.containerEngineWrapperClient.CreateContainer(ctx, cewContainer)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (h *Handler) removeContainers(
	ctx context.Context,
	deploymentContainers map[string]models_handler_database.DeploymentContainer,
) error {
	var errs []string
	for _, container := range deploymentContainers {
		err := h.removeContainer(ctx, container.Name)
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
	err := h.containerEngineWrapperClient.RemoveContainer(ctx, containerId, true)
	if err != nil {
		var notFoundErr *models_external.CEWNotFoundErr
		if !errors.As(err, &notFoundErr) {
			return err
		}
	}
	return nil
}

func (h *Handler) startContainers(
	ctx context.Context,
	deploymentContainers map[string]models_handler_database.DeploymentContainer,
) error {
	var errs []string
	for _, container := range deploymentContainers {
		err := h.containerEngineWrapperClient.StartContainer(ctx, container.Name)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func (h *Handler) stopContainers(
	ctx context.Context,
	deploymentContainers map[string]models_handler_database.DeploymentContainer,
) error {
	var errs []string
	for _, container := range deploymentContainers {
		err := h.stopContainer(ctx, container.Name)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func (h *Handler) stopContainer(ctx context.Context, containerId string) error {
	jobId, err := h.containerEngineWrapperClient.StopContainer(ctx, containerId)
	if err != nil {
		return err
	}
	job, err := helper_job.Await(ctx, h.containerEngineWrapperClient, jobId, h.config.JobPollInterval)
	if err != nil {
		return err
	}
	if job.Error != nil {
		return errors.New(job.Error.Message)
	}
	return nil
}

func (h *Handler) restartContainers(
	ctx context.Context,
	deploymentContainers map[string]models_handler_database.DeploymentContainer,
) error {
	var errs []string
	for _, container := range deploymentContainers {
		err := h.restartContainer(ctx, container.Name)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func (h *Handler) restartContainer(ctx context.Context, containerId string) error {
	jobId, err := h.containerEngineWrapperClient.RestartContainer(ctx, containerId)
	if err != nil {
		return err
	}
	job, err := helper_job.Await(ctx, h.containerEngineWrapperClient, jobId, h.config.JobPollInterval)
	if err != nil {
		return err
	}
	if job.Error != nil {
		return errors.New(job.Error.Message)
	}
	return nil
}

func getCewContainer(
	serviceImage string,
	serviceDeviceCGroupRules []string,
	servicePorts []models_external.ModuleLibPort,
	serviceRunConfig models_external.ModuleLibRunConfig,
	serviceReference string,
	deploymentId string,
	containerAlias string,
	containerName string,
	envVariables map[string]string,
	mounts []models_external.CewMount,
	devices []models_external.CewDevice,
) (models_external.Container, error) {
	return models_external.Container{
		Name:    containerName,
		Image:   serviceImage,
		EnvVars: envVariables,
		Labels: map[string]string{
			models_constants.LabelCoreId:           helper_naming.CoreId,
			models_constants.LabelManagerId:        helper_naming.ManagerId,
			models_constants.LabelDeploymentId:     deploymentId,
			models_constants.LabelServiceReference: serviceReference,
		},
		Mounts:            mounts,
		Devices:           devices,
		DeviceCGroupRules: serviceDeviceCGroupRules,
		Ports:             newCewPorts(servicePorts),
		Networks: []models_external.CewContainerNetwork{
			{
				Name:        helper_naming.ModuleContainerNetwork,
				DomainNames: []string{containerAlias, containerName},
			},
		},
		RunConfig: newCewRunConfig(serviceRunConfig),
	}, nil
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
	userDataSecrets map[string]models_handler_database.DeploymentSecret,
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
	deploymentContainers map[string]models_handler_database.DeploymentContainer,
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
	deploymentVolumes map[string]models_handler_database.DeploymentVolume,
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
	userDataHostResources map[string]models_handler_database.DeploymentHostResource,
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
	userDataSecrets map[string]models_handler_database.DeploymentSecret,
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
	userDataHostResources map[string]models_handler_database.DeploymentHostResource,
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
