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

// getAuxDeploymentsH godoc
// @Summary Get auxiliary deployments
// @Description List auxiliary deployments for the current deployment.
// @Tags Auxiliary Deployment
// @Produce	json
// @Param X-MGW-DID header string true "deployment ID"
// @Param ids query []string false "filter by aux deployment ids" collectionFormat(csv)
// @Param labels query string false "filter by labels (e.g.: k1=v1,k2=v2,k3)"
// @Param image query string false "filter by image"
// @Param enabled query integer false "filter if enabled" Enums(-1, 1)
// @Param assets query bool false "include assets"
// @Param container_info query bool false "include container info"
// @Success	200 {object} map[string]lib_model.AuxDeployment "auxiliary deployments"
// @Failure	400 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /aux-deployments [get]
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

// getAuxDeploymentH godoc
// @Summary Get auxiliary deployment
// @Description Get an auxiliary deployment for the current deployment.
// @Tags Auxiliary Deployment
// @Produce	json
// @Param X-MGW-DID header string true "deployment ID"
// @Param id path string true "aux deployment ID"
// @Param assets query bool false "include assets"
// @Param container_info query bool false "include container info"
// @Success	200 {object} lib_model.AuxDeployment "auxiliary deployment"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /aux-deployments/{id} [get]
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

// postAuxDeploymentH godoc
// @Summary Create auxiliary deployment
// @Description Create a new auxiliary deployment for the current deployment.
// @Tags Auxiliary Deployment
// @Accept json
// @Produce	plain
// @Param X-MGW-DID header string true "deployment ID"
// @Param force_pull_img query bool false "force pull image"
// @Param data body lib_model.AuxDepReq true "aux deployment data"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /aux-deployments [post]
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

// patchAuxDeploymentUpdateH godoc
// @Summary Update auxiliary deployment
// @Description Update an auxiliary deployment for the current deployment.
// @Tags Auxiliary Deployment
// @Accept json
// @Produce	plain
// @Param X-MGW-DID header string true "deployment ID"
// @Param id path string true "aux deployment ID"
// @Param incremental query bool false "do an incremental update"
// @Param force_pull_img query bool false "force pull image"
// @Param data body lib_model.AuxDepReq true "aux deployment data"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /aux-deployments/{id} [patch]
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

// deleteAuxDeploymentH godoc
// @Summary Delete auxiliary deployment
// @Description Remove an auxiliary deployment for the current deployment.
// @Tags Auxiliary Deployment
// @Produce	plain
// @Param X-MGW-DID header string true "deployment ID"
// @Param id path string true "aux deployment ID"
// @Param force query bool false "force delete"
// @Success	200 {string} string " job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /aux-deployments/{id} [delete]
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

// patchAuxDeploymentsDeleteH godoc
// @Summary Delete auxiliary deployments
// @Description Remove auxiliary deployments for the current deployment.
// @Tags Auxiliary Deployment
// @Produce	plain
// @Param X-MGW-DID header string true "deployment ID"
// @Param ids query []string false "filter by aux deployment ids" collectionFormat(csv)
// @Param labels query string false "filter by labels (e.g.: k1=v1,k2=v2,k3)"
// @Param image query string false "filter by image"
// @Param enabled query integer false "filter if enabled" Enums(-1, 1)
// @Param force query bool false "force delete"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /aux-deployments-batch/delete [patch]
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

// patchAuxDeploymentStartH godoc
// @Summary Start auxiliary deployment
// @Description Start an auxiliary deployment for the current deployment.
// @Tags Auxiliary Deployment
// @Produce	plain
// @Param X-MGW-DID header string true "deployment ID"
// @Param id path string true "aux deployment ID"
// @Success	200 {string} string "job ID"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /aux-deployments/{id}/start [patch]
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

// patchAuxDeploymentsStartH godoc
// @Summary Start auxiliary deployments
// @Description Start auxiliary deployments for the current deployment.
// @Tags Auxiliary Deployment
// @Produce	plain
// @Param X-MGW-DID header string true "deployment ID"
// @Param ids query []string false "filter by aux deployment ids" collectionFormat(csv)
// @Param labels query string false "filter by labels (e.g.: k1=v1,k2=v2,k3)"
// @Param image query string false "filter by image"
// @Param enabled query integer false "filter if enabled" Enums(-1, 1)
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /aux-deployments-batch/start [patch]
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

// patchAuxDeploymentStopH godoc
// @Summary Stop auxiliary deployment
// @Description Stop an auxiliary deployment for the current deployment.
// @Tags Auxiliary Deployment
// @Produce	plain
// @Param X-MGW-DID header string true "deployment ID"
// @Param id path string true "aux deployment ID"
// @Success	200 {string} string "job ID"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /aux-deployments/{id}/stop [patch]
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

// patchAuxDeploymentsStopH godoc
// @Summary Stop auxiliary deployments
// @Description Stop auxiliary deployments for the current deployment.
// @Tags Auxiliary Deployment
// @Produce	plain
// @Param X-MGW-DID header string true "deployment ID"
// @Param ids query []string false "filter by aux deployment ids" collectionFormat(csv)
// @Param labels query string false "filter by labels (e.g.: k1=v1,k2=v2,k3)"
// @Param image query string false "filter by image"
// @Param enabled query integer false "filter if enabled" Enums(-1, 1)
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /aux-deployments-batch/stop [patch]
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

// patchAuxDeploymentRestartH godoc
// @Summary Restart auxiliary deployment
// @Description Restart an auxiliary deployment for the current deployment.
// @Tags Auxiliary Deployment
// @Produce	plain
// @Param X-MGW-DID header string true "deployment ID"
// @Param id path string true "aux deployment ID"
// @Success	200 {string} string "job ID"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /aux-deployments/{id}/restart [patch]
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

// patchAuxDeploymentsRestartH godoc
// @Summary Restart auxiliary deployments
// @Description Restart auxiliary deployments for the current deployment.
// @Tags Auxiliary Deployment
// @Produce	plain
// @Param X-MGW-DID header string true "deployment ID"
// @Param ids query []string false "filter by aux deployment ids" collectionFormat(csv)
// @Param labels query string false "filter by labels (e.g.: k1=v1,k2=v2,k3)"
// @Param image query string false "filter by image"
// @Param enabled query integer false "filter if enabled" Enums(-1, 1)
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /aux-deployments-batch/restart [patch]
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
