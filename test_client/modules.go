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
	"strconv"
	"strings"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

func appendModulesFilterQuery(u string, filter lib_models.ModulesFilter) string {
	var items []string
	if len(filter.Ids) > 0 {
		items = append(items, "ids="+queryJoinStrings(filter.Ids))
	}
	if filter.Name != "" {
		items = append(items, "name="+url.QueryEscape(filter.Name))
	}
	if filter.Author != "" {
		items = append(items, "author="+url.QueryEscape(filter.Author))
	}
	if filter.IsDeployed != 0 {
		items = append(items, "is_deployed="+strconv.FormatInt(int64(filter.IsDeployed), 10))
	}
	if filter.DeploymentEnabled != 0 {
		items = append(items, "deployment_enabled="+strconv.FormatInt(int64(filter.DeploymentEnabled), 10))
	}
	if filter.DeploymentState != 0 {
		items = append(items, "deployment_state="+strconv.FormatInt(int64(filter.DeploymentState), 10))
	}
	if len(items) > 0 {
		return u + "?" + strings.Join(items, "&")
	}
	return u
}

func (c *Client) GetModules(ctx context.Context, filter lib_models.ModulesFilter) ([]lib_models.ModuleReduced, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathModulesCollection))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appendModulesFilterQuery(u, filter), nil)
	if err != nil {
		return nil, err
	}
	var res []lib_models.ModuleReduced
	err = doJson(c, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) GetModule(ctx context.Context, id string) (lib_models.Module, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathModuleResource, id))
	if err != nil {
		return lib_models.Module{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return lib_models.Module{}, err
	}
	var res lib_models.Module
	err = doJson(c, req, &res)
	if err != nil {
		return lib_models.Module{}, err
	}
	return res, nil
}

func (c *Client) GetModulesChangeRequest(ctx context.Context) (lib_models.ModulesChangeRequest, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathModulesChangeRequestResource))
	if err != nil {
		return lib_models.ModulesChangeRequest{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return lib_models.ModulesChangeRequest{}, err
	}
	var res lib_models.ModulesChangeRequest
	err = doJson(c, req, &res)
	if err != nil {
		return lib_models.ModulesChangeRequest{}, err
	}
	return res, nil
}

func (c *Client) CreateModulesChangeRequest(
	ctx context.Context,
	reqItems []lib_models.ChangeRequestItem,
) (lib_models.ModulesChangeRequest, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathModulesChangeRequestResource))
	if err != nil {
		return lib_models.ModulesChangeRequest{}, err
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(reqItems)
	if err != nil {
		return lib_models.ModulesChangeRequest{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, buffer)
	if err != nil {
		return lib_models.ModulesChangeRequest{}, err
	}
	var res lib_models.ModulesChangeRequest
	err = doJson(client, req, &res)
	if err != nil {
		return lib_models.ModulesChangeRequest{}, err
	}
	return res, nil
}

func (c *Client) ExecModulesChangeRequest(ctx context.Context) (lib_models.Job, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathModulesChangeRequestResource))
	if err != nil {
		return lib_models.Job{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, nil)
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

func (c *Client) CancelModulesChangeRequest(ctx context.Context) error {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathModulesChangeRequestResource))
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	return doErr(c, req)
}

func (c *Client) GetModulesAvailableUpdatesCount(ctx context.Context) (int, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathModulesAvailableUpdatesCountResource))
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	var res int
	err = doJson(client, req, &res)
	if err != nil {
		return 0, err
	}
	return res, nil
}

func (c *Client) CreateModulesUpdateAllChangeRequest(ctx context.Context) (lib_models.ModulesChangeRequest, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathModulesChangeRequestResource))
	if err != nil {
		return lib_models.ModulesChangeRequest{}, err
	}
	u += "?update_all=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		return lib_models.ModulesChangeRequest{}, err
	}
	var res lib_models.ModulesChangeRequest
	err = doJson(client, req, &res)
	if err != nil {
		return lib_models.ModulesChangeRequest{}, err
	}
	return res, nil
}
