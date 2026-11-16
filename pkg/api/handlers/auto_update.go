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

// @Summary		Get auto-update settings
// @Description	Get the settings for updating installed modules and their deployments automatically. Defaults are returned if the settings were never changed.
// @Tags		auto-update
// @Produce		json
// @Success		200	{object}	lib_models.AutoUpdateSettings	"auto-update settings"
// @Failure		500	{string}	string	"error message"
// @Router		/auto-update [get]
func GetAutoUpdateSettings(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathAutoUpdateResource, func(gc *gin.Context) {
		res, err := srv.GetAutoUpdateSettings(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, guardNil(res))
	}
}

// @Summary		Update auto-update settings
// @Description	Replace the settings for updating installed modules and their deployments automatically.
// @Tags		auto-update
// @Accept		json
// @Produce		plain
// @Param		request	body	lib_models.AutoUpdateSettings	true	"auto-update settings"
// @Success		200	"auto-update settings updated"
// @Failure		400	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/auto-update [put]
func UpdateAutoUpdateSettings(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPut, lib_constants.HttpPathAutoUpdateResource, func(gc *gin.Context) {
		var body lib_models.AutoUpdateSettings
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		err = srv.UpdateAutoUpdateSettings(gc, body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

// @Summary		Get auto-update status
// @Description	Get the auto-update settings together with the outcome of the last update run.
// @Tags		auto-update
// @Produce		json
// @Success		200	{object}	lib_models.AutoUpdateStatus	"auto-update settings and last run"
// @Failure		500	{string}	string	"error message"
// @Router		/auto-update/status [get]
func GetAutoUpdateStatus(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathAutoUpdateStatusResource, func(gc *gin.Context) {
		res, err := srv.GetAutoUpdateStatus(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, guardNil(res))
	}
}
