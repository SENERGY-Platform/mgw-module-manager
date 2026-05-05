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

package aux_deployments

import models_error "github.com/SENERGY-Platform/mgw-module-manager/lib/models/results"

type ServiceInput struct {
	Reference string                `json:"reference"`
	Name      string                `json:"name"`
	Image     string                `json:"image"`
	PullImage bool                  `json:"pull_image"`
	Labels    map[string]string     `json:"labels"`  // {name:value}
	Configs   map[string]string     `json:"configs"` // {varName:value}
	Volumes   map[string]string     `json:"volumes"` // {mntPath:reference}
	RunConfig ServiceInputRunConfig `json:"run_config"`
	Enabled   bool                  `json:"enabled"`
	Recreate  bool                  `json:"recreate"` // recreate the auxiliary deployment if parent deployment gets updated
}

type ServiceInputRunConfig struct {
	Command   []string `json:"command"`
	PseudoTTY int      `json:"pseudo_tty"`
}

type UpdateServiceInput struct {
	ServiceInput
	Incremental bool `json:"incremental"`
}

type Result struct {
	Id             string `json:"id"`
	ContainerAlias string `json:"container_alias"`
}

type BatchResult struct {
	Id string `json:"id"`
	models_error.ErrorResult
}

type VolumeResult struct {
	Reference string `json:"reference"`
	models_error.ErrorResult
}
