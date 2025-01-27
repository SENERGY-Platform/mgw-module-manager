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

// getJobsH godoc
// @Summary List jobs
// @Description	List all jobs.
// @Tags Jobs
// @Produce	json
// @Param status query string false "status to filter by" Enums(pending, running, canceled, completed, error, ok)
// @Param sort_desc query bool false "sort in descending order"
// @Param since query string false "list jobs since timestamp"
// @Param until query string false "list jobs until timestamp"
// @Success	200 {array} job_hdl_lib.Job "jobs"
// @Failure	400 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /jobs [get]
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

// getJobH godoc
// @Summary Get job
// @Description	Get a job.
// @Tags Jobs
// @Produce	json
// @Param id path string true "job id"
// @Success	200 {object} job_hdl_lib.Job "job"
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /jobs/{id} [get]
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

// patchJobCancelH godoc
// @Summary Cancel job
// @Description	Cancels a job.
// @Tags Jobs
// @Param id path string true "job id"
// @Success	200
// @Failure	404 {string} string "error message"
// @Failure	500 {string} string "error message"
// @Router /jobs/{id}/cancel [patch]
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
