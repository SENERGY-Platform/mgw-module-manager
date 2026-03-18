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
	"maps"
	"os"
	"path"
	"slices"
	"strings"

	models_handler_deployment "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployment"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) UpdateDeployments(
	ctx context.Context,
	selectedModules map[string]models_handler_module.Module,
	userInputs map[string]models_handler_deployment.UserInput,
) error {
	currentDeployments, err := h.getDeploymentsByModuleIds(ctx, slices.Collect(maps.Keys(selectedModules)))
	if err != nil {
		return err
	}
	cache, err := initCache(selectedModules)
	if err != nil {
		return err
	}
	var errs []string
	for moduleId, module := range selectedModules {
		userInput := userInputs[moduleId]
		currentDeployment := currentDeployments[moduleId]
		newDeployment, err := getExtendedDeployment(module, userInput, cache)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		userData, mergedConfigs, mergedFiles, err := getDeploymentData(module, newDeployment, userInput, cache)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.removeContainers(ctx, currentDeployment)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.clearContainerEnvironment(ctx, currentDeployment)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.storageHdl.UpdateDeployment(
			ctx,
			newDeployment.Deployment,
			slices.Collect(maps.Values(userData.HostResources)),
			slices.Collect(maps.Values(userData.Secrets)),
			slices.Collect(maps.Values(userData.Configs)),
			slices.Collect(maps.Values(userData.GlobalConfigs)),
			slices.Collect(maps.Values(userData.Files)),
			slices.Collect(maps.Values(userData.FileGroups)),
			slices.Collect(maps.Values(newDeployment.Volumes)),
			slices.Collect(maps.Values(newDeployment.Containers)),
		)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		err = h.updateCaches(ctx, module, userData, cache)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		containerData, err := h.ensureContainerEnvironment(ctx, module, newDeployment, userData, mergedConfigs, mergedFiles)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		// TODO "mount secrets" must be "unloaded" if one of the following steps fails
		err = h.createHttpEndpoints(ctx, module, newDeployment)
		if err != nil {
			errs = append(errs, err.Error())
		}
		createdContainers, err := h.createContainers(ctx, module, newDeployment, userData, containerData, cache)
		if err != nil {
			errs = append(errs, err.Error())
		}
		err = h.storageHdl.UpdateDeploymentContainerIds(ctx, createdContainers)
		if err != nil {
			// TODO how to handle already created containers?
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (h *Handler) getDeploymentsByModuleIds(
	ctx context.Context,
	moduleIds []string,
) (map[string]extendedDeployment, error) {
	deployments, err := h.storageHdl.ReadDeployments(ctx, models_handler_storage.DeploymentsFilter{
		ModuleIds: moduleIds,
	})
	if err != nil {
		return nil, err
	}
	deploymentsContainers, err := h.storageHdl.ReadDeploymentsContainers(ctx, slices.Collect(maps.Keys(deployments)))
	if err != nil {
		return nil, err
	}
	deploymentsMap := make(map[string]extendedDeployment)
	for id, deployment := range deployments {
		containers := make(map[string]models_handler_storage.DeploymentContainer)
		for _, container := range deploymentsContainers[id] {
			containers[container.Reference] = container
		}
		deploymentsMap[deployment.ModuleId] = extendedDeployment{
			Deployment: deployment,
			Containers: containers,
		}
	}
	return deploymentsMap, nil
}

func (h *Handler) clearContainerEnvironment(ctx context.Context, deployment extendedDeployment) error {
	err := h.removeSecretMounts(ctx, deployment)
	if err != nil {
		return err
	}
	err = h.removeDeploymentDir(deployment)
	if err != nil {
		return err
	}
	return h.removeFilesDir(deployment)
}

func (h *Handler) removeDeploymentDir(deployment extendedDeployment) error {
	return os.RemoveAll(path.Join(h.config.WorkDirPath, deployment.DirName))
}
