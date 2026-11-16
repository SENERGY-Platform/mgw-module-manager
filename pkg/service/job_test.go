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

package service

import (
	"testing"

	handler_jobs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
)

func TestActiveJobsErrMsg(t *testing.T) {
	deploymentJob := &handler_jobs.Job{Id: "2", Description: "update deployments"}
	moduleJob := &handler_jobs.Job{Id: "1", Description: "execute modules change request"}
	tests := []struct {
		name string
		jobs map[int]*handler_jobs.Job
		want string
	}{
		{
			name: "single job in a slot other than the first",
			jobs: map[int]*handler_jobs.Job{moduleJobSlotNum: moduleJob},
			want: "active job: execute modules change request (1)",
		},
		{
			name: "two jobs in slot order",
			jobs: map[int]*handler_jobs.Job{moduleJobSlotNum: moduleJob, deploymentJobSlotNum: deploymentJob},
			want: "active jobs: update deployments (2), execute modules change request (1)",
		},
		{
			name: "slot numbers with a gap",
			jobs: map[int]*handler_jobs.Job{7: moduleJob, 3: deploymentJob},
			want: "active jobs: update deployments (2), execute modules change request (1)",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Map iteration order varies between runs, repeating catches an order that is only right by chance.
			for range 20 {
				if got := activeJobsErrMsg(tc.jobs); got != tc.want {
					t.Fatalf("got %q, want %q", got, tc.want)
				}
			}
		})
	}
}
