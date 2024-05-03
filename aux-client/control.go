/*
 * Copyright 2024 InfAI (CC SES)
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

package aux_client

import (
	"context"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"net/http"
	"net/url"
)

func (c *Client) StartAuxDeployment(ctx context.Context, dID, aID string) (string, error) {
	u, err := url.JoinPath(c.baseUrl, model.AuxDeploymentsPath, aID, model.DepStartPath)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, nil)
	if err != nil {
		return "", err
	}
	setDepIdHeader(req, dID)
	return c.baseClient.ExecRequestString(req)
}

func (c *Client) StartAuxDeployments(ctx context.Context, dID string, filter model.AuxDepFilter) (string, error) {
	u, err := url.JoinPath(c.baseUrl, model.AuxDepBatchPath, model.DepStartPath)
	if err != nil {
		return "", err
	}
	u += genAuxDepQuery(genAuxDepFilterQuery(filter))
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, nil)
	if err != nil {
		return "", err
	}
	setDepIdHeader(req, dID)
	return c.baseClient.ExecRequestString(req)
}

func (c *Client) StopAuxDeployment(ctx context.Context, dID, aID string) (string, error) {
	u, err := url.JoinPath(c.baseUrl, model.AuxDeploymentsPath, aID, model.DepStopPath)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, nil)
	if err != nil {
		return "", err
	}
	setDepIdHeader(req, dID)
	return c.baseClient.ExecRequestString(req)
}

func (c *Client) StopAuxDeployments(ctx context.Context, dID string, filter model.AuxDepFilter) (string, error) {
	u, err := url.JoinPath(c.baseUrl, model.AuxDepBatchPath, model.DepStopPath)
	if err != nil {
		return "", err
	}
	u += genAuxDepQuery(genAuxDepFilterQuery(filter))
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, nil)
	if err != nil {
		return "", err
	}
	setDepIdHeader(req, dID)
	return c.baseClient.ExecRequestString(req)
}

func (c *Client) RestartAuxDeployment(ctx context.Context, dID, aID string) (string, error) {
	u, err := url.JoinPath(c.baseUrl, model.AuxDeploymentsPath, aID, model.DepRestartPath)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, nil)
	if err != nil {
		return "", err
	}
	setDepIdHeader(req, dID)
	return c.baseClient.ExecRequestString(req)
}

func (c *Client) RestartAuxDeployments(ctx context.Context, dID string, filter model.AuxDepFilter) (string, error) {
	u, err := url.JoinPath(c.baseUrl, model.AuxDepBatchPath, model.DepRestartPath)
	if err != nil {
		return "", err
	}
	u += genAuxDepQuery(genAuxDepFilterQuery(filter))
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, nil)
	if err != nil {
		return "", err
	}
	setDepIdHeader(req, dID)
	return c.baseClient.ExecRequestString(req)
}
