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

package test_client

import "testing"

func TestGetJobs(t *testing.T) {
	jobs, err := client.GetJobs(
		t.Context(),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), jobs)
}

func TestGetJob(t *testing.T) {
	job, err := client.GetJob(
		t.Context(),
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), job)
}

func TestCancelJobs(t *testing.T) {
	err := client.CancelJobs(
		t.Context(),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancelJob(t *testing.T) {
	err := client.CancelJob(
		t.Context(),
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
}
