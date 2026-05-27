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

package handlers

import (
	"net/http"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func GetDeploymentRequest(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathDeploymentRequestResource, func(gc *gin.Context) {
		var body []string
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.GetDeploymentRequest(gc, body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func CreateDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_constants.HttpPathDeploymentsCollection, func(gc *gin.Context) {
		var body []lib_models.DeploymentUserInput
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.CreateDeployments(gc, body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func UpdateDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPut, lib_constants.HttpPathDeploymentsCollection, func(gc *gin.Context) {
		var body []lib_models.DeploymentUserInput
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.UpdateDeployments(gc, body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func RecreateDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_constants.HttpPathRecreateDeployments, func(gc *gin.Context) {
		var body []string
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.RecreateDeployments(gc, body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func DeleteDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, lib_constants.HttpPathDeploymentsCollection, func(gc *gin.Context) {
		var query struct {
			ModuleIds []string `form:"module_ids" collection_format:"csv"`
			AllowAll  bool     `form:"allow_all"`
		}
		err := gc.MustBindWith(&query, binding.Query)
		if err != nil {
			return
		}
		res, err := srv.DeleteDeployments(gc, query.ModuleIds, query.AllowAll)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func EnableDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_constants.HttpPathEnableDeployments, func(gc *gin.Context) {
		var body []string
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.EnableDeployments(gc, body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func DisableDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_constants.HttpPathDisableDeployments, func(gc *gin.Context) {
		var body []string
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.DisableDeployments(gc, body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}
