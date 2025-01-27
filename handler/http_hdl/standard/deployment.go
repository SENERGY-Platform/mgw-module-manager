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

// getDeploymentsH godoc
// @Summary Get Deployments
// @Description Get all deployments.
// @Tags Deployments
// @Produce	json
// @Param ids query []string false "filter by deployment ids" collectionFormat(csv)
// @Param name query string false "filter by name"
// @Param module_id query string false "filter by module ID"
// @Param enabled query string false "filter by enabled status"
// @Param indirect query string false "filter by indirect status"
// @Param assets query bool false "include assets"
// @Param container_info query bool false "include container info"
// @Success	200 {object} map[string]lib_model.Deployment "deployments"
// @Failure	400 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /deployments [get]
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

// getDeploymentH godoc
// @Summary Get deployment
// @Description Get a deployment.
// @Tags Deployments
// @Produce	json
// @Param id path string true "deployment ID"
// @Param assets query bool false "include assets"
// @Param container_info query bool false "include container info"
// @Success	200 {object} lib_model.Deployment "deployment"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /deployments/{id} [get]
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

// postDeploymentH godoc
// @Summary Create deployment
// @Description Create a new deployment.
// @Tags Deployments
// @Accept json
// @Produce	plain
// @Param data body lib_model.DepCreateRequest true "deployment data"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /deployments [post]
func postDeploymentH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_model.DeploymentsPath, func(gc *gin.Context) {
		var depReq lib_model.DepCreateRequest
		err := gc.ShouldBindJSON(&depReq)
		if err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.CreateDeployment(gc.Request.Context(), depReq.ModuleID, depReq.DepInput, depReq.Dependencies)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

// patchDeploymentUpdateH godoc
// @Summary Update deployment
// @Description Update a deployment.
// @Tags Deployments
// @Accept json
// @Produce	plain
// @Param id path string true "deployment ID"
// @Param data body lib_model.DepInput true "deployment data"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /deployments/{id} [patch]
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

// patchDeploymentStartH godoc
// @Summary Start deployment
// @Description Start a deployment.
// @Tags Deployments
// @Produce	plain
// @Param id path string true "deployment ID"
// @Param dependencies query bool false "toggle start dependencies"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /deployments/{id}/start [patch]
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

// patchDeploymentsStartH godoc
// @Summary Start deployments
// @Description Start multiple deployments.
// @Tags Deployments
// @Produce	plain
// @Param ids query []string false "filter by deployment ids" collectionFormat(csv)
// @Param name query string false "filter by name"
// @Param module_id query string false "filter by module ID"
// @Param enabled query string false "filter by enabled status"
// @Param indirect query string false "filter by indirect status"
// @Param dependencies query bool false "toggle start dependencies"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /deployments-batch/start [patch]
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

// patchDeploymentStopH godoc
// @Summary Stop deployment
// @Description Stop a deployment.
// @Tags Deployments
// @Produce	plain
// @Param id path string true "deployment ID"
// @Param force query bool false "toggle force stop"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /deployments/{id}/stop [patch]
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

// patchDeploymentsStopH godoc
// @Summary Stop deployments
// @Description Stop multiple deployments.
// @Tags Deployments
// @Produce	plain
// @Param ids query []string false "filter by deployment ids" collectionFormat(csv)
// @Param name query string false "filter by name"
// @Param module_id query string false "filter by module ID"
// @Param enabled query string false "filter by enabled status"
// @Param indirect query string false "filter by indirect status"
// @Param force query bool false "toggle force stop"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /deployments-batch/stop [patch]
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

// patchDeploymentRestartH godoc
// @Summary Restart deployment
// @Description Restart a deployment.
// @Tags Deployments
// @Produce	plain
// @Param id path string true "deployment ID"
// @Success	200 {string} string "job ID"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /deployments/{id}/restart [patch]
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

// patchDeploymentsRestartH godoc
// @Summary Restart deployments
// @Description Restart multiple deployments.
// @Tags Deployments
// @Produce	plain
// @Param ids query []string false "filter by deployment ids" collectionFormat(csv)
// @Param name query string false "filter by name"
// @Param module_id query string false "filter by module ID"
// @Param enabled query string false "filter by enabled status"
// @Param indirect query string false "filter by indirect status"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /deployments-batch/restart [patch]
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

// deleteDeploymentH godoc
// @Summary Delete deployment
// @Description Delete a deployment.
// @Tags Deployments
// @Produce	plain
// @Param id path string true "deployment ID"
// @Param force query bool false "toggle force delete"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /deployments/{id} [delete]
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

// patchDeploymentsDeleteH godoc
// @Summary Delete deployments
// @Description Delete multiple deployments.
// @Tags Deployments
// @Produce	plain
// @Param ids query []string false "filter by deployment ids" collectionFormat(csv)
// @Param name query string false "filter by name"
// @Param module_id query string false "filter by module ID"
// @Param enabled query string false "filter by enabled status"
// @Param indirect query string false "filter by indirect status"
// @Param force query bool false "toggle force delete"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /deployments-batch/delete [patch]
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

// getDeploymentUpdateTemplateH godoc
// @Summary Get update template
// @Description Get a template for updating a deployment.
// @Tags Deployments
// @Produce	json
// @Param id path string true "deployment ID"
// @Success	200 {object} lib_model.DepUpdateTemplate "template"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /deployments/{id}/upt-template [get]
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
