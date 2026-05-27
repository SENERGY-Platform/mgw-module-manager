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
	"strconv"
	"strings"

	"github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

type ClientAuxiliaryDeployments struct {
	client  httpClient
	baseUrl string
}

func NewClientAuxiliaryDeployments(httpClient httpClient, baseUrl string) *ClientAuxiliaryDeployments {
	return &ClientAuxiliaryDeployments{
		client:  httpClient,
		baseUrl: baseUrl,
	}
}

func (c *ClientAuxiliaryDeployments) CreateAuxiliaryDeployment(
	ctx context.Context,
	deploymentId string,
	serviceInput models.AuxiliaryDeploymentInput,
) (models.Job, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathAuxiliaryDeploymentsCollection, deploymentId))
	if err != nil {
		return models.Job{}, err
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(serviceInput)
	if err != nil {
		return models.Job{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, buffer)
	if err != nil {
		return models.Job{}, err
	}
	var res models.Job
	err = doJson(c.client, req, &res)
	if err != nil {
		return models.Job{}, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) UpdateAuxiliaryDeployment(
	ctx context.Context,
	deploymentId string,
	auxDeploymentId string,
	serviceInput models.AuxiliaryDeploymentInput,
	incremental bool,
) (models.Job, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathAuxiliaryDeploymentResource, deploymentId, auxDeploymentId))
	if err != nil {
		return models.Job{}, err
	}
	if incremental {
		u += "?incremental=true"
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(serviceInput)
	if err != nil {
		return models.Job{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, buffer)
	if err != nil {
		return models.Job{}, err
	}
	var res models.Job
	err = doJson(c.client, req, &res)
	if err != nil {
		return models.Job{}, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) RecreateAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models.AuxiliaryDeploymentsFilterWithState,
) (models.Job, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathRecreateAuxiliaryDeployments, deploymentId))
	if err != nil {
		return models.Job{}, err
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(filter)
	if err != nil {
		return models.Job{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, buffer)
	if err != nil {
		return models.Job{}, err
	}
	var res models.Job
	err = doJson(c.client, req, &res)
	if err != nil {
		return models.Job{}, err
	}
	return res, nil
}

func appendAuxiliaryDeploymentsQuery(u string, filter models.AuxiliaryDeploymentsFilterWithState, allowAll bool) string {
	var items []string
	if len(filter.Ids) > 0 {
		items = append(items, "ids="+queryJoinStrings(filter.Ids))
	}
	if len(filter.Labels) > 0 {
		var labels []string
		for key, val := range filter.Labels {
			labels = append(labels, url.QueryEscape(key+"|"+val))
		}
		items = append(items, "labels="+strings.Join(labels, ","))
	}
	if filter.Image != "" {
		items = append(items, "image="+url.QueryEscape(filter.Image))
	}
	if filter.Enabled != 0 {
		items = append(items, "enabled="+strconv.FormatInt(int64(filter.Enabled), 10))
	}
	if filter.Recreate != 0 {
		items = append(items, "recreate="+strconv.FormatInt(int64(filter.Recreate), 10))
	}
	if filter.State != "" {
		items = append(items, "state="+filter.State)
	}
	if allowAll {
		items = append(items, "allow_all=true")
	}
	if len(items) > 0 {
		return u + "?" + strings.Join(items, "&")
	}
	return ""
}

func (c *ClientAuxiliaryDeployments) DeleteAuxiliaryDeployment(ctx context.Context, deploymentId, auxDeploymentId string) error {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathAuxiliaryDeploymentResource, deploymentId, auxDeploymentId))
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	return doErr(c.client, req)
}

func (c *ClientAuxiliaryDeployments) DeleteAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models.AuxiliaryDeploymentsFilterWithState,
	allowAll bool,
) ([]models.AuxiliaryDeploymentBatchResult, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathAuxiliaryDeploymentsCollection, deploymentId))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, appendAuxiliaryDeploymentsQuery(u, filter, allowAll), nil)
	if err != nil {
		return nil, err
	}
	var res []models.AuxiliaryDeploymentBatchResult
	err = doJson(c.client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) EnableAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models.AuxiliaryDeploymentsFilterWithState,
) ([]string, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathEnableAuxiliaryDeployments, deploymentId))
	if err != nil {
		return nil, err
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(filter)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, buffer)
	if err != nil {
		return nil, err
	}
	var res []string
	err = doJson(c.client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) DisableAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models.AuxiliaryDeploymentsFilterWithState,
) ([]string, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathDisableAuxiliaryDeployments, deploymentId))
	if err != nil {
		return nil, err
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(filter)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, buffer)
	if err != nil {
		return nil, err
	}
	var res []string
	err = doJson(c.client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) DeleteAuxiliaryDeploymentVolume(ctx context.Context, deploymentId, reference string) error {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathAuxiliaryDeploymentVolumeResource, deploymentId, reference))
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	return doErr(c.client, req)
}

func appendAuxiliaryDeploymentVolumesQuery(u string, references []string, allowAll, unused bool) string {
	var items []string
	if len(references) > 0 {
		items = append(items, "ids="+queryJoinStrings(references))
	}
	if allowAll {
		items = append(items, "allow_all=true")
	}
	if unused {
		items = append(items, "only_unsued=true")
	}
	if len(items) > 0 {
		return u + "?" + strings.Join(items, "&")
	}
	return ""
}

