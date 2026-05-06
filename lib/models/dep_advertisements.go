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

package models

import "time"

type DeploymentAdvertisementReduced struct {
	Id        string
	ModuleId  string
	Origin    string
	Reference string
	Timestamp time.Time
	Items     map[string]string
}

type DeploymentAdvertisement struct {
	Id           string
	DeploymentId string
	ModuleId     string
	Origin       string
	Reference    string
	Timestamp    time.Time
	Items        map[string]string
}
type DeploymentAdvertisementsFilter struct {
	DeploymentId string
	Ids          []string
	ModuleIds    []string
	Origins      []string
	References   []string
}

type DeploymentAdvertisementsFilterReduced struct {
	Ids        []string
	References []string
}

type DeploymentAdvertisementInput struct {
	Reference string
	Items     map[string]string
}
