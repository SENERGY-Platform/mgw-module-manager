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

type modulesQuery struct {
	Name           string `form:"name"`
	Author         string `form:"author"`
	Type           string `form:"type"`
	DeploymentType string `form:"deployment_type"`
	Tags           string `form:"tags"`
}

type deleteModuleQuery struct {
	Force bool `form:"force"`
}

// getModulesH godoc
// @Summary Get modules
// @Description List installed modules.
// @Tags Modules
// @Produce	json
// @Param name query string false "filter by name"
// @Param author query string false "filter by author"
// @Param type query string false "filter by type"
// @Param deployment_type query string false "filter by deployment type"
// @Param tags query []string false "filter by tags" collectionFormat(csv)
// @Success	200 {object} map[string]lib_model.Module "modules"
// @Failure	400 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /modules [get]
func getModulesH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_model.ModulesPath, func(gc *gin.Context) {
		query := modulesQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		filter := lib_model.ModFilter{
			Name:           query.Name,
			Author:         query.Author,
			Type:           query.Type,
			DeploymentType: query.DeploymentType,
		}
		tags := util.ParseStringSlice(query.Tags, ",")
		if len(tags) > 0 {
			s := make(map[string]struct{})
			for _, i := range tags {
				s[i] = struct{}{}
			}
			filter.Tags = s
		}
		modules, err := a.GetModules(gc.Request.Context(), filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, modules)
	}
}

// getModuleH godoc
// @Summary Get module
// @Description Get an installed module.
// @Tags Modules
// @Produce	json
// @Param id path string true "module ID"
// @Success	200 {object} lib_model.Module "module"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /modules/{id} [get]
func getModuleH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, path.Join(lib_model.ModulesPath, ":id"), func(gc *gin.Context) {
		module, err := a.GetModule(gc.Request.Context(), gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, module)
	}
}

// postModuleH godoc
// @Summary Add module
// @Description Download a module.
// @Tags Modules
// @Accept json
// @Produce	plain
// @Param info body lib_model.ModAddRequest true "module info"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	409 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /modules [post]
func postModuleH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_model.ModulesPath, func(gc *gin.Context) {
		var modReq lib_model.ModAddRequest
		err := gc.ShouldBindJSON(&modReq)
		if err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.AddModule(gc.Request.Context(), modReq.ID, modReq.Version)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

// deleteModuleH godoc
// @Summary Delete module
// @Description Remove a module.
// @Tags Modules
// @Produce	plain
// @Param id path string true "module ID"
// @Param force query bool false "force remove"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	409 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /modules/{id} [delete]
func deleteModuleH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, path.Join(lib_model.ModulesPath, ":id"), func(gc *gin.Context) {
		query := deleteModuleQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.DeleteModule(gc.Request.Context(), gc.Param("id"), query.Force)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

// getModuleDeployTemplateH godoc
// @Summary Get deployment template
// @Description Get a template for the deployment of a module.
// @Tags Modules
// @Produce	json
// @Param id path string true "module ID"
// @Success	200 {object} lib_model.ModDeployTemplate "template"
// @Failure	404 {string} string "error message"
// @Failure	409 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /modules/{id}/dep-template [get]
func getModuleDeployTemplateH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, path.Join(lib_model.ModulesPath, ":id", lib_model.DepTemplatePath), func(gc *gin.Context) {
		inputTemplate, err := a.GetModuleDeployTemplate(gc.Request.Context(), gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, inputTemplate)
	}
}

// getModuleUpdatesH godoc
// @Summary Get module updates
// @Description List available module updates.
// @Tags Modules
// @Produce	json
// @Success	200 {object} map[string]lib_model.ModUpdate "module updates"
// @Failure	500 {string} string "error message"
// @Router /updates [get]
func getModuleUpdatesH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_model.ModUpdatesPath, func(gc *gin.Context) {
		updates, err := a.GetModuleUpdates(gc.Request.Context())
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, updates)
	}
}

// getModuleUpdateH godoc
// @Summary Get module update
// @Description Get module update info.
// @Tags Modules
// @Produce	json
// @Param id path string true "module ID"
// @Success	200 {object} lib_model.ModUpdate "update info"
// @Failure	404 {string} string "error message"
// @Failure	409 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /updates/{id} [get]
func getModuleUpdateH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, path.Join(lib_model.ModUpdatesPath, ":id"), func(gc *gin.Context) {
		update, err := a.GetModuleUpdate(gc.Request.Context(), gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, update)
	}
}

// postCheckModuleUpdatesH godoc
// @Summary Check module updates
// @Description Check for new module updates.
// @Tags Modules
// @Produce	plain
// @Success	200 {string} string "job ID"
// @Failure	409 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /updates [post]
func postCheckModuleUpdatesH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_model.ModUpdatesPath, func(gc *gin.Context) {
		jID, err := a.CheckModuleUpdates(gc.Request.Context())
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

// patchPrepareModuleUpdateH godoc
// @Summary Prepare module update
// @Description Checks dependencies, downloads and marks update as pending.
// @Tags Modules
// @Accept json
// @Produce	plain
// @Param id path string true "module ID"
// @Param info body lib_model.ModUpdatePrepareRequest true "module info"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	409 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /updates/{id}/prepare [patch]
func patchPrepareModuleUpdateH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.ModUpdatesPath, ":id", lib_model.ModUptPreparePath), func(gc *gin.Context) {
		var modUptReq lib_model.ModUpdatePrepareRequest
		err := gc.ShouldBindJSON(&modUptReq)
		if err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.PrepareModuleUpdate(gc.Request.Context(), gc.Param("id"), modUptReq.Version)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

// patchCancelPendingModuleUpdateH godoc
// @Summary Cancel module update
// @Description Cancel a pending module update.
// @Tags Modules
// @Param id path string true "module ID"
// @Success	200
// @Failure	404 {string} string "error message"
// @Failure	409 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /updates/{id}/cancel [patch]
func patchCancelPendingModuleUpdateH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.ModUpdatesPath, ":id", lib_model.ModUptCancelPath), func(gc *gin.Context) {
		err := a.CancelPendingModuleUpdate(gc.Request.Context(), gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

// patchModuleUpdateH godoc
// @Summary Update module
// @Description Execute a pending module update. Dependencies and existing deployments will also be updated.
// @Tags Modules
// @Accept json
// @Produce	plain
// @Param id path string true "module ID"
// @Param data body lib_model.ModUpdateRequest true "update data"
// @Success	200 {string} string "job ID"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	409 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /updates/{id} [patch]
func patchModuleUpdateH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.ModUpdatesPath, ":id"), func(gc *gin.Context) {
		var uptReq lib_model.ModUpdateRequest
		err := gc.ShouldBindJSON(&uptReq)
		if err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jID, err := a.UpdateModule(gc.Request.Context(), gc.Param("id"), uptReq.DepInput, uptReq.Dependencies)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

// getPendingModuleUpdateTemplateH godoc
// @Summary Get module update template
// @Description Get update template for pending module update.
// @Tags Modules
// @Produce	json
// @Param id path string true "module ID"
// @Success	200 {object} lib_model.ModUpdateTemplate "template"
// @Failure	400 {string} string "error message"
// @Failure	404 {string} string "error message"
// @Failure	409 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /updates/{id}/upt-template [get]
func getPendingModuleUpdateTemplateH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, path.Join(lib_model.ModUpdatesPath, ":id", lib_model.DepUpdateTemplatePath), func(gc *gin.Context) {
		updateTemplate, err := a.GetModuleUpdateTemplate(gc.Request.Context(), gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, updateTemplate)
	}
}
