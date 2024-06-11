/*
 * Copyright 2024 InfAI (CC SES)
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

package http_hdl

import (
	"errors"
	"github.com/SENERGY-Platform/mgw-module-manager/lib"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/gin-gonic/gin"
	"net/http"
)

const advRefParam = "r"

type advertisementFilterQuery struct {
	ModuleID string `form:"module_id"`
	Origin   string `form:"origin"`
	Ref      string `form:"ref"`
}

func getAdvertisementQueryH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		var query advertisementFilterQuery
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		ads, err := a.QueryAdvertisements(gc.Request.Context(), lib_model.DepAdvFilter{
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

func getAdvertisementH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		adv, err := a.GetAdvertisement(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), gc.Param(advRefParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, adv)
	}
}

func getAdvertisementsH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		ads, err := a.GetAdvertisements(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, ads)
	}
}

func putAdvertisementH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		var advBase lib_model.DepAdvertisementBase
		err := gc.ShouldBindJSON(&advBase)
		if err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		if gc.Param(advRefParam) != advBase.Ref {
			_ = gc.Error(lib_model.NewInvalidInputError(errors.New("reference mismatch")))
			return
		}
		err = a.PutAdvertisement(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), advBase)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func putAdvertisementsH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		var ads map[string]lib_model.DepAdvertisementBase
		err := gc.ShouldBindJSON(&ads)
		if err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		err = a.PutAdvertisements(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), ads)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func deleteAdvertisementH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		err := a.DeleteAdvertisement(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey), gc.Param(advRefParam))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func deleteAdvertisementsH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		err := a.DeleteAdvertisements(gc.Request.Context(), gc.GetHeader(lib_model.DepIdHeaderKey))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}
