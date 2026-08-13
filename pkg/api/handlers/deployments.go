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

// @Summary		Get deployment request
// @Description	Get the modules that must be configured to deploy the given modules including their dependencies.
// @Tags		deployments
// @Produce		json
// @Param		module_ids	query	[]string	false	"filter by module IDs"	collectionFormat(csv)
// @Success		200	{array}	lib_models.Module	"modules to be configured"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/deployment-request [get]
func GetDeploymentRequest(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathDeploymentRequestResource, func(gc *gin.Context) {
		var query struct {
			ModuleIds []string `form:"module_ids" collection_format:"csv"`
		}
		err := gc.MustBindWith(&query, binding.Query)
		if err != nil {
			return
		}
		res, err := srv.GetDeploymentRequest(gc, query.ModuleIds)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Create deployments
// @Description	Create deployments for the given modules as a job.
// @Tags		deployments
// @Accept		json
// @Produce		json
// @Param		request	body	[]lib_models.DeploymentUserInput	true	"deployment user inputs"
// @Success		200	{object}	lib_models.Job	"job"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/deployments [post]
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

// @Summary		Update deployments
// @Description	Update the deployments of the given modules as a job.
// @Tags		deployments
// @Accept		json
// @Produce		json
// @Param		request	body	[]lib_models.DeploymentUserInput	true	"deployment user inputs"
// @Success		200	{object}	lib_models.Job	"job"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/deployments [put]
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

// @Summary		Recreate deployments
// @Description	Recreate the deployments of the given modules as a job.
// @Tags		deployments
// @Accept		json
// @Produce		json
// @Param		request	body	[]string	true	"module IDs"
// @Success		200	{object}	lib_models.Job	"job"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/deployments-recreate [post]
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

// @Summary		Delete deployments
// @Description	Delete the deployments of the given modules as a job. All deployments are deleted if allow_all is set and no module IDs are given.
// @Tags		deployments
// @Produce		json
// @Param		module_ids	query	[]string	false	"filter by module IDs"	collectionFormat(csv)
// @Param		allow_all	query	bool	false	"allow deletion of all deployments"
// @Success		200	{object}	lib_models.Job	"job"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/deployments [delete]
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

// @Summary		Enable deployments
// @Description	Enable the deployments of the given modules.
// @Tags		deployments
// @Accept		json
// @Produce		json
// @Param		request	body	[]string	true	"module IDs"
// @Success		200	{array}	string	"IDs of the enabled deployments"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/deployments-enable [post]
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

// @Summary		Disable deployments
// @Description	Disable the deployments of the given modules.
// @Tags		deployments
// @Accept		json
// @Produce		json
// @Param		request	body	[]string	true	"module IDs"
// @Success		200	{array}	string	"IDs of the disabled deployments"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/deployments-disable [post]
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
