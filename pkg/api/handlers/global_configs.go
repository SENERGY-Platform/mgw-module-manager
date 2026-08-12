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

// @Summary		Get global configs
// @Description	Get all global configs or only the global configs matching the given IDs.
// @Tags		global-configs
// @Produce		json
// @Param		ids	query	[]string	false	"filter by global config IDs"	collectionFormat(csv)
// @Success		200	{object}	map[string]lib_models.GlobalConfig	"global configs by ID"
// @Failure		400	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/global-configs [get]
func GetGlobalConfigs(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathGlobalConfigsCollection, func(gc *gin.Context) {
		var query struct {
			Ids []string `form:"ids" collection_format:"csv"`
		}
		err := gc.MustBindWith(&query, binding.Query)
		if err != nil {
			return
		}
		res, err := srv.GetGlobalConfigs(gc, query.Ids)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Get global config
// @Description	Get a single global config by its ID.
// @Tags		global-configs
// @Produce		json
// @Param		CFG_ID	path	string	true	"global config ID"
// @Success		200	{object}	lib_models.GlobalConfig	"global config"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/global-configs/{CFG_ID} [get]
func GetGlobalConfig(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathGlobalConfigResource, func(gc *gin.Context) {
		res, err := srv.GetGlobalConfig(gc, gc.Param("CFG_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Create global config
// @Description	Create a global config.
// @Tags		global-configs
// @Accept		json
// @Produce		json
// @Param		request	body	lib_models.GlobalConfigInput	true	"global config"
// @Success		200	{string}	string	"global config ID"
// @Failure		400	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/global-configs [post]
func CreateGlobalConfig(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_constants.HttpPathGlobalConfigsCollection, func(gc *gin.Context) {
		var body lib_models.GlobalConfigInput
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.CreateGlobalConfig(gc, body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Update global config
// @Description	Update a global config identified by its ID.
// @Tags		global-configs
// @Accept		json
// @Produce		plain
// @Param		CFG_ID	path	string	true	"global config ID"
// @Param		request	body	lib_models.GlobalConfigInput	true	"global config"
// @Success		200	"global config updated"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/global-configs/{CFG_ID} [put]
func UpdateGlobalConfig(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPut, lib_constants.HttpPathGlobalConfigResource, func(gc *gin.Context) {
		var body lib_models.GlobalConfigInput
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		err = srv.UpdateGlobalConfig(gc, lib_models.GlobalConfig{
			Id:             gc.Param("CFG_ID"),
			Name:           body.Name,
			InterfaceValue: body.InterfaceValue,
		})
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

// @Summary		Delete global configs
// @Description	Delete the global configs matching the given IDs. All global configs are deleted if allow_all is set and no IDs are given.
// @Tags		global-configs
// @Produce		plain
// @Param		ids	query	[]string	false	"filter by global config IDs"	collectionFormat(csv)
// @Param		allow_all	query	bool	false	"allow deletion of all global configs"
// @Success		200	"global configs deleted"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/global-configs [delete]
func DeleteGlobalConfigs(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, lib_constants.HttpPathGlobalConfigsCollection, func(gc *gin.Context) {
		var query struct {
			Ids      []string `form:"ids" collection_format:"csv"`
			AllowAll bool     `form:"allow_all"`
		}
		err := gc.MustBindWith(&query, binding.Query)
		if err != nil {
			return
		}
		err = srv.DeleteGlobalConfigs(gc, query.Ids, query.AllowAll)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

// @Summary		Delete global config
// @Description	Delete a single global config by its ID.
// @Tags		global-configs
// @Produce		plain
// @Param		CFG_ID	path	string	true	"global config ID"
// @Success		200	"global config deleted"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/global-configs/{CFG_ID} [delete]
func DeleteGlobalConfig(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, lib_constants.HttpPathGlobalConfigResource, func(gc *gin.Context) {
		err := srv.DeleteGlobalConfig(gc, gc.Param("CFG_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}
