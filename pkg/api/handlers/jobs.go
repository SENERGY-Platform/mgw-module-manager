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
	"github.com/gin-gonic/gin/binding"
)

// @Summary		Get jobs
// @Description	Get all jobs or only the jobs matching the given IDs.
// @Tags		jobs
// @Produce		json
// @Param		ids	query	[]string	false	"filter by job IDs"	collectionFormat(csv)
// @Success		200	{array}	models.Job	"jobs"
// @Failure		400	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/jobs [get]
// @Router		/restricted/jobs [get]
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

// @Summary		Get job
// @Description	Get a single job by its ID.
// @Tags		jobs
// @Produce		json
// @Param		JOB_ID	path	string	true	"job ID"
// @Success		200	{object}	models.Job	"job"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/jobs/{JOB_ID} [get]
// @Router		/restricted/jobs/{JOB_ID} [get]
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

// @Summary		Cancel jobs
// @Description	Cancel all jobs matching the given IDs.
// @Tags		jobs
// @Accept		json
// @Produce		plain
// @Param		request	body	[]string	true	"job IDs"
// @Success		200	"jobs canceled"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/jobs-cancel [post]
// @Router		/restricted/jobs-cancel [post]
func CancelJobs(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_constants.HttpPathCancelJobs, func(gc *gin.Context) {
		var body []string
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		err = srv.CancelJobs(gc, body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

// @Summary		Cancel job
// @Description	Cancel a single job by its ID.
// @Tags		jobs
// @Produce		plain
// @Param		JOB_ID	path	string	true	"job ID"
// @Success		200	"job canceled"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/jobs/{JOB_ID} [patch]
// @Router		/restricted/jobs/{JOB_ID} [patch]
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
