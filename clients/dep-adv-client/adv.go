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

package dep_adv_client

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) QueryDepAdvertisements(ctx context.Context, filter model.DepAdvFilter) ([]model.DepAdvertisement, error) {
	u, err := url.JoinPath(c.baseUrl, model.DiscoveryPath)
	if err != nil {
		return nil, err
	}
	u += genQueryAdvertisementsQuery(filter)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var ads []model.DepAdvertisement
	err = c.baseClient.ExecRequestJSON(req, &ads)
	if err != nil {
		return nil, err
	}
	return ads, nil
}

func (c *Client) GetDepAdvertisement(ctx context.Context, dID, ref string) (model.DepAdvertisement, error) {
	u, err := url.JoinPath(c.baseUrl, model.DepAdvertisementsPath, ref)
	if err != nil {
		return model.DepAdvertisement{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return model.DepAdvertisement{}, err
	}
	setDepIdHeader(req, dID)
	var adv model.DepAdvertisement
	err = c.baseClient.ExecRequestJSON(req, &adv)
	if err != nil {
		return model.DepAdvertisement{}, err
	}
	return adv, nil
}

func (c *Client) GetDepAdvertisements(ctx context.Context, dID string) (map[string]model.DepAdvertisement, error) {
	u, err := url.JoinPath(c.baseUrl, model.DepAdvertisementsPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	setDepIdHeader(req, dID)
	var ads map[string]model.DepAdvertisement
	err = c.baseClient.ExecRequestJSON(req, &ads)
	if err != nil {
		return nil, err
	}
	return ads, nil
}

func (c *Client) PutDepAdvertisement(ctx context.Context, dID string, adv model.DepAdvertisementBase) error {
	u, err := url.JoinPath(c.baseUrl, model.DepAdvertisementsPath, adv.Ref)
	if err != nil {
		return err
	}
	body, err := json.Marshal(adv)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	setDepIdHeader(req, dID)
	return c.baseClient.ExecRequestVoid(req)
}

func (c *Client) PutDepAdvertisements(ctx context.Context, dID string, ads map[string]model.DepAdvertisementBase) error {
	u, err := url.JoinPath(c.baseUrl, model.DepAdvertisementsBatchPath)
	if err != nil {
		return err
	}
	body, err := json.Marshal(ads)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	setDepIdHeader(req, dID)
	return c.baseClient.ExecRequestVoid(req)
}

func (c *Client) DeleteDepAdvertisement(ctx context.Context, dID, ref string) error {
	u, err := url.JoinPath(c.baseUrl, model.DepAdvertisementsPath, ref)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	setDepIdHeader(req, dID)
	return c.baseClient.ExecRequestVoid(req)
}

func (c *Client) DeleteDepAdvertisements(ctx context.Context, dID string) error {
	u, err := url.JoinPath(c.baseUrl, model.DepAdvertisementsBatchPath)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	setDepIdHeader(req, dID)
	return c.baseClient.ExecRequestVoid(req)
}

func genQueryAdvertisementsQuery(filter model.DepAdvFilter) string {
	var q []string
	if filter.ModuleID != "" {
		q = append(q, "module_id="+filter.ModuleID)
	}
	if filter.Origin != "" {
		q = append(q, "origin="+filter.Origin)
	}
	if len(filter.Ref) > 0 {
		q = append(q, "ref="+filter.Ref)
	}
	if len(q) > 0 {
		return "?" + strings.Join(q, "&")
	}
	return ""
}
