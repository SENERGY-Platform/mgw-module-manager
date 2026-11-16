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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

type ClientAutoUpdate struct {
	client  httpClient
	baseUrl string
}

func NewClientAutoUpdate(httpClient httpClient, baseUrl string) *ClientAutoUpdate {
	return &ClientAutoUpdate{
		client:  httpClient,
		baseUrl: baseUrl,
	}
}

func (c *ClientAutoUpdate) GetAutoUpdateSettings(ctx context.Context) (models.AutoUpdateSettings, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathAutoUpdateResource))
	if err != nil {
		return models.AutoUpdateSettings{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return models.AutoUpdateSettings{}, err
	}
	var res models.AutoUpdateSettings
	err = doJson(c.client, req, &res)
	if err != nil {
		return models.AutoUpdateSettings{}, err
	}
	return res, nil
}

func (c *ClientAutoUpdate) UpdateAutoUpdateSettings(ctx context.Context, settings models.AutoUpdateSettings) error {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathAutoUpdateResource))
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(settings)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, buffer)
	if err != nil {
		return err
	}
	return doErr(c.client, req)
}

func (c *ClientAutoUpdate) GetAutoUpdateStatus(ctx context.Context) (models.AutoUpdateStatus, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathAutoUpdateStatusResource))
	if err != nil {
		return models.AutoUpdateStatus{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return models.AutoUpdateStatus{}, err
	}
	var res models.AutoUpdateStatus
	err = doJson(c.client, req, &res)
	if err != nil {
		return models.AutoUpdateStatus{}, err
	}
	return res, nil
}
