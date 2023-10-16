/*
 * Copyright 2023 InfAI (CC SES)
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

import "time"

type SubDepBase struct {
	DepID  string            `json:"dep_id"`
	Image  string            `json:"image"`
	Labels map[string]string `json:"labels"`
	Name   *string           `json:"name"`
}

type SubDeployment struct {
	ID string `json:"id"` // uuid
	SubDepBase
	Ref     string           `json:"ref"`    // container name: mgw-sd- + SubDeployment:ID
	CtrID   string           `json:"ctr_id"` // docker container id
	CtrInfo *SubDepContainer `json:"ctr_info"`
	Created time.Time        `json:"created"`
	Updated time.Time        `json:"updated"`
}

type SubDepContainer struct {
	ImageID string `json:"image_id"` // docker image id
	State   string `json:"state"`    // docker container state
}

type SubDepRequest struct {
	SubDepBase
	EnvVars map[string]string `json:"env_vars"`
}

type SubDepFilter struct {
	Labels map[string]string `json:"labels"`
	Image  string            `json:"image"`
	State  string            `json:"state"`
}
