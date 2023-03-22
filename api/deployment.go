/*
 * Copyright 2022 InfAI (CC SES)
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

package api

import (
	"module-manager/model"
)

func (a *Api) GetInputTemplate(id string) (model.InputTemplate, error) {
	m, err := a.moduleHandler.Read(id)
	if err != nil {
		return model.InputTemplate{}, err
	}
	template := a.deploymentHandler.InputTemplate(m)
	return template, nil
}

func (a *Api) AddDeployment(dr model.DeploymentRequest) (string, error) {
	m, err := a.moduleHandler.Read(dr.ModuleID)
	if err != nil {
		return "", err
	}
	id, err := a.deploymentHandler.Add(dr.DeploymentBase, m)
	if err != nil {
		return "", err
	}
	return id, nil
}