func (c *ClientAuxiliaryDeployments) DeleteAuxiliaryDeploymentVolumes(
	ctx context.Context,
	deploymentId string,
	filterReferences []string,
	allowAll bool,
) ([]models.AuxiliaryDeploymentVolumeResult, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathAuxiliaryDeploymentVolumesCollection, deploymentId))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		appendAuxiliaryDeploymentVolumesQuery(u, filterReferences, allowAll, false),
		nil,
	)
	if err != nil {
		return nil, err
	}
	var res []models.AuxiliaryDeploymentVolumeResult
	err = doJson(c.client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) DeleteUnusedAuxiliaryDeploymentVolumes(
	ctx context.Context,
	deploymentId string,
	excludeReferences []string,
) ([]models.AuxiliaryDeploymentVolumeResult, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathAuxiliaryDeploymentVolumesCollection, deploymentId))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		appendAuxiliaryDeploymentVolumesQuery(u, excludeReferences, false, true),
		nil,
	)
	if err != nil {
		return nil, err
	}
	var res []models.AuxiliaryDeploymentVolumeResult
	err = doJson(c.client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) GetAuxiliaryDeployment(
	ctx context.Context,
	deploymentId string,
	auxDeploymentId string,
) (models.AuxiliaryDeployment, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathAuxiliaryDeploymentResource, deploymentId, auxDeploymentId))
	if err != nil {
		return models.AuxiliaryDeployment{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return models.AuxiliaryDeployment{}, err
	}
	var res models.AuxiliaryDeployment
	err = doJson(c.client, req, &res)
	if err != nil {
		return models.AuxiliaryDeployment{}, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) GetAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models.AuxiliaryDeploymentsFilterWithState,
) (map[string]models.AuxiliaryDeployment, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathAuxiliaryDeploymentsCollection, deploymentId))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appendAuxiliaryDeploymentsQuery(u, filter, false), nil)
	if err != nil {
		return nil, err
	}
	var res map[string]models.AuxiliaryDeployment
	err = doJson(c.client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) GetReducedAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models.AuxiliaryDeploymentsFilterWithState,
) (map[string]models.AuxiliaryDeploymentReduced, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathReducedAuxiliaryDeploymentsCollection, deploymentId))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appendAuxiliaryDeploymentsQuery(u, filter, false), nil)
	if err != nil {
		return nil, err
	}
	var res map[string]models.AuxiliaryDeploymentReduced
	err = doJson(c.client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) GetAuxiliaryDeploymentVolumes(
	ctx context.Context,
	deploymentId string,
	filterReferences []string,
) (map[string]models.AuxiliaryDeploymentVolume, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathAuxiliaryDeploymentVolumesCollection, deploymentId))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet,
		appendAuxiliaryDeploymentVolumesQuery(u, filterReferences, false, false),
		nil,
	)
	if err != nil {
		return nil, err
	}
	var res map[string]models.AuxiliaryDeploymentVolume
	err = doJson(c.client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) GetAuxiliaryDeploymentVolumesWithMounts(
	ctx context.Context,
	deploymentId string,
	filterReferences []string,
) (map[string]models.AuxiliaryDeploymentVolumeWithMounts, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathAuxiliaryDeploymentVolumesWithMountsCollection, deploymentId))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet,
		appendAuxiliaryDeploymentVolumesQuery(u, filterReferences, false, false),
		nil,
	)
	var res map[string]models.AuxiliaryDeploymentVolumeWithMounts
	err = doJson(c.client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) GetCreateAuxiliaryDeploymentJobResult(
	ctx context.Context,
	jobId string,
) (models.AuxiliaryDeploymentCreateJobResult, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathCreateAuxiliaryDeploymentResultResource, jobId))
	if err != nil {
		return models.AuxiliaryDeploymentCreateJobResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return models.AuxiliaryDeploymentCreateJobResult{}, err
	}
	var res models.AuxiliaryDeploymentCreateJobResult
	err = doJson(c.client, req, &res)
	if err != nil {
		return models.AuxiliaryDeploymentCreateJobResult{}, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) GetUpdateAuxiliaryDeploymentJobResult(
	ctx context.Context,
	jobId string,
) (models.JobResult, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathUpdateAuxiliaryDeploymentResultResource, jobId))
	if err != nil {
		return models.JobResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return models.JobResult{}, err
	}
	var res models.JobResult
	err = doJson(c.client, req, &res)
	if err != nil {
		return models.JobResult{}, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) GetAuxiliaryDeploymentsJobResult(
	ctx context.Context,
	jobId string,
) (models.AuxiliaryDeploymentJobResult, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathAuxiliaryDeploymentsResultResource, jobId))
	if err != nil {
		return models.AuxiliaryDeploymentJobResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return models.AuxiliaryDeploymentJobResult{}, err
	}
	var res models.AuxiliaryDeploymentJobResult
	err = doJson(c.client, req, &res)
	if err != nil {
		return models.AuxiliaryDeploymentJobResult{}, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) GetJobs(ctx context.Context, filterIds []string) ([]models.Job, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathJobsCollection))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var res []models.Job
	err = doJson(c.client, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) GetJob(ctx context.Context, id string) (models.Job, error) {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathJobResource, id))
	if err != nil {
		return models.Job{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return models.Job{}, err
	}
	var res models.Job
	err = doJson(c.client, req, &res)
	if err != nil {
		return models.Job{}, err
	}
	return res, nil
}

func (c *ClientAuxiliaryDeployments) CancelJobs(ctx context.Context, ids []string) error {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathCancelJobs))
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(ids)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, buffer)
	if err != nil {
		return err
	}
	return doErr(c.client, req)
}

func (c *ClientAuxiliaryDeployments) CancelJob(ctx context.Context, id string) error {
	u, err := url.JoinPath(c.baseUrl, getUrlRelPath(constants.HttpPathJobResource, id))
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, nil)
	if err != nil {
		return err
	}
	return doErr(c.client, req)
}
