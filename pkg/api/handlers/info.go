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

// @Summary		Get service info
// @Description	Get information about the service such as name, version and uptime.
// @Tags		info
// @Produce		json
// @Success		200	{object}	models.ServiceInfo	"service info"
// @Failure		500	{string}	string	"error message"
// @Router		/info [get]
// @Router		/restricted/info [get]
func ServiceInfo(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathServiceInfoResource, func(gc *gin.Context) {
		gc.JSON(http.StatusOK, srv.ServiceInfo())
	}
}
