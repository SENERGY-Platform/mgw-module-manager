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
	"sync"
	"time"

	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
)

type Job struct {
	context    context.Context
	cancelFunc context.CancelFunc
	slotNum    int
	doneFunc   func(int)
	data       JobData
	mu         sync.RWMutex
}

type JobData struct {
	Id    string
	Start time.Time
	End   time.Time
	Error error
}

func (j *Job) Context() context.Context {
	return j.context
}

func (j *Job) Cancel() {
	j.cancelFunc()
}

func (j *Job) Done() {
	defer j.cancelFunc()
	j.setEnd()
	j.doneFunc(j.slotNum)
}

func (j *Job) SetError(err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.data.Error = err
}

func (j *Job) Data() JobData {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.data
}

func (j *Job) setEnd() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.data.End = helper_time.Now()
}
