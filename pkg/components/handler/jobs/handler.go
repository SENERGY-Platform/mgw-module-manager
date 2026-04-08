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

package handler_jobs

import (
	"context"
	"errors"
	"maps"
	"sync"
	"time"

	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	helper_uuid "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/uuid"
)

type Config struct {
	MaxJobAge        time.Duration `json:"max_job_age" env_var:"JOBS_HANDLER_MAX_JOB_AGE"`
	CleanupLoopDelay time.Duration `json:"cleanup_loop_delay" env_var:"JOBS_HANDLER_CLEANUP_LOOP_DELAY"`
}

type Handler struct {
	jobSlots map[int]*Job
	jobMap   map[string]*Job
	config   Config
	ctx      context.Context
	mu       sync.RWMutex
}

func New(ctx context.Context, config Config) *Handler {
	return &Handler{
		jobSlots: make(map[int]*Job),
		jobMap:   make(map[string]*Job),
		config:   config,
		ctx:      ctx,
	}
}

func (h *Handler) Create(slotNum int) (*Job, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.jobSlots[slotNum]
	if ok {
		return nil, errors.New("job exists") // TODO
	}
	id, err := helper_uuid.New()
	if err != nil {
		return nil, err
	}
	ctx, cf := context.WithCancel(h.ctx)
	job := &Job{
		context:    ctx,
		cancelFunc: cf,
		slotNum:    slotNum,
		doneFunc:   h.done,
		data: JobData{
			Id:    id,
			Start: helper_time.Now(),
		},
	}
	h.jobSlots[slotNum] = job
	h.jobMap[id] = job
	return job, nil
}

func (h *Handler) CurrentJob(slotNum int) (*Job, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	job, ok := h.jobSlots[slotNum]
	return job, ok
}

func (h *Handler) CurrentJobs(slotNumFilter []int) map[int]*Job {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(slotNumFilter) == 0 {
		return maps.Clone(h.jobSlots)
	}
	tmp := make(map[int]*Job)
	for _, slotNum := range slotNumFilter {
		job, ok := h.jobSlots[slotNum]
		if ok {
			tmp[slotNum] = job
		}
	}
	return tmp
}

func (h *Handler) Job(id string) (*Job, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	job, ok := h.jobMap[id]
	return job, ok
}

func (h *Handler) Jobs(filterIds []string) map[string]*Job {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(filterIds) == 0 {
		return maps.Clone(h.jobMap)
	}
	tmp := make(map[string]*Job)
	for _, id := range filterIds {
		job, ok := h.jobMap[id]
		if ok {
			tmp[id] = job
		}
	}
	return tmp
}

func (h *Handler) Cleanup(ctx context.Context) {
	timer := time.NewTimer(h.config.CleanupLoopDelay)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			h.cleanup()
			timer.Reset(h.config.CleanupLoopDelay)
		case <-ctx.Done():
			return
		}
	}
}

func (h *Handler) cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()
	tmp := make(map[string]*Job)
	now := helper_time.Now()
	for id, job := range h.jobMap {
		data := job.Data()
		if now.Sub(data.End) < h.config.MaxJobAge {
			tmp[id] = job
		}
	}
	h.jobMap = tmp
}

func (h *Handler) done(slotNum int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.jobSlots, slotNum)
}
