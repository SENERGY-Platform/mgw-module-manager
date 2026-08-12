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
	_ "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-gonic/gin"
)

// @Summary		Get deployments job result
// @Description	Get the result of a job created by deploying modules.
// @Tags		job-results
// @Produce		json
// @Param		JOB_ID	path	string	true	"job ID"
// @Success		200	{object}	models.DeploymentJobResult	"deployments job result"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/results/deployments/{JOB_ID} [get]
func GetDeploymentsJobResult(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathDeploymentResultResource, func(gc *gin.Context) {
		res, err := srv.GetDeploymentsJobResult(gc, gc.Param("JOB_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Get update deployments job result
// @Description	Get the result of a job created by updating deployments.
// @Tags		job-results
// @Produce		json
// @Param		JOB_ID	path	string	true	"job ID"
// @Success		200	{object}	models.DeploymentUpdateJobResult	"update deployments job result"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/results/deployments-update/{JOB_ID} [get]
func GetUpdateDeploymentsJobResult(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathUpdateDeploymentResultResource, func(gc *gin.Context) {
		res, err := srv.GetUpdateDeploymentsJobResult(gc, gc.Param("JOB_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Get delete deployments job result
// @Description	Get the result of a job created by deleting deployments.
// @Tags		job-results
// @Produce		json
// @Param		JOB_ID	path	string	true	"job ID"
// @Success		200	{object}	models.DeploymentDeleteJobResult	"delete deployments job result"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/results/deployments-delete/{JOB_ID} [get]
func GetDeleteDeploymentsJobResult(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathDeleteDeploymentResultResource, func(gc *gin.Context) {
		res, err := srv.GetDeleteDeploymentsJobResult(gc, gc.Param("JOB_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Get modules change job result
// @Description	Get the result of a job created by executing a modules change request.
// @Tags		job-results
// @Produce		json
// @Param		JOB_ID	path	string	true	"job ID"
// @Success		200	{object}	models.ModulesChangeJobResult	"modules change job result"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/results/modules-change/{JOB_ID} [get]
func GetModuleChangeJobResult(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathChangeModulesResultResource, func(gc *gin.Context) {
		res, err := srv.GetModuleChangeJobResult(gc, gc.Param("JOB_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Get refresh repositories job result
// @Description	Get the result of a job created by refreshing repositories.
// @Tags		job-results
// @Produce		json
// @Param		JOB_ID	path	string	true	"job ID"
// @Success		200	{object}	models.RepositoryJobResult	"refresh repositories job result"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/results/repositories-refresh/{JOB_ID} [get]
func GetRefreshRepositoriesJobResult(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathRefreshRepositoriesResultResource, func(gc *gin.Context) {
		res, err := srv.GetRefreshRepositoriesJobResult(gc, gc.Param("JOB_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Get create auxiliary deployment job result
// @Description	Get the result of a job created by creating an auxiliary deployment.
// @Tags		job-results
// @Produce		json
// @Param		JOB_ID	path	string	true	"job ID"
// @Success		200	{object}	models.AuxiliaryDeploymentCreateJobResult	"create auxiliary deployment job result"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/results/auxiliary-deployment-create/{JOB_ID} [get]
// @Router		/restricted/results/auxiliary-deployment-create/{JOB_ID} [get]
func GetCreateAuxiliaryDeploymentJobResult(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathCreateAuxiliaryDeploymentResultResource, func(gc *gin.Context) {
		res, err := srv.GetCreateAuxiliaryDeploymentJobResult(gc, gc.Param("JOB_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Get update auxiliary deployment job result
// @Description	Get the result of a job created by updating an auxiliary deployment.
// @Tags		job-results
// @Produce		json
// @Param		JOB_ID	path	string	true	"job ID"
// @Success		200	{object}	models.JobResult	"update auxiliary deployment job result"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/results/auxiliary-deployment-update/{JOB_ID} [get]
// @Router		/restricted/results/auxiliary-deployment-update/{JOB_ID} [get]
func GetUpdateAuxiliaryDeploymentJobResult(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathUpdateAuxiliaryDeploymentResultResource, func(gc *gin.Context) {
		res, err := srv.GetUpdateAuxiliaryDeploymentJobResult(gc, gc.Param("JOB_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Get auxiliary deployments job result
// @Description	Get the result of a job created by recreating, deleting, enabling or disabling auxiliary deployments.
// @Tags		job-results
// @Produce		json
// @Param		JOB_ID	path	string	true	"job ID"
// @Success		200	{object}	models.AuxiliaryDeploymentJobResult	"auxiliary deployments job result"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/results/auxiliary-deployments/{JOB_ID} [get]
// @Router		/restricted/results/auxiliary-deployments/{JOB_ID} [get]
func GetAuxiliaryDeploymentsJobResult(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathAuxiliaryDeploymentsResultResource, func(gc *gin.Context) {
		res, err := srv.GetAuxiliaryDeploymentsJobResult(gc, gc.Param("JOB_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}
