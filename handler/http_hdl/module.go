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

const modIdParam = "m"

type modulesQuery struct {
	Name           string   `form:"name"`
	Author         string   `form:"author"`
	Type           string   `form:"type"`
	DeploymentType string   `form:"deployment_type"`
	InDependencies []string `form:"in_dependency"`
	Tags           []string `form:"tag"`
}

type deleteModuleQuery struct {
	Orphans bool `form:"orphans"`
	Force   bool `form:"force"`
}

type updateModuleQuery struct {
	Orphans bool `form:"orphans"`
}

func getModulesH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		query := modulesQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(model.NewInvalidInputError(err))
			return
		}
		filter := model.ModFilter{
			Name:           query.Name,
			Author:         query.Author,
			Type:           query.Type,
			DeploymentType: query.DeploymentType,
		}
		if len(query.InDependencies) > 0 {
			s := make(map[string]struct{})
			for _, i := range query.InDependencies {
				s[i] = struct{}{}
			}
			filter.InDependencies = s
		}
		if len(query.Tags) > 0 {
			s := make(map[string]struct{})
			for _, i := range query.Tags {
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

func getModuleH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		module, err := a.GetModule(gc.Request.Context(), gc.Param(modIdParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, module)
	}
}

func postModuleH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		var modReq model.ModAddRequest
		err := gc.ShouldBindJSON(&modReq)
		if err != nil {
			_ = gc.Error(model.NewInvalidInputError(err))
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

func deleteModuleH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		query := deleteModuleQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(model.NewInvalidInputError(err))
			return
		}
		err := a.DeleteModule(gc.Request.Context(), gc.Param(modIdParam), query.Orphans, query.Force)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func getModuleDeployTemplateH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		inputTemplate, err := a.GetModuleDeployTemplate(gc.Request.Context(), gc.Param(modIdParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, inputTemplate)
	}
}

func getModuleUpdatesH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		updates := a.GetModuleUpdates(gc.Request.Context())
		gc.JSON(http.StatusOK, updates)
	}
}

func getModuleUpdateH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		update, err := a.GetModuleUpdate(gc.Request.Context(), gc.Param(modIdParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, update)
	}
}

func postCheckModuleUpdatesH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		jID, err := a.CheckModuleUpdates(gc.Request.Context())
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchPrepareModuleUpdateH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		var modUptReq model.ModUpdatePrepareRequest
		err := gc.ShouldBindJSON(&modUptReq)
		if err != nil {
			_ = gc.Error(model.NewInvalidInputError(err))
			return
		}
		jID, err := a.PrepareModuleUpdate(gc.Request.Context(), gc.Param(modIdParam), modUptReq.Version)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, jID)
	}
}

func patchCancelPendingModuleUpdateH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		err := a.CancelPendingModuleUpdate(gc.Request.Context(), gc.Param(modIdParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func patchModuleUpdateH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		var uptReq model.ModUpdateRequest
		err := gc.ShouldBindJSON(&uptReq)
		if err != nil {
			_ = gc.Error(model.NewInvalidInputError(err))
			return
		}
		query := updateModuleQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(model.NewInvalidInputError(err))
			return
		}
		id, err := a.UpdateModule(gc.Request.Context(), gc.Param(modIdParam), uptReq.DepInput, uptReq.Dependencies, query.Orphans)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.String(http.StatusOK, id)
	}
}

func getPendingModuleUpdateTemplateH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		updateTemplate, err := a.GetModuleUpdateTemplate(gc.Request.Context(), gc.Param(modIdParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, updateTemplate)
	}
}
