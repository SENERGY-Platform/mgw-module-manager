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

package http_hdl

import (
	"github.com/SENERGY-Platform/mgw-module-manager/lib"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/gin-gonic/gin"
)

func SetRoutes(e *gin.Engine, a lib.Api) {
	e.GET(model.ModulesPath, getModulesH(a))
	e.GET(model.ModulesPath+"/:"+modIdParam, getModuleH(a))
	e.GET(model.ModulesPath+"/:"+modIdParam+"/"+model.DepTemplatePath, getInputTemplateH(a))
	e.GET(model.DeploymentsPath, getDeploymentsH(a))
	e.POST(model.DeploymentsPath, postDeploymentH(a))
	e.GET(model.DeploymentsPath+"/:"+depIdParam, getDeploymentH(a))
	e.PUT(model.DeploymentsPath+"/:"+depIdParam, putDeploymentH(a))
	e.POST(model.DeploymentsPath+"/:"+depIdParam+"/"+model.DepCtrlPath, postDeploymentCtrlH(a))
	e.DELETE(model.DeploymentsPath+"/:"+depIdParam, deleteDeploymentH(a))
}
