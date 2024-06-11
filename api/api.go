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
	"github.com/SENERGY-Platform/go-service-base/job-hdl"
	"github.com/SENERGY-Platform/go-service-base/srv-info-hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
)

type Api struct {
	moduleHandler        handler.ModuleHandler
	modStagingHandler    handler.ModStagingHandler
	modUpdateHandler     handler.ModUpdateHandler
	deploymentHandler    handler.DeploymentHandler
	auxDeploymentHandler handler.AuxDeploymentHandler
	auxJobHandler        handler.AuxJobHandler
	advHandler           handler.DepAdvertisementHandler
	jobHandler           job_hdl.JobHandler
	srvInfoHdl           srv_info_hdl.SrvInfoHandler
	mu                   *util.RWMutex
}

func New(moduleHandler handler.ModuleHandler, moduleStagingHandler handler.ModStagingHandler, moduleUpdateHandler handler.ModUpdateHandler, deploymentHandler handler.DeploymentHandler, auxDeploymentHandler handler.AuxDeploymentHandler, jobHandler job_hdl.JobHandler, auxJobHandler handler.AuxJobHandler, advHandler handler.DepAdvertisementHandler, srvInfoHandler srv_info_hdl.SrvInfoHandler) *Api {
	return &Api{
		moduleHandler:        moduleHandler,
		modStagingHandler:    moduleStagingHandler,
		modUpdateHandler:     moduleUpdateHandler,
		deploymentHandler:    deploymentHandler,
		auxDeploymentHandler: auxDeploymentHandler,
		auxJobHandler:        auxJobHandler,
		advHandler:           advHandler,
		jobHandler:           jobHandler,
		srvInfoHdl:           srvInfoHandler,
		mu:                   &util.RWMutex{},
	}
}

type apiErr struct {
	msg string
	err error
}

func newApiErr(msg string, err error) error {
	return &apiErr{
		msg: msg,
		err: err,
	}
}

func (e *apiErr) Error() string {
	if e.msg == "" {
		return e.err.Error()
	}
	return e.msg + ": " + e.err.Error()
}

func (e *apiErr) Unwrap() error {
	return e.err
}
