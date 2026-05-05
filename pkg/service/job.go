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
	"slices"

	lib_models_service "github.com/SENERGY-Platform/mgw-module-manager/lib/models/service"
	handler_jobs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
)

func (s *Service) Jobs(_ context.Context, filterIds []string) ([]lib_models_service.Job, error) {
	handlerJobs := s.jobsHandler.Jobs(filterIds)
	var jobs []lib_models_service.Job
	for _, handlerJob := range handlerJobs {
		jobs = append(jobs, getJob(handlerJob))
	}
	slices.SortStableFunc(jobs, func(a, b lib_models_service.Job) int {
		return a.Start.Compare(b.Start)
	})
	return jobs, nil
}

func (s *Service) Job(_ context.Context, jobId string) (lib_models_service.Job, error) {
	handlerJob, ok := s.jobsHandler.Job(jobId)
	if !ok {
		return lib_models_service.Job{}, models_error.NotFoundErr
	}
	return getJob(handlerJob), nil
}

func (s *Service) CancelJobs(_ context.Context, jobIds []string) error {
	handlerJobs := s.jobsHandler.Jobs(jobIds)
	for _, handlerJob := range handlerJobs {
		handlerJob.Cancel()
	}
	return nil
}

func (s *Service) CancelJob(_ context.Context, jobId string) error {
	handlerJob, ok := s.jobsHandler.Job(jobId)
	if !ok {
		return models_error.NotFoundErr
	}
	handlerJob.Cancel()
	return nil
}

func getJob(handlerJob *handler_jobs.Job) lib_models_service.Job {
	job := lib_models_service.Job{
		Id:          handlerJob.Id,
		Description: handlerJob.Description,
		Start:       handlerJob.Start,
		End:         handlerJob.End(),
	}
	return job
}
