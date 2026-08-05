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

func (c *Client) CreateGlobalConfig(ctx context.Context, input lib_models.GlobalConfigInput) (string, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathGlobalConfigsCollection))
	if err != nil {
		return "", err
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(input)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, buffer)
	if err != nil {
		return "", err
	}
	var res string
	err = doJson(client, req, &res)
	if err != nil {
		return "", err
	}
	return res, nil
}

func (c *Client) GetGlobalConfig(ctx context.Context, id string) (lib_models.GlobalConfig, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathGlobalConfigResource, id))
	if err != nil {
		return lib_models.GlobalConfig{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return lib_models.GlobalConfig{}, err
	}
	var res lib_models.GlobalConfig
	err = doJson(client, req, &res)
	if err != nil {
		return lib_models.GlobalConfig{}, err
	}
	return res, nil
}

func (c *Client) GetGlobalConfigs(ctx context.Context, ids []string) (map[string]lib_models.GlobalConfig, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathGlobalConfigsCollection))
	if err != nil {
		return nil, err
	}
	if len(ids) > 0 {
		u += "?ids=" + queryJoinStrings(ids)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var res map[string]lib_models.GlobalConfig
	err = doJson(client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) UpdateGlobalConfig(ctx context.Context, config lib_models.GlobalConfig) error {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathGlobalConfigResource, config.Id))
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(lib_models.GlobalConfigInput{
		Name:           config.Name,
		InterfaceValue: config.InterfaceValue,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, buffer)
	if err != nil {
		return err
	}
	return doErr(client, req)
}

func (c *Client) DeleteGlobalConfig(ctx context.Context, id string) error {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathGlobalConfigResource, id))
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	return doErr(client, req)
}

func appendDeleteGlobalConfigsFilter(u string, ids []string, allowAll bool) string {
	var items []string
	if len(ids) > 0 {
		items = append(items, "ids="+queryJoinStrings(ids))
	}
	if allowAll {
		items = append(items, "allow_all=true")
	}
	if len(items) > 0 {
		return u + "?" + strings.Join(items, "&")
	}
	return u
}

func (c *Client) DeleteGlobalConfigs(ctx context.Context, ids []string, allowAll bool) error {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathGlobalConfigsCollection))
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, appendDeleteGlobalConfigsFilter(u, ids, allowAll), nil)
	if err != nil {
		return err
	}
	return doErr(client, req)
}
