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

// @Summary		Get reduced modules
// @Description	Get installed modules with a reduced set of fields.
// @Tags		modules
// @Produce		json
// @Param		ids	query	[]string	false	"filter by module IDs"	collectionFormat(csv)
// @Param		name	query	string	false	"filter by module name"
// @Param		tags	query	[]string	false	"filter by module tags"	collectionFormat(csv)
// @Param		author	query	string	false	"filter by module author"
// @Param		is_deployed	query	int	false	"filter by deployment existence, values: -1,0,1"
// @Param		deployment_enabled	query	int	false	"filter by deployment enabled state, values: -1,0,1"
// @Param		deployment_state	query	int	false	"filter by deployment state, values: -1,0,1"
// @Success		200	{array}	lib_models.ModuleReduced	"reduced modules"
// @Failure		400	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/modules-reduced [get]
func GetReducedModules(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathReducedModulesCollection, func(gc *gin.Context) {
		filter, err := getModulesFilter(gc)
		if err != nil {
			return
		}
		res, err := srv.GetReducedModules(gc, filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Get modules
// @Description	Get installed modules.
// @Tags		modules
// @Produce		json
// @Param		ids	query	[]string	false	"filter by module IDs"	collectionFormat(csv)
// @Param		name	query	string	false	"filter by module name"
// @Param		tags	query	[]string	false	"filter by module tags"	collectionFormat(csv)
// @Param		author	query	string	false	"filter by module author"
// @Param		is_deployed	query	int	false	"filter by deployment existence, values: -1,0,1"
// @Param		deployment_enabled	query	int	false	"filter by deployment enabled state, values: -1,0,1"
// @Param		deployment_state	query	int	false	"filter by deployment state, values: -1,0,1"
// @Success		200	{array}	lib_models.Module	"modules"
// @Failure		400	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/modules [get]
func GetModules(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathModulesCollection, func(gc *gin.Context) {
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

// @Summary		Get module
// @Description	Get a single installed module by its ID.
// @Tags		modules
// @Produce		json
// @Param		MOD_ID	path	string	true	"module ID"
// @Success		200	{object}	lib_models.Module	"module"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/modules/{MOD_ID} [get]
func GetModule(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathModuleResource, func(gc *gin.Context) {
		res, err := srv.GetModule(gc, gc.Param("MOD_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Get modules change request
// @Description	Get the currently pending modules change request.
// @Tags		modules
// @Produce		json
// @Success		200	{object}	lib_models.ModulesChangeRequest	"modules change request"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/modules-change-request [get]
func GetModulesChangeRequest(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathModulesChangeRequestResource, func(gc *gin.Context) {
		res, err := srv.GetModulesChangeRequest(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Create modules change request
// @Description	Create a modules change request for the given items or, if update_all is set, for all available module updates.
// @Tags		modules
// @Accept		json
// @Produce		json
// @Param		update_all	query	bool	false	"create a change request for all available module updates"
// @Param		request	body	[]lib_models.ChangeRequestItem	false	"change request items (ignored if update_all is set)"
// @Success		200	{object}	lib_models.ModulesChangeRequest	"modules change request"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/modules-change-request [post]
func CreateModulesChangeRequest(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_constants.HttpPathModulesChangeRequestResource, func(gc *gin.Context) {
		var query struct {
			UpdateAll bool `form:"update_all"`
		}
		err := gc.MustBindWith(&query, binding.Query)
		if err != nil {
			return
		}
		var res lib_models.ModulesChangeRequest
		if query.UpdateAll {
			res, err = srv.CreateModulesUpdateAllChangeRequest(gc)
		} else {
			var body []lib_models.ChangeRequestItem
			err = gc.MustBindWith(&body, binding.JSON)
			if err != nil {
				return
			}
			res, err = srv.CreateModulesChangeRequest(gc, body)
		}
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Execute modules change request
// @Description	Execute the currently pending modules change request as a job.
// @Tags		modules
// @Produce		json
// @Success		200	{object}	lib_models.Job	"job"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/modules-change-request [patch]
func ExecModulesChangeRequest(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, lib_constants.HttpPathModulesChangeRequestResource, func(gc *gin.Context) {
		res, err := srv.ExecModulesChangeRequest(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Cancel modules change request
// @Description	Discard the currently pending modules change request.
// @Tags		modules
// @Produce		plain
// @Success		200	"modules change request canceled"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/modules-change-request [delete]
func CancelModulesChangeRequest(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, lib_constants.HttpPathModulesChangeRequestResource, func(gc *gin.Context) {
		err := srv.CancelModulesChangeRequest(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

// @Summary		Get modules available updates count
// @Description	Get the number of installed modules for which an update is available.
// @Tags		modules
// @Produce		json
// @Success		200	{integer}	int	"number of available module updates"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/modules-available-updates [get]
func GetModulesAvailableUpdatesCount(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathModulesAvailableUpdatesCountResource, func(gc *gin.Context) {
		res, err := srv.GetModulesAvailableUpdatesCount(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}
