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
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func GetJobs(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathJobsCollection, func(gc *gin.Context) {
		var query struct {
			Ids []string `form:"ids" collection_format:"csv"`
		}
		err := gc.MustBindWith(&query, binding.Query)
		if err != nil {
			return
		}
		res, err := srv.GetJobs(gc, query.Ids)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetJob(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathJobResource, func(gc *gin.Context) {
		res, err := srv.GetJob(gc, gc.Param("JOB_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func CancelJobs(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, lib_constants.HttpPathJobsCollection, func(gc *gin.Context) {
		var query struct {
			Ids []string `form:"ids" collection_format:"csv"`
		}
		err := gc.MustBindWith(&query, binding.Query)
		if err != nil {
			return
		}
		err = srv.CancelJobs(gc, query.Ids)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func CancelJob(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, lib_constants.HttpPathJobResource, func(gc *gin.Context) {
		err := srv.CancelJob(gc, gc.Param("JOB_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}
