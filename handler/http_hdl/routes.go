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
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/gin-gonic/gin"
	"sort"
)

func SetRoutes(e *gin.Engine, a lib.Api) {
	e.GET(lib_model.ModulesPath, getModulesH(a))
	e.POST(lib_model.ModulesPath, postModuleH(a))
	e.GET(lib_model.ModulesPath+"/:"+modIdParam, getModuleH(a))
	e.DELETE(lib_model.ModulesPath+"/:"+modIdParam, deleteModuleH(a))
	e.GET(lib_model.ModulesPath+"/:"+modIdParam+"/"+lib_model.DepTemplatePath, getModuleDeployTemplateH(a))
	e.GET(lib_model.ModUpdatesPath, getModuleUpdatesH(a))
	e.GET(lib_model.ModUpdatesPath+"/:"+modIdParam, getModuleUpdateH(a))
	e.PATCH(lib_model.ModUpdatesPath+"/:"+modIdParam, patchModuleUpdateH(a))
	e.PATCH(lib_model.ModUpdatesPath+"/:"+modIdParam+"/"+lib_model.ModUptPreparePath, patchPrepareModuleUpdateH(a))
	e.PATCH(lib_model.ModUpdatesPath+"/:"+modIdParam+"/"+lib_model.ModUptCancelPath, patchCancelPendingModuleUpdateH(a))
	e.GET(lib_model.ModUpdatesPath+"/:"+modIdParam+"/"+lib_model.DepUpdateTemplatePath, getPendingModuleUpdateTemplateH(a))
	e.POST(lib_model.ModUpdatesPath, postCheckModuleUpdatesH(a))
	e.GET(lib_model.DeploymentsPath, getDeploymentsH(a))
	e.POST(lib_model.DeploymentsPath, postDeploymentH(a))
	e.GET(lib_model.DeploymentsPath+"/:"+depIdParam, getDeploymentH(a))
	e.PATCH(lib_model.DeploymentsPath+"/:"+depIdParam, patchDeploymentUpdateH(a))
	e.PATCH(lib_model.DeploymentsPath+"/:"+depIdParam+"/"+lib_model.DepStartPath, patchDeploymentStartH(a))
	e.PATCH(lib_model.DeploymentsPath+"/:"+depIdParam+"/"+lib_model.DepStopPath, patchDeploymentStopH(a))
	e.PATCH(lib_model.DeploymentsPath+"/:"+depIdParam+"/"+lib_model.DepRestartPath, patchDeploymentRestartH(a))
	e.PATCH(lib_model.DepBatchPath+"/"+lib_model.DepStartPath, patchDeploymentsStartH(a))
	e.PATCH(lib_model.DepBatchPath+"/"+lib_model.DepStopPath, patchDeploymentsStopH(a))
	e.PATCH(lib_model.DepBatchPath+"/"+lib_model.DepRestartPath, patchDeploymentsRestartH(a))
	e.PATCH(lib_model.DepBatchPath+"/"+lib_model.DepDeletePath, patchDeploymentsDeleteH(a))
	e.DELETE(lib_model.DeploymentsPath+"/:"+depIdParam, deleteDeploymentH(a))
	e.GET(lib_model.DeploymentsPath+"/:"+depIdParam+"/"+lib_model.DepUpdateTemplatePath, getDeploymentUpdateTemplateH(a))
	e.GET(lib_model.JobsPath, getJobsH(a))
	e.GET(lib_model.JobsPath+"/:"+jobIdParam, getJobH(a))
	e.PATCH(lib_model.JobsPath+"/:"+jobIdParam+"/"+lib_model.JobsCancelPath, patchJobCancelH(a))
	e.GET(lib_model.SrvInfoPath, getSrvInfoH(a))
	e.GET("health-check", getServiceHealthH(a))
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

func GetPathFilter() []string {
	return []string{
		"/health-check",
	}
}
