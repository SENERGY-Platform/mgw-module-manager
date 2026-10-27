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

package deployments

import (
	"time"

	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

// restartRecord holds the runtime monitor starts of a deployment, oldest first.
type restartRecord struct {
	starts      []time.Time
	lastError   string
	lastErrorAt time.Time
}

// GetRestartInfo returns the restarts within the restart window for those of the given deployments that have any.
func (h *Handler) GetRestartInfo(ids []string) map[string]pkg_models.DeploymentRestartInfo {
	h.restartsMu.Lock()
	defer h.restartsMu.Unlock()
	now := h.now()
	infos := make(map[string]pkg_models.DeploymentRestartInfo)
	for _, id := range ids {
		record, ok := h.restarts[id]
		if !ok {
			continue
		}
		if h.pruneRestartRecord(record, now) {
			delete(h.restarts, id)
			continue
		}
		info := pkg_models.DeploymentRestartInfo{
			Restarts:  len(record.starts),
			LastError: record.lastError,
		}
		if len(record.starts) > 0 {
			last := record.starts[len(record.starts)-1]
			info.LastRestart = &last
		}
		infos[id] = info
	}
	return infos
}

// RestartThreshold returns the number of restarts within the window from which a deployment is reported as unhealthy, 0 if disabled.
func (h *Handler) RestartThreshold() int {
	return h.config.RestartThreshold
}

func (h *Handler) recordRestart(id string) {
	h.restartsMu.Lock()
	defer h.restartsMu.Unlock()
	now := h.now()
	record := h.getRestartRecord(id)
	h.pruneRestartRecord(record, now)
	record.starts = append(record.starts, now)
}

func (h *Handler) recordStartError(id string, err error) {
	h.restartsMu.Lock()
	defer h.restartsMu.Unlock()
	now := h.now()
	record := h.getRestartRecord(id)
	h.pruneRestartRecord(record, now)
	record.lastError = err.Error()
	record.lastErrorAt = now
}

func (h *Handler) forgetRestarts(id string) {
	h.restartsMu.Lock()
	defer h.restartsMu.Unlock()
	delete(h.restarts, id)
}

func (h *Handler) getRestartRecord(id string) *restartRecord {
	record, ok := h.restarts[id]
	if !ok {
		record = &restartRecord{}
		h.restarts[id] = record
	}
	return record
}

// pruneRestartRecord drops everything older than the restart window and reports whether nothing is left.
func (h *Handler) pruneRestartRecord(record *restartRecord, now time.Time) bool {
	cutoff := now.Add(-h.config.RestartWindow)
	n := 0
	for n < len(record.starts) && record.starts[n].Before(cutoff) {
		n++
	}
	record.starts = record.starts[n:]
	if record.lastError != "" && record.lastErrorAt.Before(cutoff) {
		record.lastError = ""
		record.lastErrorAt = time.Time{}
	}
	return len(record.starts) == 0 && record.lastError == ""
}
