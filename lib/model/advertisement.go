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

package model

type AdvertisementBase struct {
	Ref   string            `json:"ref"`
	Items map[string]string `json:"items"`
}

type Advertisement struct {
	ModuleID string `json:"module_id"`
	Origin   string `json:"origin"`
	AdvertisementBase
}

type AdvFilter struct {
	ModuleID     string `json:"module_id"`
	DeploymentID string `json:"deployment_id"`
}
