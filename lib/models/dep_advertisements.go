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

// DeploymentAdvertisementReduced is an advertisement as returned by the query endpoints that span all deployments.
// It omits the deployment ID so that deployments can discover each other by module ID and reference without gaining knowledge of internal deployment IDs.
type DeploymentAdvertisementReduced struct {
	Id        string            // unique ID of the advertisement, generated from the deployment ID and the reference
	ModuleId  string            // ID of the module whose deployment published the advertisement
	Reference string            // identifier chosen by the publishing deployment, unique per deployment
	Timestamp time.Time         // point in time at which the advertisement was created or last replaced
	Items     map[string]string // advertised data as key value pairs, content is defined by the publishing module
}

// DeploymentAdvertisement is an advertisement published by a deployment to make data available to other deployments.
// Advertisements are managed by the deployment they belong to and are replaced as a whole, they are not merged.
type DeploymentAdvertisement struct {
	Id           string            // unique ID of the advertisement, generated from the deployment ID and the reference
	DeploymentId string            // ID of the deployment that published the advertisement
	ModuleId     string            // ID of the module the publishing deployment is based on
	Reference    string            // identifier chosen by the publishing deployment, unique per deployment
	Timestamp    time.Time         // point in time at which the advertisement was created or last replaced
	Items        map[string]string // advertised data as key value pairs, content is defined by the publishing module
}

// DeploymentAdvertisementsFilter restricts the advertisements returned by the query endpoints that span all deployments.
// All fields are optional, an empty field is not applied, set fields are combined conjunctively.
type DeploymentAdvertisementsFilter struct {
	DeploymentId string   // only return advertisements of the deployment with this ID
	Ids          []string // only return advertisements with one of these IDs
	ModuleIds    []string // only return advertisements published by deployments of these modules
	References   []string // only return advertisements with one of these references
}

// DeploymentAdvertisementsFilterReduced restricts the advertisements returned or deleted for a single deployment.
// All fields are optional, an empty field is not applied, set fields are combined conjunctively.
type DeploymentAdvertisementsFilterReduced struct {
	Ids        []string // only return advertisements with one of these IDs
	References []string // only return advertisements with one of these references
}

// DeploymentAdvertisementInput is the user input for creating or replacing an advertisement of a deployment.
type DeploymentAdvertisementInput struct {
	Reference string            // identifier of the advertisement, an existing advertisement with the same reference is replaced
	Items     map[string]string // advertised data as key value pairs, content is defined by the publishing module
}
