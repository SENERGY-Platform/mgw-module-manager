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
	"strings"

	"github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

type ClientDeploymentAdvertisements struct {
	client  httpClient
	baseUrl string
}

func NewClientDeploymentAdvertisements(httpClient httpClient, baseUrl string) *ClientDeploymentAdvertisements {
	return &ClientDeploymentAdvertisements{
		client:  httpClient,
		baseUrl: baseUrl,
	}
}

func (c *ClientDeploymentAdvertisements) GetDeploymentAdvertisement(
	ctx context.Context,
	deploymentId string,
	reference string,
) (models.DeploymentAdvertisement, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathDeploymentAdvertisementResource, deploymentId, reference))
	if err != nil {
		return models.DeploymentAdvertisement{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return models.DeploymentAdvertisement{}, err
	}
	var res models.DeploymentAdvertisement
	err = doJson(c.client, req, &res)
	if err != nil {
		return models.DeploymentAdvertisement{}, err
	}
	return res, nil
}

func (c *ClientDeploymentAdvertisements) GetDeploymentAdvertisementById(
	ctx context.Context,
	deploymentId string,
	id string,
) (models.DeploymentAdvertisement, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathDeploymentAdvertisementByIdResource, deploymentId, id))
	if err != nil {
		return models.DeploymentAdvertisement{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return models.DeploymentAdvertisement{}, err
	}
	var res models.DeploymentAdvertisement
	err = doJson(c.client, req, &res)
	if err != nil {
		return models.DeploymentAdvertisement{}, err
	}
	return res, nil
}

func appendDeploymentAdvertisementsQueryReduced(u string, filter models.DeploymentAdvertisementsFilterReduced, allowAll bool) string {
	var items []string
	if len(filter.Ids) > 0 {
		items = append(items, "ids="+queryJoinStrings(filter.Ids))
	}
	if len(filter.References) > 0 {
		items = append(items, "references="+queryJoinStrings(filter.References))
	}
	if allowAll {
		items = append(items, "allow_all=true")
	}
	if len(items) > 0 {
		return u + "?" + strings.Join(items, "&")
	}
	return ""
}

func (c *ClientDeploymentAdvertisements) GetDeploymentAdvertisements(
	ctx context.Context,
	deploymentId string,
	filter models.DeploymentAdvertisementsFilterReduced,
) (map[string]models.DeploymentAdvertisement, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathDeploymentAdvertisementsCollection, deploymentId))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appendDeploymentAdvertisementsQueryReduced(u, filter, false), nil)
	if err != nil {
		return nil, err
	}
	var res map[string]models.DeploymentAdvertisement
	err = doJson(c.client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ClientDeploymentAdvertisements) PutDeploymentAdvertisement(
	ctx context.Context,
	deploymentId string,
	reference string,
	items map[string]string,
) (string, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathDeploymentAdvertisementResource, deploymentId, reference))
	if err != nil {
		return "", err
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(items)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, buffer)
	if err != nil {
		return "", err
	}
	var res string
	err = doJson(c.client, req, &res)
	if err != nil {
		return "", err
	}
	return res, nil
}

func (c *ClientDeploymentAdvertisements) PutDeploymentAdvertisements(
	ctx context.Context,
	deploymentId string,
	inputs []models.DeploymentAdvertisementInput,
	incremental bool,
) (map[string]string, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathDeploymentAdvertisementsCollection, deploymentId))
	if err != nil {
		return nil, err
	}
	if incremental {
		u += "?incremental=true"
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(inputs)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, buffer)
	if err != nil {
		return nil, err
	}
	var res map[string]string
	err = doJson(c.client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ClientDeploymentAdvertisements) DeleteDeploymentAdvertisement(
	ctx context.Context,
	deploymentId string,
	reference string,
) error {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathDeploymentAdvertisementResource, deploymentId, reference))
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	return doErr(c.client, req)
}

func (c *ClientDeploymentAdvertisements) DeleteDeploymentAdvertisements(
	ctx context.Context,
	deploymentId string,
	filter models.DeploymentAdvertisementsFilterReduced,
	allowAll bool,
) error {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathDeploymentAdvertisementsCollection, deploymentId))
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, appendDeploymentAdvertisementsQueryReduced(u, filter, allowAll), nil)
	if err != nil {
		return err
	}
	return doErr(c.client, req)
}

func appendDeploymentAdvertisementsQuery(u string, filter models.DeploymentAdvertisementsFilter) string {
	var items []string
	if len(filter.Ids) > 0 {
		items = append(items, "ids="+queryJoinStrings(filter.Ids))
	}
	if len(filter.ModuleIds) > 0 {
		items = append(items, "module_ids="+queryJoinStrings(filter.ModuleIds))
	}
	if len(filter.Origins) > 0 {
		items = append(items, "origins="+queryJoinStrings(filter.Origins))
	}
	if len(filter.References) > 0 {
		items = append(items, "references="+queryJoinStrings(filter.References))
	}
	if len(items) > 0 {
		return u + "?" + strings.Join(items, "&")
	}
	return ""
}

func (c *ClientDeploymentAdvertisements) QueryDeploymentAdvertisements(
	ctx context.Context,
	filter models.DeploymentAdvertisementsFilter,
) ([]models.DeploymentAdvertisementReduced, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathDeploymentAdvertisementsQueryCollection))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appendDeploymentAdvertisementsQuery(u, filter), nil)
	if err != nil {
		return nil, err
	}
	var res []models.DeploymentAdvertisementReduced
	err = doJson(c.client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ClientDeploymentAdvertisements) QueryDeploymentAdvertisement(
	ctx context.Context,
	id string,
) (models.DeploymentAdvertisementReduced, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathDeploymentAdvertisementQueryResource, id))
	if err != nil {
		return models.DeploymentAdvertisementReduced{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return models.DeploymentAdvertisementReduced{}, err
	}
	var res models.DeploymentAdvertisementReduced
	err = doJson(c.client, req, &res)
	if err != nil {
		return models.DeploymentAdvertisementReduced{}, err
	}
	return res, nil
}
