/*
 * Copyright 2026 InfAI (CC SES)
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

package handler

import (
	"net/http"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func getModulesFilter(gc *gin.Context) (lib_models.ModulesFilter, error) {
	var query struct {
		Ids               []string `form:"ids" collection_format:"csv"`
		Name              string   `form:"name"`
		Tags              []string `form:"tags" collection_format:"csv"`
		Author            string   `form:"author"`
		IsDeployed        int      `form:"is_deployed"`
		DeploymentEnabled int      `form:"deployment_enabled"`
		DeploymentState   int      `form:"deployment_state"`
	}
	err := gc.MustBindWith(&query, binding.Query)
	if err != nil {
		return lib_models.ModulesFilter{}, err
	}
	return lib_models.ModulesFilter{
		Ids:               query.Ids,
		Name:              query.Name,
		Tags:              query.Tags,
		Author:            query.Author,
		IsDeployed:        query.IsDeployed,
		DeploymentEnabled: query.DeploymentEnabled,
		DeploymentState:   query.DeploymentState,
	}, nil
}

func GetModules(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "modules", func(gc *gin.Context) {
		filter, err := getModulesFilter(gc)
		if err != nil {
			return
		}
		res, err := srv.GetModules(gc, filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetModule(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "modules/:id", func(gc *gin.Context) {
		res, err := srv.GetModule(gc, gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetModulesChangeRequest(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "modules-change-request", func(gc *gin.Context) {
		res, err := srv.GetModulesChangeRequest(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func getCreateModulesChangeRequestUpdateAll(gc *gin.Context) (bool, error) {
	var query struct {
		UpdateAll bool `form:"update_all"`
	}
	err := gc.MustBindWith(&query, binding.Query)
	if err != nil {
		return false, err
	}
	return query.UpdateAll, nil
}

func CreateModulesChangeRequest(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, "modules-change-request", func(gc *gin.Context) {
		updateAll, err := getCreateModulesChangeRequestUpdateAll(gc)
		if err != nil {
			return
		}
		var res lib_models.ModulesChangeRequest
		if updateAll {
			res, err = srv.CreateModulesUpdateAllChangeRequest(gc)
			if err != nil {
				_ = gc.Error(err)
				return
			}
		} else {
			var body []lib_models.ChangeRequestItem
			err = gc.MustBindWith(&body, binding.JSON)
			if err != nil {
				return
			}
			res, err = srv.CreateModulesChangeRequest(gc, body)
			if err != nil {
				_ = gc.Error(err)
				return
			}
		}
		gc.JSON(http.StatusOK, res)
	}
}

func ExecModulesChangeRequest(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, "modules-change-request", func(gc *gin.Context) {
		res, err := srv.ExecModulesChangeRequest(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func CancelModulesChangeRequest(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, "modules-change-request", func(gc *gin.Context) {
		err := srv.CancelModulesChangeRequest(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func GetModulesAvailableUpdatesCount(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "modules-available-updates", func(gc *gin.Context) {
		res, err := srv.GetModulesAvailableUpdatesCount(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}
