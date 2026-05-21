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

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func GetDeploymentRequest(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, "deployment-request", func(gc *gin.Context) {
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
	return http.MethodPost, "deployments", func(gc *gin.Context) {
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
	return http.MethodPut, "deployments", func(gc *gin.Context) {
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
	return http.MethodPost, "deployments-recreate", func(gc *gin.Context) {
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

func getDeleteDeploymentsFilter(gc *gin.Context) ([]string, bool, error) {
	var query struct {
		ModuleIds []string `form:"module_ids" collection_format:"csv"`
		AllowAll  bool     `form:"allow_all"`
	}
	err := gc.MustBindWith(&query, binding.Query)
	if err != nil {
		return nil, false, err
	}
	return query.ModuleIds, query.AllowAll, err
}

func DeleteDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, "deployments", func(gc *gin.Context) {
		moduleIds, allowAll, err := getDeleteDeploymentsFilter(gc)
		if err != nil {
			return
		}
		res, err := srv.DeleteDeployments(gc, moduleIds, allowAll)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func EnableDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, "deployments-enable", func(gc *gin.Context) {
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
	return http.MethodPost, "deployments-disable", func(gc *gin.Context) {
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
