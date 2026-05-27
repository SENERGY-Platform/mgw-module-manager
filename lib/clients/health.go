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

package clients

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

type ClientHealth struct {
	client  httpClient
	baseUrl string
}

func NewClientHealth(httpClient httpClient, baseUrl string) *ClientHealth {
	return &ClientHealth{
		client:  httpClient,
		baseUrl: baseUrl,
	}
}

func appendDeploymentsHealthQuery(u string, filter models.DeploymentsHealthInfoFilter) string {
	var items []string
	if len(filter.ModuleIds) > 0 {
		items = append(items, "module_ids="+queryJoinStrings(filter.ModuleIds, ","))
	}
	if len(filter.ExclModuleIds) > 0 {
		items = append(items, "excl_module_ids="+queryJoinStrings(filter.ExclModuleIds, ","))
	}
	if filter.AuxiliaryDeployments {
		items = append(items, "auxiliary_deployments=true")
	}
	if len(filter.AuxDeploymentsOfIds) > 0 {
		items = append(items, "auxiliary_deployments_of_ids="+queryJoinStrings(filter.AuxDeploymentsOfIds, ","))
	}
	if len(filter.ExclAuxDeploymentsOfIds) > 0 {
		items = append(items, "excl_auxiliary_deployments_of_ids="+queryJoinStrings(filter.ExclAuxDeploymentsOfIds, ","))
	}
	if filter.IncludeHealthy {
		items = append(items, "include_healthy=true")
	}
	if len(items) > 0 {
		return u + "?" + strings.Join(items, "&")
	}
	return u
}

func (c *ClientHealth) DeploymentsHealth(ctx context.Context, filter models.DeploymentsHealthInfoFilter) (models.DeploymentsHealthInfo, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathDeploymentsHealthCollection))
	if err != nil {
		return models.DeploymentsHealthInfo{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appendDeploymentsHealthQuery(u, filter), nil)
	if err != nil {
		return models.DeploymentsHealthInfo{}, err
	}
	var res models.DeploymentsHealthInfo
	err = doJson(c.client, req, &res)
	if err != nil {
		return models.DeploymentsHealthInfo{}, err
	}
	return res, nil
}
