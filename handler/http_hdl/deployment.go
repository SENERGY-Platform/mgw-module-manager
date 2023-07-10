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
	"net/http"
)

const depIdParam = "d"

type getDeploymentsQuery struct {
	Name     string `form:"name"`
	ModuleID string `form:"module_id"`
	Indirect bool   `form:"indirect"`
}

type deleteDeploymentQuery struct {
	Orphans bool `form:"orphans"`
}

type stopDeploymentQuery struct {
	Dependencies bool `form:"dependencies"`
}

func getDeploymentsH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		query := getDeploymentsQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(model.NewInvalidInputError(err))
			return
		}
		filter := model.DepFilter{
			ModuleID: query.ModuleID,
			Name:     query.Name,
			Indirect: query.Indirect,
		}
		deployments, err := a.GetDeployments(gc.Request.Context(), filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, deployments)
	}
}

func getDeploymentH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		deployment, err := a.GetDeployment(gc.Request.Context(), gc.Param(depIdParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, deployment)
	}
}

func getDeploymentInstancesH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		depInst, err := a.GetDeploymentInstances(gc.Request.Context())
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, depInst)
	}
}

func getDeploymentInstanceH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		depInst, err := a.GetDeploymentInstance(gc.Request.Context(), gc.Param(depIdParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, depInst)
	}
}

func postDeploymentH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		var depReq model.DepCreateRequest
		err := gc.ShouldBindJSON(&depReq)
		if err != nil {
			_ = gc.Error(model.NewInvalidInputError(err))
			return
		}
		id, err := a.CreateDeployment(gc.Request.Context(), depReq.ModuleID, depReq.DepInput, depReq.Dependencies)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, id)
	}
}

func patchDeploymentUpdateH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		var depReq model.DepInput
		err := gc.ShouldBindJSON(&depReq)
		if err != nil {
			_ = gc.Error(model.NewInvalidInputError(err))
			return
		}
		jID, err := a.UpdateDeployment(gc.Request.Context(), gc.Param(depIdParam), depReq)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchDeploymentStartH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		err := a.StartDeployment(gc.Request.Context(), gc.Param(depIdParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func patchDeploymentStopH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		query := stopDeploymentQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(model.NewInvalidInputError(err))
			return
		}
		jID, err := a.StopDeployment(gc.Request.Context(), gc.Param(depIdParam), query.Dependencies)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func deleteDeploymentH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		query := deleteDeploymentQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(model.NewInvalidInputError(err))
			return
		}
		err := a.DeleteDeployment(gc.Request.Context(), gc.Param(depIdParam), query.Orphans)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func getDeploymentUpdateTemplateH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		inputTemplate, err := a.GetDeploymentUpdateTemplate(gc.Request.Context(), gc.Param(depIdParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, inputTemplate)
	}
}

func getDeploymentsHealthH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		healthInfo, err := a.GetDeploymentsHealth(gc.Request.Context())
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, healthInfo)
	}
}

func getDeploymentHealthH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		healthInfo, err := a.GetDeploymentHealth(gc.Request.Context(), gc.Param(depIdParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, healthInfo)
	}
}
