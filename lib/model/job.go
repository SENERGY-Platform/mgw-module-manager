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

package model

import "time"

type Job struct {
	ID          string     `json:"id"`
	Error       any        `json:"error"`
	Created     time.Time  `json:"created"`
	Started     *time.Time `json:"started"`
	Completed   *time.Time `json:"completed"`
	Canceled    *time.Time `json:"canceled"`
	Description string     `json:"description"`
}

type JobStatus = string

type JobFilter struct {
	Status   JobStatus
	SortDesc bool
	Since    time.Time
	Until    time.Time
}
