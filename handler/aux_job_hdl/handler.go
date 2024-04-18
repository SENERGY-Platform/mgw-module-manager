/*
 * Copyright 2024 InfAI (CC SES)
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

package aux_job_hdl

import (
	"sync"
	"time"
)

type Handler struct {
	jobs map[string]map[string]time.Time // {dID:{jID:timestamp}}
	mu   sync.RWMutex
}

func New() *Handler {
	return &Handler{
		jobs: make(map[string]map[string]time.Time),
	}
}

func (h *Handler) Add(dID, jID string) {
	h.mu.Lock()
	if _, ok := h.jobs[dID]; !ok {
		h.jobs[dID] = make(map[string]time.Time)
	}
	h.jobs[dID][jID] = time.Now()
	h.mu.Unlock()
}

func (h *Handler) Check(dID, jID string) (ok bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if _, ok = h.jobs[dID]; !ok {
		return
	}
	_, ok = h.jobs[dID][jID]
	return
}

func (h *Handler) Purge(maxAge time.Duration) {
	h.mu.Lock()
	var old [][2]string
	for dID, jobs := range h.jobs {
		for jID, timestamp := range jobs {
			if time.Since(timestamp) >= maxAge {
				old = append(old, [2]string{dID, jID})
			}
		}
	}
	for _, item := range old {
		delete(h.jobs[item[0]], item[1])
	}
	var empty []string
	for dID, jobs := range h.jobs {
		if len(jobs) == 0 {
			empty = append(empty, dID)
		}
	}
	for _, item := range empty {
		delete(h.jobs, item)
	}
	h.mu.Unlock()
}
