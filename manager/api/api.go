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
	"github.com/gin-gonic/gin"
	"module-manager/manager/api/util"
	"module-manager/manager/itf"
)

type Api struct {
	moduleHandler     itf.ModuleHandler
	deploymentHandler itf.DeploymentHandler
}

func New(moduleHandler itf.ModuleHandler, deploymentHandler itf.DeploymentHandler) *Api {
	return &Api{moduleHandler: moduleHandler, deploymentHandler: deploymentHandler}
}

func (a *Api) SetRoutes(e *gin.Engine) {
	e.GET("modules", a.GetModules)
	e.GET("modules/:"+util.ModuleParam, a.GetModule)
	e.GET("modules/:"+util.ModuleParam+"/input_template", a.GetModuleInputTemplate)
	e.POST("deployments", a.PostDeployment)
}
