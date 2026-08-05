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

package test_client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

func (c *Client) GetDeploymentRequest(ctx context.Context, moduleIds []string) ([]lib_models.Module, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathDeploymentRequestResource))
	if err != nil {
		return nil, err
	}
	if len(moduleIds) > 0 {
		u += "?module_ids=" + queryJoinStrings(moduleIds)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var res []lib_models.Module
	err = doJson(client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) CreateDeployments(ctx context.Context, userInputs []lib_models.DeploymentUserInput) (lib_models.Job, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathDeploymentsCollection))
	if err != nil {
		return lib_models.Job{}, err
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(userInputs)
	if err != nil {
		return lib_models.Job{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, buffer)
	if err != nil {
		return lib_models.Job{}, err
	}
	var res lib_models.Job
	err = doJson(client, req, &res)
	if err != nil {
		return lib_models.Job{}, err
	}
	return res, nil
}

func (c *Client) UpdateDeployments(ctx context.Context, userInputs []lib_models.DeploymentUserInput) (lib_models.Job, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathDeploymentsCollection))
	if err != nil {
		return lib_models.Job{}, err
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(userInputs)
	if err != nil {
		return lib_models.Job{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, buffer)
	if err != nil {
		return lib_models.Job{}, err
	}
	var res lib_models.Job
	err = doJson(client, req, &res)
	if err != nil {
		return lib_models.Job{}, err
	}
	return res, nil
}

func (c *Client) RecreateDeployments(ctx context.Context, moduleIds []string) (lib_models.Job, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathRecreateDeployments))
	if err != nil {
		return lib_models.Job{}, err
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(moduleIds)
	if err != nil {
		return lib_models.Job{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, buffer)
	if err != nil {
		return lib_models.Job{}, err
	}
	var res lib_models.Job
	err = doJson(client, req, &res)
	if err != nil {
		return lib_models.Job{}, err
	}
	return res, nil
}

func appendDeleteDeploymentsQuery(u string, moduleIds []string, allowAll bool) string {
	var items []string
	if len(moduleIds) > 0 {
		items = append(items, "module_ids="+queryJoinStrings(moduleIds))
	}
	if allowAll {
		items = append(items, "allow_all=true")
	}
	if len(items) > 0 {
		return u + "?" + strings.Join(items, "&")
	}
	return u
}

func (c *Client) DeleteDeployments(ctx context.Context, moduleIds []string, allowAll bool) (lib_models.Job, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathDeploymentsCollection))
	if err != nil {
		return lib_models.Job{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, appendDeleteDeploymentsQuery(u, moduleIds, allowAll), nil)
	if err != nil {
		return lib_models.Job{}, err
	}
	var res lib_models.Job
	err = doJson(client, req, &res)
	if err != nil {
		return lib_models.Job{}, err
	}
	return res, nil
}

func (c *Client) EnableDeployments(ctx context.Context, moduleIds []string) ([]string, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathEnableDeployments))
	if err != nil {
		return nil, err
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(moduleIds)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, buffer)
	if err != nil {
		return nil, err
	}
	var res []string
	err = doJson(client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) DisableDeployments(ctx context.Context, moduleIds []string) ([]string, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathDisableDeployments))
	if err != nil {
		return nil, err
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(moduleIds)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, buffer)
	if err != nil {
		return nil, err
	}
	var res []string
	err = doJson(client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func appendDeploymentsHealthFilterQuery(u string, filter lib_models.DeploymentsHealthInfoFilter) string {
	var items []string
	if len(filter.ModuleIds) > 0 {
		items = append(items, "module_ids="+queryJoinStrings(filter.ModuleIds))
	}
	if len(filter.ExclModuleIds) > 0 {
		items = append(items, "excl_module_ids="+queryJoinStrings(filter.ExclModuleIds))
	}
	if filter.AuxiliaryDeployments {
		items = append(items, "auxiliary_deployments=true")
	}
	if len(filter.AuxDeploymentsOfIds) > 0 {
		items = append(items, "auxiliary_deployments_of_ids="+queryJoinStrings(filter.AuxDeploymentsOfIds))
	}
	if len(filter.ExclAuxDeploymentsOfIds) > 0 {
		items = append(items, "excl_auxiliary_deployments_of_ids="+queryJoinStrings(filter.ExclAuxDeploymentsOfIds))
	}
	if filter.IncludeHealthy {
		items = append(items, "include_healthy=true")
	}
	if len(items) > 0 {
		return u + "?" + strings.Join(items, "&")
	}
	return u
}

func (c *Client) DeploymentsHealth(ctx context.Context, filter lib_models.DeploymentsHealthInfoFilter) (lib_models.DeploymentsHealthInfo, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathDeploymentsHealthCollection))
	if err != nil {
		return lib_models.DeploymentsHealthInfo{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appendDeploymentsHealthFilterQuery(u, filter), nil)
	if err != nil {
		return lib_models.DeploymentsHealthInfo{}, err
	}
	var res lib_models.DeploymentsHealthInfo
	err = doJson(client, req, &res)
	if err != nil {
		return lib_models.DeploymentsHealthInfo{}, err
	}
	return res, nil
}
