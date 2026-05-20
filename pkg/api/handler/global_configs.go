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

package handler

import (
	"net/http"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func getGlobalConfigsFilter(gc *gin.Context) ([]string, error) {
	var query struct {
		Ids []string `form:"ids" collection_format:"csv"`
	}
	err := gc.MustBindWith(&query, binding.Query)
	if err != nil {
		return nil, err
	}
	return query.Ids, nil
}

func GetGlobalConfigs(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "global-configs", func(gc *gin.Context) {
		filter, err := getGlobalConfigsFilter(gc)
		if err != nil {
			return
		}
		res, err := srv.GetGlobalConfigs(gc, filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetGlobalConfig(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "global-configs/:id", func(gc *gin.Context) {
		res, err := srv.GetGlobalConfig(gc, gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func CreateGlobalConfig(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, "global-configs", func(gc *gin.Context) {
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

func UpdateGlobalConfig(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPut, "global-configs/:id", func(gc *gin.Context) {
		var body lib_models.GlobalConfigInput
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		err = srv.UpdateGlobalConfig(gc, lib_models.GlobalConfig{
			Id:             gc.Param("id"),
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

func getDeleteGlobalConfigsFilter(gc *gin.Context) ([]string, bool, error) {
	var query struct {
		Ids      []string `form:"ids" collection_format:"csv"`
		AllowAll bool     `form:"allow_all"`
	}
	err := gc.MustBindWith(&query, binding.Query)
	if err != nil {
		return nil, false, err
	}
	return query.Ids, query.AllowAll, nil
}

func DeleteGlobalConfigs(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, "global-configs", func(gc *gin.Context) {
		ids, allowAll, err := getDeleteGlobalConfigsFilter(gc)
		if err != nil {
			return
		}
		err = srv.DeleteGlobalConfigs(gc, ids, allowAll)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func DeleteGlobalConfig(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, "global-configs/:id", func(gc *gin.Context) {
		err := srv.DeleteGlobalConfig(gc, gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}
