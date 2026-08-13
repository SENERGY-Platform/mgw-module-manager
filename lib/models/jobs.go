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

package models

import (
	"time"
)

// Job represents an operation that is executed asynchronously.
// Operations returning a Job are completed once End is set, the outcome must then be retrieved from the result endpoint of the respective operation.
type Job struct {
	Id          string    `json:"id"`          // unique ID, used to query the job, cancel it and retrieve its result
	Description string    `json:"description"` // human readable description of the operation the job executes
	Start       time.Time `json:"start"`       // point in time at which the job was created
	End         time.Time `json:"end"`         // point in time at which the job ended, zero value as long as the job is running or was canceled before completion
}

// JobResult is embedded in all job results and identifies the job as well as errors that aborted it.
type JobResult struct {
	JobId       string `json:"job_id"` // ID of the job that produced the result
	ErrorResult        // set if the job was aborted, individual item errors are reported by the results of the embedding model
}
