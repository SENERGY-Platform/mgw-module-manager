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

package http_engine

import (
	"github.com/SENERGY-Platform/mgw-module-manager/itf"
	"github.com/gin-gonic/gin"
)

func SetRoutes(e *gin.Engine, a itf.Api) {
	e.GET("modules", getModulesH(a))
	e.GET("modules/:"+modIdParam, getModuleH(a))
	e.GET("modules/:"+modIdParam+"/input_template", getInputTemplateH(a))
	e.GET("deployments", getDeploymentsH(a))
	e.GET("deployments/:"+depIdParam, getDeploymentH(a))
	e.PUT("deployments/:"+depIdParam, putDeploymentH(a))
	e.DELETE("deployments/:"+depIdParam, deleteDeploymentH(a))
	e.POST("deployments", postDeploymentH(a))
}
