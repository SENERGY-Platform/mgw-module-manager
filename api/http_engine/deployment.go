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
	"github.com/SENERGY-Platform/mgw-module-manager/lib"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/gin-gonic/gin"
	"net/http"
)

const depIdParam = "d"

func getDeploymentsH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		deployments, err := a.GetDeployments(gc.Request.Context())
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, deployments)
	}
}

func getDeploymentH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		deployment, err := a.GetDeployment(gc.Request.Context(), gc.Param(depIdParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, deployment)
	}
}

func postDeploymentH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		var depReq model.DepRequest
		err := gc.ShouldBindJSON(&depReq)
		if err != nil {
			_ = gc.Error(model.NewInvalidInputError(err))
			return
		}
		id, err := a.CreateDeployment(gc.Request.Context(), depReq)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, id)
	}
}

func putDeploymentH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		var depReq model.DepRequest
		err := gc.ShouldBindJSON(&depReq)
		if err != nil {
			_ = gc.Error(model.NewInvalidInputError(err))
			return
		}
		err = a.UpdateDeployment(gc.Request.Context(), gc.Param(depIdParam), depReq)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func deleteDeploymentH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		err := a.DeleteDeployment(gc.Request.Context(), gc.Param(depIdParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}
