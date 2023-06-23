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
	"sort"
)

func SetRoutes(e *gin.Engine, a lib.Api) {
	e.GET(model.ModulesPath, getModulesH(a))
	e.POST(model.ModulesPath, postModuleH(a))
	e.GET(model.ModulesPath+"/:"+modIdParam, getModuleH(a))
	e.DELETE(model.ModulesPath+"/:"+modIdParam, deleteModuleH(a))
	e.GET(model.ModulesPath+"/:"+modIdParam+"/"+model.DepTemplatePath, getModuleDeployTemplateH(a))
	e.GET(model.ModUpdatesPath, getModuleUpdates(a))
	e.GET(model.ModUpdatesPath+"/:"+modIdParam, getModuleUpdate(a))
	e.POST(model.ModUpdatesPath, postCheckModuleUpdates(a))
	e.GET(model.DeploymentsPath, getDeploymentsH(a))
	e.POST(model.DeploymentsPath, postDeploymentH(a))
	e.GET(model.DeploymentsPath+"/:"+depIdParam, getDeploymentH(a))
	e.PATCH(model.DeploymentsPath+"/:"+depIdParam, patchDeploymentUpdateH(a))
	e.PATCH(model.DeploymentsPath+"/:"+depIdParam+"/"+model.DepStartPath, patchDeploymentStart(a))
	e.PATCH(model.DeploymentsPath+"/:"+depIdParam+"/"+model.DepStopPath, patchDeploymentStop(a))
	e.DELETE(model.DeploymentsPath+"/:"+depIdParam, deleteDeploymentH(a))
	e.GET(model.DeploymentsPath+"/:"+depIdParam+"/"+model.DepUpdateTemplatePath, getDeploymentUpdateTemplateH(a))
	e.GET("/"+model.JobsPath, getJobsH(a))
	e.GET("/"+model.JobsPath+"/:"+jobIdParam, getJobH(a))
	e.PATCH("/"+model.JobsPath+"/:"+jobIdParam+"/"+model.JobsCancelPath, patchJobCancelH(a))
}

func GetRoutes(e *gin.Engine) [][2]string {
	routes := e.Routes()
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Path < routes[j].Path
	})
	var rInfo [][2]string
	for _, info := range routes {
		rInfo = append(rInfo, [2]string{info.Method, info.Path})
	}
	return rInfo
}
