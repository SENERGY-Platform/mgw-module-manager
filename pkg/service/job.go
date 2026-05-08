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

package service

import (
	"context"
	"fmt"
	"slices"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	handler_jobs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
)

func (s *Service) Jobs(_ context.Context, filterIds []string) ([]lib_models.Job, error) {
	handlerJobs := s.jobsHandler.Jobs(filterIds)
	var jobs []lib_models.Job
	for _, handlerJob := range handlerJobs {
		jobs = append(jobs, getJob(handlerJob))
	}
	slices.SortStableFunc(jobs, func(a, b lib_models.Job) int {
		return a.Start.Compare(b.Start)
	})
	return jobs, nil
}

func (s *Service) Job(_ context.Context, id string) (lib_models.Job, error) {
	handlerJob, ok := s.jobsHandler.Job(id)
	if !ok {
		return lib_models.Job{}, lib_errors.New[lib_errors.ErrNotFound]("")
	}
	return getJob(handlerJob), nil
}

func (s *Service) CancelJobs(_ context.Context, ids []string) error {
	handlerJobs := s.jobsHandler.Jobs(ids)
	for _, handlerJob := range handlerJobs {
		handlerJob.Cancel()
	}
	return nil
}

func (s *Service) CancelJob(_ context.Context, id string) error {
	handlerJob, ok := s.jobsHandler.Job(id)
	if !ok {
		return lib_errors.New[lib_errors.ErrNotFound]("")
	}
	handlerJob.Cancel()
	return nil
}

func getJob(handlerJob *handler_jobs.Job) lib_models.Job {
	job := lib_models.Job{
		Id:          handlerJob.Id,
		Description: handlerJob.Description,
		Start:       handlerJob.Start,
		End:         handlerJob.End(),
	}
	return job
}

func activeJobErrMsg(j *handler_jobs.Job) string {
	return fmt.Sprintf("active job: %s (%s)", j.Description, j.Id)
}

func activeJobsErrMsg(jobs map[int]*handler_jobs.Job) string {
	msg := "active job"
	lenJobs := len(jobs)
	if lenJobs > 1 {
		msg += "s"
	}
	msg += ": "
	for i, j := range jobs {
		msg += fmt.Sprintf("%s (%s)", j.Description, j.Id)
		if i < lenJobs-1 {
			msg += ", "
		}
	}
	return msg
}
