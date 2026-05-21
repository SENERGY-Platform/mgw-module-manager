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

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-gonic/gin"
)

func GetDeploymentsJobResult(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "results/deployments/:job_id", func(gc *gin.Context) {
		res, err := srv.GetDeploymentsJobResult(gc, gc.Param("job_id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetUpdateDeploymentsJobResult(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "results/deployments-update/:job_id", func(gc *gin.Context) {
		res, err := srv.GetUpdateDeploymentsJobResult(gc, gc.Param("job_id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetModuleChangeJobResult(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "results/modules-change/:job_id", func(gc *gin.Context) {
		res, err := srv.GetModuleChangeJobResult(gc, gc.Param("job_id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetRefreshRepositoriesJobResult(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "results/repositories-refresh/:job_id", func(gc *gin.Context) {
		res, err := srv.GetRefreshRepositoriesJobResult(gc, gc.Param("job_id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetCreateAuxiliaryDeploymentJobResult(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "results/auxiliary-deployment-create/:job_id", func(gc *gin.Context) {
		res, err := srv.GetCreateAuxiliaryDeploymentJobResult(gc, gc.Param("job_id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetUpdateAuxiliaryDeploymentJobResult(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "results/auxiliary-deployment-update/:job_id", func(gc *gin.Context) {
		res, err := srv.GetUpdateAuxiliaryDeploymentJobResult(gc, gc.Param("job_id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetAuxiliaryDeploymentsJobResult(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "results/auxiliary-deployments/:job_id", func(gc *gin.Context) {
		res, err := srv.GetAuxiliaryDeploymentsJobResult(gc, gc.Param("job_id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}
