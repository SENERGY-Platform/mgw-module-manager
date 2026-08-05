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
	"fmt"
	"net/http"
	"net/url"
	"strings"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

func appendRefreshRepositoriesQuery(u string, filter lib_models.RepositoriesRefreshFilter) string {
	var items []string
	if len(filter.Types) > 0 {
		items = append(items, "types="+queryJoinStrings(filter.Types))
	}
	if len(filter.Sources) > 0 {
		items = append(items, "sources="+queryJoinStrings(filter.Sources))
	}
	if len(items) > 0 {
		return u + "?" + strings.Join(items, "&")
	}
	return u
}

func (c *Client) RefreshRepositories(ctx context.Context, filter lib_models.RepositoriesRefreshFilter) (lib_models.Job, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(lib_constants.HttpPathRepositoriesCollection))
	if err != nil {
		return lib_models.Job{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, appendRefreshRepositoriesQuery(u, filter), nil)
	if err != nil {
		return lib_models.Job{}, err
	}
	var res lib_models.Job
	err = doJson(c, req, &res)
	if err != nil {
		return lib_models.Job{}, err
	}
	return res, nil
}

func (c *Client) GetRepositories(ctx context.Context) ([]lib_models.Repository, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(lib_constants.HttpPathRepositoriesCollection))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var res []lib_models.Repository
	err = doJson(c, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) CreateRepository(ctx context.Context, repositoryType string, data []byte) error {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(lib_constants.HttpPathRepositoriesCollection))
	if err != nil {
		return err
	}
	u += "?type=" + url.QueryEscape(repositoryType)
	buffer := bytes.NewBuffer(data)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, buffer)
	if err != nil {
		return err
	}
	return doErr(c, req)
}

func (c *Client) DeleteRepository(ctx context.Context, source string) error {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(lib_constants.HttpPathRepositoryResource, source))
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	return doErr(c, req)
}

func appendRepoModulesFilterQuery(u string, filter lib_models.RepoModulesFilter) string {
	var items []string
	if len(filter.Ids) > 0 {
		items = append(items, "ids="+queryJoinStrings(filter.Ids))
	}
	if len(filter.Repositories) > 0 {
		var sources []string
		var channels []string
		for _, repository := range filter.Repositories {
			sources = append(sources, repository.Source)
			for _, channel := range repository.Channels {
				channels = append(channels, fmt.Sprintf("%s|%s", repository.Source, channel))
			}
		}
		items = append(items, "repositories="+queryJoinStrings(sources))
		items = append(items, "repository_channels="+queryJoinStrings(channels))
	}
	if filter.Name != "" {
		items = append(items, "name="+url.QueryEscape(filter.Name))
	}
	if filter.Installed {
		items = append(items, "installed=true")
	}
	if filter.UpdateAvailable {
		items = append(items, "update_available=true")
	}
	if len(items) > 0 {
		return u + "?" + strings.Join(items, "&")
	}
	return u
}

func (c *Client) GetRepositoryModules(ctx context.Context, filter lib_models.RepoModulesFilter) ([]lib_models.RepoModule, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(lib_constants.HttpPathRepositoryModulesCollection))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appendRepoModulesFilterQuery(u, filter), nil)
	if err != nil {
		return nil, err
	}
	var res []lib_models.RepoModule
	err = doJson(c, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
