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
	"slices"
	"strings"

	helper_job "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/job"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_deployment "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployment"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) DeleteDeployments(ctx context.Context, filter models_handler_deployment.DeploymentsFilter) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	deployments, err := h.storageHdl.ReadDeployments(ctx, filter.DeploymentsFilter)
	if err != nil {
		return err
	}
	deploymentsVolumes, deploymentsContainers, err := h.getDeploymentsVolumesAndContainersFromDB(ctx, slices.Collect(maps.Keys(deployments)))
	if err != nil {
		return err
	}
	var errs []string
	for id, deployment := range deployments {
		err = h.deleteDeployment(
			ctx,
			deployment.Id,
			deployment.DirName,
			deployment.FilesDirName,
			deploymentsContainers[id],
			deploymentsVolumes[id],
		)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (h *Handler) deleteDeployment(
	ctx context.Context,
	deploymentId string,
	deploymentDirName string,
	deploymentFilesDirName string,
	containers map[string]models_handler_storage.DeploymentContainer,
	volumes map[string]models_handler_storage.DeploymentVolume,
) error {
	err := h.removeDeploymentEnvironment(ctx, deploymentId, deploymentDirName, deploymentFilesDirName, containers)
	if err != nil {
		return err
	}
	err = h.removeContainerVolumes(ctx, volumes)
	if err != nil {
		return err
	}
	err = h.removeHttpEndpoints(ctx, deploymentId)
	if err != nil {
		return err
	}
	return h.storageHdl.DeleteDeployment(ctx, deploymentId)
}

func (h *Handler) removeHttpEndpoints(ctx context.Context, deploymentId string) error {
	jobId, err := h.cmClient.RemoveEndpoints(ctx, models_external.CmEndpointFiler{Ref: deploymentId}, false)
	if err != nil {
		return err
	}
	job, err := helper_job.Await(ctx, h.cmClient, jobId, h.config.JobPollInterval)
	if err != nil {
		return err
	}
	if job.Error != nil {
		return errors.New(job.Error.Message)
	}
	return nil
}
