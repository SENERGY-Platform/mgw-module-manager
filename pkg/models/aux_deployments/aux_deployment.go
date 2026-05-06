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

import (
	"time"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

type AuxiliaryDeployment struct {
	Id           string
	DeploymentId string
	Reference    string
	Name         string
	Image        string
	Created      time.Time
	Updated      time.Time
	Enabled      bool
	Recreate     bool
	Container    AuxiliaryDeploymentContainer
	RunConfig    lib_models.AuxiliaryDeploymentRunConfig
}

type AuxiliaryDeploymentContainer struct {
	Name  string
	Alias string
}

type AuxiliaryDeploymentVolumeMount struct {
	VolumeId              string
	VolumeName            string
	Reference             string
	AuxiliaryDeploymentId string
	MountPath             string
}

type AuxiliaryDeploymentParent struct {
	Id                   string
	Enabled              bool
	AuxiliaryDeployments map[string]AuxiliaryDeployment
}
