/*
 * Copyright 2023 InfAI (CC SES)
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

package http_engine

import (
	"github.com/SENERGY-Platform/mgw-module-manager/itf"
	"github.com/gin-gonic/gin"
	"net/http"
)

const modIdParam = "m"

func getModulesH(a itf.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		modules, err := a.GetModules(gc.Request.Context())
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, modules)
	}
}

func getModuleH(a itf.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		module, err := a.GetModule(gc.Request.Context(), gc.Param(modIdParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, module)
	}
}

func getInputTemplateH(a itf.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		inputTemplate, err := a.GetInputTemplate(gc.Request.Context(), gc.Param(modIdParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, inputTemplate)
	}
}