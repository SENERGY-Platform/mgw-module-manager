/*
 * Copyright 2025 InfAI (CC SES)
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

package standard

import (
	gin_mw "github.com/SENERGY-Platform/gin-middleware"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/http_hdl/shared"
	_ "github.com/SENERGY-Platform/mgw-module-manager/handler/http_hdl/swagger_docs"
	"github.com/SENERGY-Platform/mgw-module-manager/lib"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var routes = gin_mw.Routes[lib.Api]{
	getModulesH,
	getModuleH,
	postModuleH,
	deleteModuleH,
	getModuleDeployTemplateH,
	getModuleUpdatesH,
	getModuleUpdateH,
	postCheckModuleUpdatesH,
	patchPrepareModuleUpdateH,
	patchCancelPendingModuleUpdateH,
	patchModuleUpdateH,
	getPendingModuleUpdateTemplateH,
	getDeploymentsH,
	getDeploymentH,
	postDeploymentH,
	patchDeploymentUpdateH,
	patchDeploymentStartH,
	patchDeploymentsStartH,
	patchDeploymentStopH,
	patchDeploymentsStopH,
	patchDeploymentRestartH,
	patchDeploymentsRestartH,
	deleteDeploymentH,
	patchDeploymentsDeleteH,
	getDeploymentUpdateTemplateH,
	getJobsH,
	getJobH,
	patchJobCancelH,
	getServiceHealthH,
}

// SetRoutes
// @title Module Manager API
// @version 0.7.2
// @description Provides access to module and deployment management functions.
// @license.name Apache-2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /
func SetRoutes(e *gin.Engine, a lib.Api) error {
	rg := e.Group("")
	routes = append(routes, shared.Routes...)
	err := routes.Set(a, rg, util.Logger)
	if err != nil {
		return err
	}
	rg.GET("swagger/*any", ginSwagger.WrapHandler(swaggerFiles.NewHandler(), ginSwagger.InstanceName("standard")))
	return nil
}
