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

package job_hdl

import (
	"context"
	"fmt"
	"github.com/SENERGY-Platform/go-cc-job-handler/ccjh"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/google/uuid"
	"sort"
	"sync"
	"time"
)

type Handler struct {
	mu        sync.RWMutex
	ctx       context.Context
	ccHandler *ccjh.Handler
	jobs      map[string]*job
}

func New(ctx context.Context, ccHandler *ccjh.Handler) *Handler {
	return &Handler{
		ctx:       ctx,
		ccHandler: ccHandler,
		jobs:      make(map[string]*job),
	}
}

func (h *Handler) Create(desc string, tFunc func(context.Context, context.CancelFunc) error) (string, error) {
	uid, err := uuid.NewRandom()
	if err != nil {
		return "", model.NewInternalError(err)
	}
	id := uid.String()
	ctx, cf := context.WithCancel(h.ctx)
	j := job{
		meta: model.Job{
			ID:          id,
			Created:     time.Now().UTC(),
			Description: desc,
		},
		tFunc: tFunc,
		ctx:   ctx,
		cFunc: cf,
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	err = h.ccHandler.Add(&j)
	if err != nil {
		return "", model.NewInternalError(err)
	}
	h.jobs[id] = &j
	return id, nil
}

func (h *Handler) Get(id string) (model.Job, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	j, ok := h.jobs[id]
	if !ok {
		return model.Job{}, model.NewNotFoundError(fmt.Errorf("%s not found", id))
	}
	return j.Meta(), nil
}

func (h *Handler) Cancel(id string) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	j, ok := h.jobs[id]
	if !ok {
		return model.NewNotFoundError(fmt.Errorf("%s not found", id))
	}
	j.Cancel()
	return nil
}

func (h *Handler) List(filter model.JobFilter) []model.Job {
	var jobs []model.Job
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, v := range h.jobs {
		if check(filter, v.Meta()) {
			jobs = append(jobs, v.Meta())
		}
	}
	if filter.SortDesc {
		sort.Slice(jobs, func(i, j int) bool {
			return jobs[i].Created.After(jobs[j].Created)
		})
	} else {
		sort.Slice(jobs, func(i, j int) bool {
			return jobs[i].Created.Before(jobs[j].Created)
		})
	}
	return jobs
}

func (h *Handler) PurgeJobs(maxAge int64) int {
	var l []string
	tNow := time.Now().UTC()
	h.mu.RLock()
	for k, v := range h.jobs {
		m := v.Meta()
		if v.IsCanceled() || m.Completed != nil || m.Canceled != nil {
			if tNow.Sub(m.Created).Microseconds() >= maxAge {
				l = append(l, k)
			}
		}
	}
	h.mu.RUnlock()
	h.mu.Lock()
	for _, id := range l {
		delete(h.jobs, id)
	}
	h.mu.Unlock()
	return len(l)
}

func check(filter model.JobFilter, job model.Job) bool {
	if !filter.Since.IsZero() && !job.Created.After(filter.Since) {
		return false
	}
	if !filter.Until.IsZero() && !job.Created.Before(filter.Until) {
		return false
	}
	switch filter.Status {
	case model.JobPending:
		if job.Started != nil || job.Canceled != nil || job.Completed != nil {
			return false
		}
	case model.JobRunning:
		if job.Started == nil || job.Canceled != nil || job.Completed != nil {
			return false
		}
	case model.JobCanceled:
		if job.Canceled == nil {
			return false
		}
	case model.JobCompleted:
		if job.Completed == nil {
			return false
		}
	case model.JobError:
		if job.Completed != nil && job.Error == nil {
			return false
		}
	case model.JobOK:
		if job.Completed != nil && job.Error != nil {
			return false
		}
	}
	return true
}
