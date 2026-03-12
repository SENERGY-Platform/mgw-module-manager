/*
 * Copyright 2025 InfAI (CC SES)
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

package shared

import (
	_ "github.com/SENERGY-Platform/mgw-go-service-base/srv-info-hdl/lib"
	"github.com/SENERGY-Platform/mgw-module-manager/lib"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/gin-gonic/gin"
	"net/http"
)

// getSrvInfoH godoc
// @Summary Get service info
// @Description	Get basic service and runtime information.
// @Tags Info
// @Produce	json
// @Success	200 {object} lib.SrvInfo "info"
// @Failure	500 {string} string "error message"
// @Router /info [get]
func getSrvInfoH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_model.SrvInfoPath, func(gc *gin.Context) {
		gc.JSON(http.StatusOK, a.GetSrvInfo(gc.Request.Context()))
	}
}
