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
	"errors"
	"strings"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/containers"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/aux_deployments"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

func (h *Handler) DeleteAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models_handler_aux_deployments.AuxiliaryDeploymentsFilter,
) ([]string, error) {
	mu := h.mutexes.Get(deploymentId)
	mu.Lock()
	defer mu.Unlock()
	auxDeployments, err := h.databaseHandler.ReadAuxiliaryDeployments(ctx, deploymentId, filter.AuxiliaryDeploymentsFilter)
	if err != nil {
		return nil, err
	}
	if filter.State != "" {
		cewContainers, err := h.getCewContainers(ctx, auxDeployments)
		if err != nil {
			return nil, err
		}
		tmp := make(map[string]models_handler_database.AuxiliaryDeployment)
		for id, auxDep := range auxDeployments {
			cewContainer := cewContainers[auxDep.Container.Name]
			if cewContainer.State == filter.State {
				tmp[id] = auxDep
			}
		}
		auxDeployments = tmp
	}
	var deleted []string
	var errs []string
	for id, auxDep := range auxDeployments {
		err = helper_containers.Remove(ctx, h.containerEngineWrapperClient, auxDep.Container.Name)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		deleted = append(deleted, id)
	}
	err = h.databaseHandler.DeleteAuxiliaryDeployments(ctx, deleted)
	if err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return deleted, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return deleted, nil
}

func (h *Handler) deleteAuxiliaryDeployment(ctx context.Context, id, containerName string) error {
	err := helper_containers.Remove(ctx, h.containerEngineWrapperClient, containerName)
	if err != nil {
		return err
	}
	err = h.databaseHandler.DeleteAuxiliaryDeployment(ctx, id)
	if err != nil {
		return err
	}
	return nil
}
