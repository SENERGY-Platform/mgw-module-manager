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

import (
	"time"
)

type AuxDepBase struct {
	ID        string            `json:"id"` // uuid
	DepID     string            `json:"dep_id"`
	Image     string            `json:"image"`
	Labels    map[string]string `json:"labels"`
	Configs   map[string]string `json:"configs"`
	Volumes   map[string]string `json:"volumes"` // {name:mntPoint}
	Ref       string            `json:"ref"`
	Name      string            `json:"name"`
	RunConfig AuxDepRunConfig   `json:"run_config"`
	Enabled   bool              `json:"enabled"`
	Created   time.Time         `json:"created"`
	Updated   time.Time         `json:"updated"`
}

type AuxDepRunConfig struct {
	Command   string `json:"command"`
	PseudoTTY bool   `json:"pseudo_tty"`
}

type AuxDeployment struct {
	AuxDepBase
	Container AuxDepContainer `json:"container"`
}

type AuxDepContainer struct {
	ID    string         `json:"id"`
	Alias string         `json:"alias"`
	Info  *ContainerInfo `json:"info"`
}

type AuxDepFilter struct {
	IDs     []string
	Labels  map[string]string
	Image   string
	Enabled ToggleVal
}

type AuxDepReq struct {
	Image     string            `json:"image"`
	Labels    map[string]string `json:"labels"`
	Configs   map[string]string `json:"configs"`
	Volumes   map[string]string `json:"volumes"` // {name:mntPoint}
	Ref       string            `json:"ref"`     // only required by create method
	Name      string            `json:"name"`
	RunConfig *AuxDepRunConfig  `json:"run_config"`
}
