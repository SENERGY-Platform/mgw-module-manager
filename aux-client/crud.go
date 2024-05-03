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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) GetAuxDeployments(ctx context.Context, dID string, filter model.AuxDepFilter, assets, containerInfo bool) (map[string]model.AuxDeployment, error) {
	u, err := url.JoinPath(c.baseUrl, model.AuxDeploymentsPath)
	if err != nil {
		return nil, err
	}
	q := genAuxDepFilterQuery(filter)
	if assets {
		q += "assets=true"
	}
	if containerInfo {
		q += "container_info=true"
	}
	if q != "" {
		u += "?" + q
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	setDepIdHeader(req, dID)
	var auxDeployments map[string]model.AuxDeployment
	err = c.baseClient.ExecRequestJSON(req, &auxDeployments)
	if err != nil {
		return nil, err
	}
	return auxDeployments, nil
}

func (c *Client) GetAuxDeployment(ctx context.Context, dID, aID string, assets, containerInfo bool) (model.AuxDeployment, error) {
	u, err := url.JoinPath(c.baseUrl, model.AuxDeploymentsPath, aID)
	if err != nil {
		return model.AuxDeployment{}, err
	}
	q := ""
	if assets {
		q += "assets=true"
	}
	if containerInfo {
		q += "container_info=true"
	}
	if q != "" {
		u += "?" + q
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return model.AuxDeployment{}, err
	}
	setDepIdHeader(req, dID)
	var auxDeployment model.AuxDeployment
	err = c.baseClient.ExecRequestJSON(req, &auxDeployment)
	if err != nil {
		return model.AuxDeployment{}, err
	}
	return auxDeployment, nil
}

func (c *Client) CreateAuxDeployment(ctx context.Context, dID string, auxDepInput model.AuxDepReq, forcePullImg bool) (string, error) {
	u, err := url.JoinPath(c.baseUrl, model.AuxDeploymentsPath)
	if err != nil {
		return "", err
	}
	if forcePullImg {
		u += "?force_pull_img=true"
	}
	body, err := json.Marshal(auxDepInput)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	setDepIdHeader(req, dID)
	return c.baseClient.ExecRequestString(req)
}

func (c *Client) UpdateAuxDeployment(ctx context.Context, dID, aID string, auxDepInput model.AuxDepReq, incremental, forcePullImg bool) (string, error) {
	u, err := url.JoinPath(c.baseUrl, model.AuxDeploymentsPath, aID)
	if err != nil {
		return "", err
	}
	q := ""
	if incremental {
		q += "incremental=true"
	}
	if forcePullImg {
		q += "force_pull_img=true"
	}
	if q != "" {
		u += "?" + q
	}
	body, err := json.Marshal(auxDepInput)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	setDepIdHeader(req, dID)
	return c.baseClient.ExecRequestString(req)
}

func (c *Client) DeleteAuxDeployment(ctx context.Context, dID, aID string, force bool) (string, error) {
	u, err := url.JoinPath(c.baseUrl, model.AuxDeploymentsPath, aID)
	if err != nil {
		return "", err
	}
	if force {
		u += "?force=true"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return "", err
	}
	setDepIdHeader(req, dID)
	return c.baseClient.ExecRequestString(req)
}

func (c *Client) DeleteAuxDeployments(ctx context.Context, dID string, filter model.AuxDepFilter, force bool) (string, error) {
	u, err := url.JoinPath(c.baseUrl, model.AuxDepBatchPath, model.DepDeletePath)
	if err != nil {
		return "", err
	}
	q := genAuxDepFilterQuery(filter)
	if force {
		q += "force=true"
	}
	if q != "" {
		u += "?" + q
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, nil)
	if err != nil {
		return "", err
	}
	setDepIdHeader(req, dID)
	return c.baseClient.ExecRequestString(req)
}

func genAuxDepFilterQuery(filter model.AuxDepFilter) string {
	var q []string
	if filter.Image != "" {
		q = append(q, "image="+filter.Image)
	}
	if filter.Enabled != 0 {
		q = append(q, "enabled="+fmt.Sprintf("%v", filter.Enabled))
	}
	if len(filter.Labels) > 0 {
		q = append(q, "labels="+genLabels(filter.Labels, "=", ","))
	}
	if len(q) > 0 {
		return strings.Join(q, "&")
	}
	return ""
}
