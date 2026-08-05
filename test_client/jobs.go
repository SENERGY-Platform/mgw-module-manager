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
	"time"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

func (c *Client) GetJobs(ctx context.Context, filterIds []string) ([]lib_models.Job, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathJobsCollection))
	if err != nil {
		return nil, err
	}
	if len(filterIds) > 0 {
		u += "?ids=" + queryJoinStrings(filterIds)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var res []lib_models.Job
	err = doJson(c, req, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) GetJob(ctx context.Context, id string) (lib_models.Job, error) {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathJobResource, id))
	if err != nil {
		return lib_models.Job{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
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

func (c *Client) CancelJobs(ctx context.Context, ids []string) error {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathCancelJobs))
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
	return doErr(c, req)
}

func (c *Client) CancelJob(ctx context.Context, id string) error {
	u, err := url.JoinPath(c.BaseUrl, getUrlRelPath(lib_constants.HttpPathJobResource, id))
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, nil)
	if err != nil {
		return err
	}
	return doErr(c, req)
}

func (c *Client) AwaitJob(ctx context.Context, id string) (lib_models.Job, error) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			j, err := c.GetJob(ctx, id)
			if err != nil {
				return lib_models.Job{}, err
			}
			if !j.End.IsZero() {
				return j, nil
			}
			timer.Reset(time.Second)
		case <-ctx.Done():
			err := c.CancelJob(context.Background(), id)
			if err != nil {
				return lib_models.Job{}, err
			}
			return lib_models.Job{}, ctx.Err()
		}
	}
}
