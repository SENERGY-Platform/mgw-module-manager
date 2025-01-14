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

package standard

import (
	job_hdl_lib "github.com/SENERGY-Platform/go-service-base/job-hdl/lib"
	"github.com/SENERGY-Platform/mgw-module-manager/lib"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/gin-gonic/gin"
	"net/http"
	"path"
	"time"
)

type jobsQuery struct {
	Status   string `form:"status"`
	SortDesc bool   `form:"sort_desc"`
	Since    string `form:"since"`
	Until    string `form:"until"`
}

func getJobsH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_model.JobsPath, func(gc *gin.Context) {
		query := jobsQuery{}
		if err := gc.ShouldBindQuery(&query); err != nil {
			_ = gc.Error(lib_model.NewInvalidInputError(err))
			return
		}
		jobOptions := job_hdl_lib.JobFilter{
			Status:   query.Status,
			SortDesc: query.SortDesc,
		}
		if query.Since != "" {
			t, err := time.Parse(time.RFC3339Nano, query.Since)
			if err != nil {
				_ = gc.Error(lib_model.NewInvalidInputError(err))
				return
			}
			jobOptions.Since = t
		}
		if query.Until != "" {
			t, err := time.Parse(time.RFC3339Nano, query.Until)
			if err != nil {
				_ = gc.Error(lib_model.NewInvalidInputError(err))
				return
			}
			jobOptions.Until = t
		}
		jobs, _ := a.GetJobs(gc.Request.Context(), jobOptions)
		gc.JSON(http.StatusOK, jobs)
	}
}

func getJobH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodGet, path.Join(lib_model.JobsPath, ":id"), func(gc *gin.Context) {
		job, err := a.GetJob(gc.Request.Context(), gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, job)
	}
}

func patchJobCancelH(a lib.Api) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, path.Join(lib_model.JobsPath, ":id", lib_model.JobsCancelPath), func(gc *gin.Context) {
		err := a.CancelJob(gc.Request.Context(), gc.Param("id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}
