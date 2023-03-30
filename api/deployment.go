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

func (a *Api) AddDeployment(dr model.DepRequest) (string, error) {
	m, err := a.moduleHandler.Get(dr.ModuleID)
	if err != nil {
		return "", err
	}
	id, err := a.deploymentHandler.Add(m, dr.Name, dr.HostResources, dr.Secrets, dr.Configs)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (a *Api) GetDeployments() ([]model.Deployment, error) {
	panic("not implemented")
}

func (a *Api) GetDeployment() (model.Deployment, error) {
	panic("not implemented")
}

func (a *Api) StartDeployment(id string) error {
	panic("not implemented")
}

func (a *Api) StopDeployment(id string) error {
	panic("not implemented")
}

func (a *Api) UpdateDeployment(dr model.DepRequest) {
	panic("not implemented")
}

func (a *Api) DeleteDeployment(id string) error {
	panic("not implemented")
}
