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
	"reflect"
	"slices"
	"testing"
	"time"

	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
)

func newEndedJob(t *testing.T, h *Handler, endedAgo time.Duration) *Job {
	t.Helper()
	job, err := h.CreateJob("ended")
	if err != nil {
		t.Fatal(err)
	}
	job.Done()
	job.mu.Lock()
	job.end = helper_time.Now().Add(-endedAgo)
	job.mu.Unlock()
	return job
}

func TestHandler_Cleanup(t *testing.T) {
	const maxAge = time.Hour
	h := New(context.Background(), Config{MaxJobAge: maxAge})
	running, err := h.CreateJob("running")
	if err != nil {
		t.Fatal(err)
	}
	running.Start = helper_time.Now().Add(-10 * maxAge)
	runningSlot, err := h.CreateSlotJob(1, "running slot job")
	if err != nil {
		t.Fatal(err)
	}
	runningSlot.Start = helper_time.Now().Add(-10 * maxAge)
	recent := newEndedJob(t, h, maxAge/2)
	old := newEndedJob(t, h, maxAge+time.Minute)

	removed := h.cleanup()

	if !reflect.DeepEqual(removed, []string{old.Id}) {
		t.Errorf("expected only the job that ended more than the max age ago to be removed, got %v", removed)
	}
	kept := h.Jobs(nil)
	for _, job := range []*Job{running, runningSlot, recent} {
		if _, ok := kept[job.Id]; !ok {
			t.Errorf("expected job '%s' to be kept", job.Description)
		}
	}
	if _, ok := kept[old.Id]; ok {
		t.Error("expected the old job to be removed")
	}
	if job, ok := h.CurrentSlotJob(1); !ok || job != runningSlot {
		t.Error("expected the running slot job to keep its slot")
	}

	// once ended, a job is measured from its end and not from its start
	running.Done()
	if removed = h.cleanup(); slices.Contains(removed, running.Id) {
		t.Error("expected a job that just ended to be kept")
	}
}
