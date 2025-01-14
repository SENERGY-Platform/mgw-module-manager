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

package shared

import (
	"github.com/SENERGY-Platform/mgw-module-manager/handler/http_hdl/util"
	"github.com/SENERGY-Platform/mgw-module-manager/lib"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/gin-gonic/gin"
	"net/http"
	"path"
)

type getAuxDeploymentsFilterQuery struct {
	IDs     string `form:"ids"`
	Labels  string `form:"labels"`
	Image   string `form:"image"`
	Enabled int8   `form:"enabled"`
}

type getAuxDeploymentQuery struct {
	Assets        bool `form:"assets"`
	ContainerInfo bool `form:"container_info"`
}

type createAuxDeploymentQuery struct {
	ForcePullImg bool `form:"force_pull_img"`
}

type updateAuxDeploymentQuery struct {
	createAuxDeploymentQuery
	Incremental bool `form:"incremental"`
}

type getAuxDeploymentsQuery struct {
	getAuxDeploymentsFilterQuery
	getAuxDeploymentQuery
}

type deleteAuxDeploymentQuery struct {
	Force bool `form:"force"`
}

type deleteAuxDeploymentsQuery struct {
	getAuxDeploymentsFilterQuery
	deleteAuxDeploymentQuery
}

func getAuxDeploymentsH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_model.AuxDeploymentsPath, func(gc *gin.Context) {
		var query getAuxDeploymentsQuery
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		filter := lib_model.AuxDepFilter{
			IDs:     util.ParseStringSlice(query.IDs, ","),
			Labels:  util.GenLabels(util.ParseStringSlice(query.Labels, ",")),
			Image:   query.Image,
			Enabled: query.Enabled,
		}
		auxDeployments, err := a.GetAuxDeployments(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), filter, query.Assets, query.ContainerInfo)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, auxDeployments)
	}
}

func getAuxDeploymentH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, path.Join(lib_model.AuxDeploymentsPath, ":id"), func(gc *gin.Context) {
		var query getAuxDeploymentQuery
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		auxDeployment, err := a.GetAuxDeployment(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), gc.Param("id"), query.Assets, query.ContainerInfo)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, auxDeployment)
	}
}

func postAuxDeploymentH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_model.AuxDeploymentsPath, func(gc *gin.Context) {
		var query createAuxDeploymentQuery
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		var auxDepReq lib_model.AuxDepReq
		err := gc.ShouldBindJSON(&auxDepReq)
		if err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		id, err := a.CreateAuxDeployment(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), auxDepReq, query.ForcePullImg)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, id)
	}
}

func patchAuxDeploymentUpdateH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.AuxDeploymentsPath, ":id"), func(gc *gin.Context) {
		var query updateAuxDeploymentQuery
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		var auxDepReq lib_model.AuxDepReq
		err := gc.ShouldBindJSON(&auxDepReq)
		if err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.UpdateAuxDeployment(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), gc.Param("id"), auxDepReq, query.Incremental, query.ForcePullImg)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func deleteAuxDeploymentH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, path.Join(lib_model.AuxDeploymentsPath, ":id"), func(gc *gin.Context) {
		var query deleteAuxDeploymentQuery
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.DeleteAuxDeployment(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), gc.Param("id"), query.Force)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchAuxDeploymentsDeleteH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.AuxDepBatchPath, lib_model.DepDeletePath), func(gc *gin.Context) {
		var query deleteAuxDeploymentsQuery
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		filter := lib_model.AuxDepFilter{
			IDs:     util.ParseStringSlice(query.IDs, ","),
			Labels:  util.GenLabels(util.ParseStringSlice(query.Labels, ",")),
			Image:   query.Image,
			Enabled: query.Enabled,
		}
		jID, err := a.DeleteAuxDeployments(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), filter, query.Force)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchAuxDeploymentStartH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.AuxDeploymentsPath, ":id", lib_model.DepStartPath), func(gc *gin.Context) {
		jID, err := a.StartAuxDeployment(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchAuxDeploymentsStartH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.AuxDepBatchPath, lib_model.DepStartPath), func(gc *gin.Context) {
		var query getAuxDeploymentsFilterQuery
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		filter := lib_model.AuxDepFilter{
			IDs:     util.ParseStringSlice(query.IDs, ","),
			Labels:  util.GenLabels(util.ParseStringSlice(query.Labels, ",")),
			Image:   query.Image,
			Enabled: query.Enabled,
		}
		jID, err := a.StartAuxDeployments(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchAuxDeploymentStopH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.AuxDeploymentsPath, ":id", lib_model.DepStopPath), func(gc *gin.Context) {
		jID, err := a.StopAuxDeployment(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchAuxDeploymentsStopH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.AuxDepBatchPath, lib_model.DepStopPath), func(gc *gin.Context) {
		var query getAuxDeploymentsFilterQuery
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		filter := lib_model.AuxDepFilter{
			IDs:     util.ParseStringSlice(query.IDs, ","),
			Labels:  util.GenLabels(util.ParseStringSlice(query.Labels, ",")),
			Image:   query.Image,
			Enabled: query.Enabled,
		}
		jID, err := a.StopAuxDeployments(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchAuxDeploymentRestartH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.AuxDeploymentsPath, ":id", lib_model.DepRestartPath), func(gc *gin.Context) {
		jID, err := a.RestartAuxDeployment(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchAuxDeploymentsRestartH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.AuxDepBatchPath, lib_model.DepRestartPath), func(gc *gin.Context) {
		var query getAuxDeploymentsFilterQuery
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		filter := lib_model.AuxDepFilter{
			IDs:     util.ParseStringSlice(query.IDs, ","),
			Labels:  util.GenLabels(util.ParseStringSlice(query.Labels, ",")),
			Image:   query.Image,
			Enabled: query.Enabled,
		}
		jID, err := a.RestartAuxDeployments(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}
