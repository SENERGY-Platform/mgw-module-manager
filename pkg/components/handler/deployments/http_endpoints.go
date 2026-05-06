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

	helper_job "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/job"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) createHttpEndpoints(
	ctx context.Context,
	moduleServices map[string]external_models.ModuleLibService,
	moduleId string,
	deploymentContainers map[string]pkg_models.DeploymentContainerBase,
) error {
	var endpoints []external_models.CmEndpointBase
	for reference, service := range moduleServices {
		container := deploymentContainers[reference]
		for externalPath, endpoint := range service.HttpEndpoints {
			endpoints = append(endpoints, newCmEndpointBase(container.Reference, container.Alias, endpoint, moduleId, externalPath))
		}
	}
	if len(endpoints) == 0 {
		return nil
	}
	jobId, err := h.coreManagerClient.SetEndpoints(ctx, endpoints)
	if err != nil {
		return err
	}
	job, err := helper_job.Await(ctx, h.coreManagerClient, jobId, h.config.JobPollInterval)
	if err != nil {
		return err
	}
	if job.Error != nil {
		return errors.New(job.Error.Message)
	}
	return nil
}

func newCmEndpointBase(
	containerReference string,
	containerAlias string,
	serviceEndpoint external_models.ModuleLibHttpEndpoint,
	moduleId,
	externalPath string,
) external_models.CmEndpointBase {
	return external_models.CmEndpointBase{
		Ref:     containerReference,
		Host:    containerAlias,
		Port:    &serviceEndpoint.Port, // TODO unnecessary pointer
		IntPath: serviceEndpoint.Path,
		ExtPath: externalPath,
		ProxyConf: external_models.CmProxyConfig{
			Headers:     serviceEndpoint.ProxyConf.Headers,
			WebSocket:   serviceEndpoint.ProxyConf.WebSocket,
			ReadTimeout: serviceEndpoint.ProxyConf.ReadTimeout,
		},
		StringSub: external_models.CmStringSub{
			ReplaceOnce: serviceEndpoint.StringSub.ReplaceOnce,
			MimeTypes:   serviceEndpoint.StringSub.MimeTypes,
			Filters:     serviceEndpoint.StringSub.Filters,
		},
		Labels: map[string]string{
			constants.LabelHttpEndpointModuleId:         moduleId,
			constants.LabelHttpEndpointServiceReference: containerReference,
		},
	}
}
