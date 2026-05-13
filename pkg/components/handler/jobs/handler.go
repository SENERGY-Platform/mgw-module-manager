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

package jobs

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	helper_context "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/context"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	helper_uuid "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/uuid"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	"github.com/bytedance/gopkg/util/logger"
)

const ContextKeyJobId = "job_id"

type Config struct {
	MaxJobAge        time.Duration `json:"max_job_age" env_var:"JOBS_HANDLER_MAX_JOB_AGE"`
	CleanupLoopDelay time.Duration `json:"cleanup_loop_delay" env_var:"JOBS_HANDLER_CLEANUP_LOOP_DELAY"`
}

type Handler struct {
	jobSlots       map[int]*Job
	jobMap         map[string]*Job
	config         Config
	cleanupHandler func([]string)
	ctx            context.Context
	mu             sync.RWMutex
}

func New(ctx context.Context, config Config) *Handler {
	return &Handler{
		jobSlots: make(map[int]*Job),
		jobMap:   make(map[string]*Job),
		config:   config,
		ctx:      ctx,
	}
}

func (h *Handler) CreateJob(description string) (*Job, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	id, err := helper_uuid.New()
	if err != nil {
		return nil, err
	}
	ctx, cf := context.WithCancel(h.ctx)
	ctx = helper_context.WithValues(ctx, ContextKeyJobId, id)
	job := &Job{
		Id:          id,
		Description: description,
		Start:       helper_time.Now(),
		context:     ctx,
		cancelFunc:  cf,
	}
	h.jobMap[id] = job
	return job, nil
}

func (h *Handler) CreateSlotJob(slotNum int, description string) (*Job, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	j, ok := h.jobSlots[slotNum]
	if ok {
		return nil, lib_errors.New[lib_errors.ErrActiveJob](fmt.Sprintf("active job: %s (%s)", j.Description, j.Id))
	}
	id, err := helper_uuid.New()
	if err != nil {
		return nil, err
	}
	ctx, cf := context.WithCancel(h.ctx)
	ctx = helper_context.WithValues(ctx, ContextKeyJobId, id)
	job := &Job{
		Id:          id,
		Description: description,
		Start:       helper_time.Now(),
		doneHandler: slotJobDoneHandler{
			slotNum:  slotNum,
			doneFunc: h.slotJobDone,
		},
		context:    ctx,
		cancelFunc: cf,
	}
	h.jobSlots[slotNum] = job
	h.jobMap[id] = job
	return job, nil
}

func (h *Handler) CurrentSlotJob(slotNum int) (*Job, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	job, ok := h.jobSlots[slotNum]
	return job, ok
}

func (h *Handler) CurrentSlotJobs(slotNumFilter []int) map[int]*Job {
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

func (h *Handler) SetCleanupHandler(f func([]string)) {
	h.cleanupHandler = f
}

func (h *Handler) Cleanup(ctx context.Context) {
	timer := time.NewTimer(h.config.CleanupLoopDelay)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			oldJobs := h.cleanup()
			if h.cleanupHandler != nil && len(oldJobs) > 0 {
				h.cleanupHandler(oldJobs)
			}
			timer.Reset(h.config.CleanupLoopDelay)
		case <-ctx.Done():
			return
		}
	}
}

func (h *Handler) cleanup() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var oldJobs []string
	tmp := make(map[string]*Job)
	now := helper_time.Now()
	for id, job := range h.jobMap {
		end := job.End()
		if now.Sub(end) < h.config.MaxJobAge {
			tmp[id] = job
		} else {
			oldJobs = append(oldJobs, id)
		}
	}
	h.jobMap = tmp
	logger.Debug("job cleanup", slog_keys.JobIds, oldJobs)
	return oldJobs
}

func (h *Handler) slotJobDone(slotNum int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.jobSlots, slotNum)
}

type slotJobDoneHandler struct {
	slotNum  int
	doneFunc func(int)
}

func (h slotJobDoneHandler) JobDone() {
	h.doneFunc(h.slotNum)
}
