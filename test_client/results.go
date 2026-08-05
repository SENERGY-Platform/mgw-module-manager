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
	"context"
	"net/http"
	"net/url"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

func (c *Client) GetDeploymentsJobResult(ctx context.Context, jobId string) (lib_models.DeploymentJobResult, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(lib_constants.HttpPathDeploymentResultResource, jobId))
	if err != nil {
		return lib_models.DeploymentJobResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return lib_models.DeploymentJobResult{}, err
	}
	var res lib_models.DeploymentJobResult
	err = doJson(client, req, &res)
	if err != nil {
		return lib_models.DeploymentJobResult{}, err
	}
	return res, nil
}

func (c *Client) GetUpdateDeploymentsJobResult(ctx context.Context, jobId string) (lib_models.DeploymentUpdateJobResult, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(lib_constants.HttpPathUpdateDeploymentResultResource, jobId))
	if err != nil {
		return lib_models.DeploymentUpdateJobResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return lib_models.DeploymentUpdateJobResult{}, err
	}
	var res lib_models.DeploymentUpdateJobResult
	err = doJson(client, req, &res)
	if err != nil {
		return lib_models.DeploymentUpdateJobResult{}, err
	}
	return res, nil
}

func (c *Client) GetDeleteDeploymentsJobResult(ctx context.Context, jobId string) (lib_models.DeploymentDeleteJobResult, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(lib_constants.HttpPathDeleteDeploymentResultResource, jobId))
	if err != nil {
		return lib_models.DeploymentDeleteJobResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return lib_models.DeploymentDeleteJobResult{}, err
	}
	var res lib_models.DeploymentDeleteJobResult
	err = doJson(client, req, &res)
	if err != nil {
		return lib_models.DeploymentDeleteJobResult{}, err
	}
	return res, nil
}

func (c *Client) GetModuleChangeJobResult(ctx context.Context, jobId string) (lib_models.ModulesChangeJobResult, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(lib_constants.HttpPathChangeModulesResultResource, jobId))
	if err != nil {
		return lib_models.ModulesChangeJobResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return lib_models.ModulesChangeJobResult{}, err
	}
	var res lib_models.ModulesChangeJobResult
	err = doJson(client, req, &res)
	if err != nil {
		return lib_models.ModulesChangeJobResult{}, err
	}
	return res, nil
}

func (c *Client) GetRefreshRepositoriesJobResult(ctx context.Context, jobId string) (lib_models.RepositoryJobResult, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(lib_constants.HttpPathRefreshRepositoriesResultResource, jobId))
	if err != nil {
		return lib_models.RepositoryJobResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return lib_models.RepositoryJobResult{}, err
	}
	var res lib_models.RepositoryJobResult
	err = doJson(client, req, &res)
	if err != nil {
		return lib_models.RepositoryJobResult{}, err
	}
	return res, nil
}
