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
	"net/http"
)

const depIdParam = "d"

type getDeploymentsFilterQuery struct {
	IDs      string `form:"ids"`
	Name     string `form:"name"`
	ModuleID string `form:"module_id"`
	Enabled  bool   `form:"enabled"`
	Indirect bool   `form:"indirect"`
}

type getDeploymentQuery struct {
	Assets        bool `form:"assets"`
	ContainerInfo bool `form:"container_info"`
}

type getDeploymentsQuery struct {
	getDeploymentsFilterQuery
	getDeploymentQuery
}

type deleteDeploymentQuery struct {
	Force bool `form:"force"`
}

type deleteDeploymentsQuery struct {
	getDeploymentsFilterQuery
	deleteDeploymentQuery
}

type startDeploymentQuery struct {
	Dependencies bool `form:"dependencies"`
}

type startDeploymentsQuery struct {
	getDeploymentsFilterQuery
	startDeploymentQuery
}

type stopDeploymentQuery struct {
	Force bool `form:"force"`
}

type stopDeploymentsQuery struct {
	getDeploymentsFilterQuery
	stopDeploymentQuery
}

func getDeploymentsH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		query := getDeploymentsQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		filter := lib_model.DepFilter{
			IDs:      parseStringSlice(query.IDs, ","),
			ModuleID: query.ModuleID,
			Name:     query.Name,
			Enabled:  query.Enabled,
			Indirect: query.Indirect,
		}
		deployments, err := a.GetDeployments(gc.Request.Context(), filter, query.Assets, query.ContainerInfo)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, deployments)
	}
}

func getDeploymentH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		query := getDeploymentQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		deployment, err := a.GetDeployment(gc.Request.Context(), gc.Param(depIdParam), query.Assets, query.ContainerInfo)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, deployment)
	}
}

func postDeploymentH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		var depReq lib_model.DepCreateRequest
		err := gc.ShouldBindJSON(&depReq)
		if err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
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
		var depReq lib_model.DepInput
		err := gc.ShouldBindJSON(&depReq)
		if err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
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
		query := startDeploymentQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.StartDeployment(gc.Request.Context(), gc.Param(depIdParam), query.Dependencies)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchDeploymentsStartH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		query := startDeploymentsQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.StartDeployments(gc.Request.Context(), lib_model.DepFilter{
			IDs:      parseStringSlice(query.IDs, ","),
			ModuleID: query.ModuleID,
			Name:     query.Name,
			Enabled:  query.Enabled,
			Indirect: query.Indirect,
		}, query.Dependencies)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchDeploymentStopH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		query := stopDeploymentQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.StopDeployment(gc.Request.Context(), gc.Param(depIdParam), query.Force)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchDeploymentsStopH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		query := stopDeploymentsQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.StopDeployments(gc.Request.Context(), lib_model.DepFilter{
			IDs:      parseStringSlice(query.IDs, ","),
			ModuleID: query.ModuleID,
			Name:     query.Name,
			Enabled:  query.Enabled,
			Indirect: query.Indirect,
		}, query.Force)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchDeploymentRestartH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		jID, err := a.RestartDeployment(gc.Request.Context(), gc.Param(depIdParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchDeploymentsRestartH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		query := getDeploymentsFilterQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.RestartDeployments(gc.Request.Context(), lib_model.DepFilter{
			IDs:      parseStringSlice(query.IDs, ","),
			ModuleID: query.ModuleID,
			Name:     query.Name,
			Enabled:  query.Enabled,
			Indirect: query.Indirect,
		})
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
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.DeleteDeployment(gc.Request.Context(), gc.Param(depIdParam), query.Force)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchDeploymentsDeleteH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		query := deleteDeploymentsQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.DeleteDeployments(gc.Request.Context(), lib_model.DepFilter{
			IDs:      parseStringSlice(query.IDs, ","),
			ModuleID: query.ModuleID,
			Name:     query.Name,
			Enabled:  query.Enabled,
			Indirect: query.Indirect,
		}, query.Force)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
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
