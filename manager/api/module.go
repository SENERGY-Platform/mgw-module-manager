/*
 * Copyright 2022 InfAI (CC SES)
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

package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (a *Api) GetModules(gc *gin.Context) {
	modules, err := a.moduleHandler.List()
	if err != nil {
		_ = gc.Error(err)
		return
	}
	gc.JSON(http.StatusOK, &modules)
}

func (a *Api) GetModule(gc *gin.Context) {
	module, err := a.moduleHandler.Read(gc.Param("module"))
	if err != nil {
		_ = gc.Error(err)
		return
	}
	gc.JSON(http.StatusOK, &module)
}

func (a *Api) GetModuleInputTemplate(gc *gin.Context) {
	module, err := a.moduleHandler.Read(gc.Param("module"))
	if err != nil {
		_ = gc.Error(err)
		return
	}
	template := a.deploymentHandler.InputTemplate(module)
	gc.JSON(http.StatusOK, &template)
}
