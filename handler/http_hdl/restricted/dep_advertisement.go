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

package restricted

import (
	"errors"
	"github.com/SENERGY-Platform/mgw-module-manager/lib"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/gin-gonic/gin"
	"net/http"
	"path"
)

func getDepAdvertisementH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, path.Join(lib_model.DepAdvertisementsPath, ":id"), func(gc *gin.Context) {
		adv, err := a.GetDepAdvertisement(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, adv)
	}
}

func getDepAdvertisementsH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_model.DepAdvertisementsPath, func(gc *gin.Context) {
		ads, err := a.GetDepAdvertisements(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, ads)
	}
}

func putDepAdvertisementH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPut, path.Join(lib_model.DepAdvertisementsPath, ":id"), func(gc *gin.Context) {
		var advBase lib_model.DepAdvertisementBase
		err := gc.ShouldBindJSON(&advBase)
		if err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		if gc.Param("id") != advBase.Ref {
			_ = gc.Error(lib_model.NewInvalidInputError(errors.New("reference mismatch")))
			return
		}
		err = a.PutDepAdvertisement(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), advBase)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func putDepAdvertisementsH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPut, lib_model.DepAdvertisementsBatchPath, func(gc *gin.Context) {
		var ads map[string]lib_model.DepAdvertisementBase
		err := gc.ShouldBindJSON(&ads)
		if err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		err = a.PutDepAdvertisements(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), ads)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func deleteDepAdvertisementH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, path.Join(lib_model.DepAdvertisementsPath, ":id"), func(gc *gin.Context) {
		err := a.DeleteDepAdvertisement(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func deleteDepAdvertisementsH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, lib_model.DepAdvertisementsBatchPath, func(gc *gin.Context) {
		err := a.DeleteDepAdvertisements(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}
