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
	standardGrp := e.Group("")
	restrictedGrp := e.Group(lib_model.RestrictedPath)
	setSharedRoutes(a, standardGrp, restrictedGrp)
	setModulesRoutes(a, standardGrp.Group(lib_model.ModulesPath))
	setUpdatesRoutes(a, standardGrp.Group(lib_model.ModUpdatesPath))
	setDeploymentsRoutes(a, standardGrp.Group(lib_model.DeploymentsPath))
	setDeploymentsBatchRoutes(a, standardGrp.Group(lib_model.DepBatchPath))
	setJobsRoutes(a, standardGrp.Group(lib_model.JobsPath))
	standardGrp.GET("health-check", getServiceHealthH(a))
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

func setModulesRoutes(a lib.Api, rg *gin.RouterGroup) {
	rg.GET("", getModulesH(a))
	rg.POST("", postModuleH(a))
	rg.GET(":"+modIdParam, getModuleH(a))
	rg.DELETE(":"+modIdParam, deleteModuleH(a))
	rg.GET(":"+modIdParam+"/"+lib_model.DepTemplatePath, getModuleDeployTemplateH(a))
}

func setUpdatesRoutes(a lib.Api, rg *gin.RouterGroup) {
	rg.GET("", getModuleUpdatesH(a))
	rg.POST("", postCheckModuleUpdatesH(a))
	rg.GET(":"+modIdParam, getModuleUpdateH(a))
	rg.PATCH(":"+modIdParam, patchModuleUpdateH(a))
	rg.GET(":"+modIdParam+"/"+lib_model.DepUpdateTemplatePath, getPendingModuleUpdateTemplateH(a))
	rg.PATCH(":"+modIdParam+"/"+lib_model.ModUptPreparePath, patchPrepareModuleUpdateH(a))
	rg.PATCH(":"+modIdParam+"/"+lib_model.ModUptCancelPath, patchCancelPendingModuleUpdateH(a))
}

func setDeploymentsRoutes(a lib.Api, rg *gin.RouterGroup) {
	rg.GET("", getDeploymentsH(a))
	rg.POST("", postDeploymentH(a))
	rg.GET(":"+depIdParam, getDeploymentH(a))
	rg.PATCH(":"+depIdParam, patchDeploymentUpdateH(a))
	rg.DELETE(":"+depIdParam, deleteDeploymentH(a))
	rg.GET(":"+depIdParam+"/"+lib_model.DepUpdateTemplatePath, getDeploymentUpdateTemplateH(a))
	rg.PATCH(":"+depIdParam+"/"+lib_model.DepStartPath, patchDeploymentStartH(a))
	rg.PATCH(":"+depIdParam+"/"+lib_model.DepStopPath, patchDeploymentStopH(a))
	rg.PATCH(":"+depIdParam+"/"+lib_model.DepRestartPath, patchDeploymentRestartH(a))
}

func setDeploymentsBatchRoutes(a lib.Api, rg *gin.RouterGroup) {
	rg.PATCH(lib_model.DepStartPath, patchDeploymentsStartH(a))
	rg.PATCH(lib_model.DepStopPath, patchDeploymentsStopH(a))
	rg.PATCH(lib_model.DepRestartPath, patchDeploymentsRestartH(a))
	rg.PATCH(lib_model.DepDeletePath, patchDeploymentsDeleteH(a))
}

func setJobsRoutes(a lib.Api, rg *gin.RouterGroup) {
	rg.GET("", getJobsH(a))
	rg.GET(":"+jobIdParam, getJobH(a))
	rg.PATCH(":"+jobIdParam+"/"+lib_model.JobsCancelPath, patchJobCancelH(a))
}

func setSharedRoutes(a lib.Api, rGroups ...*gin.RouterGroup) {
	for _, rg := range rGroups {
		rg.GET(lib_model.SrvInfoPath, getSrvInfoH(a))
	}
}
