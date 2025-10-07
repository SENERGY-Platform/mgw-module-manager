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
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	"time"
)

type Module struct {
	*module_lib.Module
	Added   time.Time `json:"added"`
	Updated time.Time `json:"updated"`
}

type ModFilter struct {
	IDs            []string
	Name           string
	Author         string
	Type           string
	DeploymentType string
	Tags           map[string]struct{}
}

type ModAddRequest struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type ModDeployTemplate struct {
	InputTemplate
	Dependencies map[string]InputTemplate `json:"dependencies"`
}

type ModUpdateTemplate = ModDeployTemplate

type ModUpdate struct {
	Versions        []string          `json:"versions"`
	Checked         time.Time         `json:"checked"`
	Pending         bool              `json:"pending"`
	PendingVersions map[string]string `json:"pending_versions"`
}

type ModUpdatePrepareRequest struct {
	Version string `json:"version"`
}

type ModUpdateRequest struct {
	DepInput
	Dependencies map[string]DepInput `json:"dependencies"`
}
