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
	"github.com/SENERGY-Platform/mgw-module-manager/handler/http_hdl/util"
	"github.com/SENERGY-Platform/mgw-module-manager/lib"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/gin-gonic/gin"
	"net/http"
	"path"
)

type getDeploymentsFilterQuery struct {
	IDs      string `form:"ids"`
	Name     string `form:"name"`
	ModuleID string `form:"module_id"`
	Enabled  int8   `form:"enabled"`
	Indirect int8   `form:"indirect"`
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

func getDeploymentsH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_model.DeploymentsPath, func(gc *gin.Context) {
		query := getDeploymentsQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		filter := lib_model.DepFilter{
			IDs:      util.ParseStringSlice(query.IDs, ","),
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

func getDeploymentH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, path.Join(lib_model.DeploymentsPath, ":id"), func(gc *gin.Context) {
		query := getDeploymentQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		deployment, err := a.GetDeployment(gc.Request.Context(), gc.Param("id"), query.Assets, query.ContainerInfo)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, deployment)
	}
}

func postDeploymentH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_model.DeploymentsPath, func(gc *gin.Context) {
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

func patchDeploymentUpdateH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.DeploymentsPath, ":id"), func(gc *gin.Context) {
		var depReq lib_model.DepInput
		err := gc.ShouldBindJSON(&depReq)
		if err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.UpdateDeployment(gc.Request.Context(), gc.Param("id"), depReq)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchDeploymentStartH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.DeploymentsPath, ":id", lib_model.DepStartPath), func(gc *gin.Context) {
		query := startDeploymentQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.StartDeployment(gc.Request.Context(), gc.Param("id"), query.Dependencies)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchDeploymentsStartH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.DepBatchPath, lib_model.DepStartPath), func(gc *gin.Context) {
		query := startDeploymentsQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.StartDeployments(gc.Request.Context(), lib_model.DepFilter{
			IDs:      util.ParseStringSlice(query.IDs, ","),
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

func patchDeploymentStopH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.DeploymentsPath, ":id", lib_model.DepStopPath), func(gc *gin.Context) {
		query := stopDeploymentQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.StopDeployment(gc.Request.Context(), gc.Param("id"), query.Force)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchDeploymentsStopH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.DepBatchPath, lib_model.DepStopPath), func(gc *gin.Context) {
		query := stopDeploymentsQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.StopDeployments(gc.Request.Context(), lib_model.DepFilter{
			IDs:      util.ParseStringSlice(query.IDs, ","),
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

func patchDeploymentRestartH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.DeploymentsPath, ":id", lib_model.DepRestartPath), func(gc *gin.Context) {
		jID, err := a.RestartDeployment(gc.Request.Context(), gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchDeploymentsRestartH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.DepBatchPath, lib_model.DepRestartPath), func(gc *gin.Context) {
		query := getDeploymentsFilterQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.RestartDeployments(gc.Request.Context(), lib_model.DepFilter{
			IDs:      util.ParseStringSlice(query.IDs, ","),
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

func deleteDeploymentH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, path.Join(lib_model.DeploymentsPath, ":id"), func(gc *gin.Context) {
		query := deleteDeploymentQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.DeleteDeployment(gc.Request.Context(), gc.Param("id"), query.Force)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchDeploymentsDeleteH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.DepBatchPath, lib_model.DepDeletePath), func(gc *gin.Context) {
		query := deleteDeploymentsQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.DeleteDeployments(gc.Request.Context(), lib_model.DepFilter{
			IDs:      util.ParseStringSlice(query.IDs, ","),
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

func getDeploymentUpdateTemplateH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, path.Join(lib_model.DeploymentsPath, ":id", lib_model.DepUpdateTemplatePath), func(gc *gin.Context) {
		inputTemplate, err := a.GetDeploymentUpdateTemplate(gc.Request.Context(), gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, inputTemplate)
	}
}
