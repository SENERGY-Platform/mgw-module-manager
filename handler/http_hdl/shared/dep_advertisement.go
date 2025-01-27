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
	"github.com/SENERGY-Platform/mgw-module-manager/lib"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/gin-gonic/gin"
	"net/http"
)

type advertisementFilterQuery struct {
	ModuleID string `form:"module_id"`
	Origin   string `form:"origin"`
	Ref      string `form:"ref"`
}

// getDepAdvertisementQueryH godoc
// @Summary Query advertisements
// @Description Query deployment advertisements.
// @Tags Deployment Advertisements
// @Produce	json
// @Param module_id query string false "filter by module ID"
// @Param origin query string false "filter by origin"
// @Param ref query string false "filter by reference"
// @Success	200 {array} lib_model.DepAdvertisement "advertisements"
// @Failure	400 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /discovery [get]
func getDepAdvertisementQueryH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_model.DiscoveryPath, func(gc *gin.Context) {
		var query advertisementFilterQuery
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		ads, err := a.QueryDepAdvertisements(gc.Request.Context(), lib_model.DepAdvFilter{
			ModuleID: query.ModuleID,
			Origin:   query.Origin,
			Ref:      query.Ref,
		})
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, ads)
	}
}
