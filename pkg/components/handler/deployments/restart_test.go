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
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

// rstTestCew records the restarted containers and whether the deployment was excluded from the runtime monitor
// while it happened.
type rstTestCew struct {
	containerEngineWrapperClient
	h         *Handler
	deployId  map[string]string // container name -> deployment id
	failFor   map[string]error
	mu        sync.Mutex
	restarted []string
	excluded  map[string]bool
}

func (m *rstTestCew) RestartContainer(_ context.Context, id string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.failFor[id]; err != nil {
		return "", err
	}
	m.h.runtimeMonitorJobsMu.RLock()
	_, ok := m.h.runtimeMonitorJobs[m.deployId[id]]
	m.h.runtimeMonitorJobsMu.RUnlock()
	m.excluded[id] = ok
	m.restarted = append(m.restarted, id)
	return "job-" + id, nil
}

func (m *rstTestCew) GetJob(_ context.Context, id string) (external_models.JobLibJob, error) {
	now := time.Now()
	return external_models.JobLibJob{ID: id, Completed: &now}, nil
}

func newRstTestHandler(deployments map[string]pkg_models.DeploymentBase, containers map[string]map[string]pkg_models.DeploymentContainerBase) (*Handler, *rstTestCew) {
	db := &restartTestDbMock{deployments: deployments, containers: containers}
	cew := &rstTestCew{deployId: make(map[string]string), failFor: make(map[string]error), excluded: make(map[string]bool)}
	for depId, cons := range containers {
		for _, c := range cons {
			cew.deployId[c.Name] = depId
		}
	}
	h := New(db, cew, nil, nil, nil, Config{JobPollInterval: time.Millisecond, RestartWindow: time.Minute * 10, RestartThreshold: 3})
	cew.h = h
	return h, cew
}

func rstTestSelection(ids ...string) map[string]pkg_models.Module {
	mods := make(map[string]pkg_models.Module)
	for _, id := range ids {
		mods[id] = updTestModule(id)
	}
	return mods
}

func rstTestContainers(depId string, names ...string) map[string]pkg_models.DeploymentContainerBase {
	cons := make(map[string]pkg_models.DeploymentContainerBase)
	for _, name := range names {
		cons[name] = pkg_models.DeploymentContainerBase{Name: name, DeploymentId: depId, Reference: name}
	}
	return cons
}

func TestHandler_RestartDeployments(t *testing.T) {
	h, cew := newRstTestHandler(
		map[string]pkg_models.DeploymentBase{
			"dep-a": {Id: "dep-a", ModuleId: "a", Enabled: true},
			"dep-b": {Id: "dep-b", ModuleId: "b", Enabled: false},
		},
		map[string]map[string]pkg_models.DeploymentContainerBase{
			"dep-a": rstTestContainers("dep-a", "a1", "a2"),
			"dep-b": rstTestContainers("dep-b", "b1"),
		},
	)
	results, err := h.RestartDeployments(context.Background(), rstTestSelection("a", "b", "c"))
	if err != nil {
		t.Fatal(err)
	}
	want := []lib_models.DeploymentResult{
		{ModuleId: "a", Id: "dep-a"},
		{ModuleId: "b", Id: "dep-b", ErrorResult: lib_models.NewErrorResult("deployment is disabled")},
		{ModuleId: "c", ErrorResult: lib_models.NewErrorResult("deployment not found")},
	}
	if !reflect.DeepEqual(results, want) {
		t.Errorf("results %+v, want %+v", results, want)
	}
	if !reflect.DeepEqual(cew.restarted, []string{"a1", "a2"}) {
		t.Errorf("restarted %v, want [a1 a2]", cew.restarted)
	}
}

func TestHandler_RestartDeployments_ExcludedFromRuntimeMonitor(t *testing.T) {
	h, cew := newRstTestHandler(
		map[string]pkg_models.DeploymentBase{"dep-a": {Id: "dep-a", ModuleId: "a", Enabled: true}},
		map[string]map[string]pkg_models.DeploymentContainerBase{"dep-a": rstTestContainers("dep-a", "a1")},
	)
	results, err := h.RestartDeployments(context.Background(), rstTestSelection("a"))
	if err != nil {
		t.Fatal(err)
	}
	if results[0].HasError {
		t.Fatalf("unexpected error %s", results[0].ErrorMsg)
	}
	if !cew.excluded["a1"] {
		t.Error("deployment must be in runtimeMonitorJobs while its containers are restarted")
	}
	if n := len(h.runtimeMonitorJobs); n != 0 {
		t.Errorf("expected runtimeMonitorJobs to be empty afterwards, got %d entries", n)
	}
	if infos := h.GetRestartInfo([]string{"dep-a"}); len(infos) != 0 {
		t.Errorf("a manual restart must not be recorded, got %+v", infos)
	}
}

func TestHandler_RestartDeployments_SkipsDeploymentInRuntimeMonitorJobs(t *testing.T) {
	h, cew := newRstTestHandler(
		map[string]pkg_models.DeploymentBase{"dep-a": {Id: "dep-a", ModuleId: "a", Enabled: true}},
		map[string]map[string]pkg_models.DeploymentContainerBase{"dep-a": rstTestContainers("dep-a", "a1")},
	)
	h.runtimeMonitorJobsAdd("dep-a")
	results, err := h.RestartDeployments(context.Background(), rstTestSelection("a"))
	if err != nil {
		t.Fatal(err)
	}
	want := []lib_models.DeploymentResult{
		{ModuleId: "a", Id: "dep-a", ErrorResult: lib_models.NewErrorResult("deployment is being started or stopped")},
	}
	if !reflect.DeepEqual(results, want) {
		t.Errorf("results %+v, want %+v", results, want)
	}
	if len(cew.restarted) != 0 {
		t.Errorf("expected no restart, got %v", cew.restarted)
	}
	if _, ok := h.runtimeMonitorJobs["dep-a"]; !ok {
		t.Error("the entry of the other job must be kept")
	}
}

func TestHandler_RestartDeployments_ContainerError(t *testing.T) {
	h, cew := newRstTestHandler(
		map[string]pkg_models.DeploymentBase{"dep-a": {Id: "dep-a", ModuleId: "a", Enabled: true}},
		map[string]map[string]pkg_models.DeploymentContainerBase{"dep-a": rstTestContainers("dep-a", "a1", "a2")},
	)
	cew.failFor["a1"] = errors.New("restart failed")
	results, err := h.RestartDeployments(context.Background(), rstTestSelection("a"))
	if err != nil {
		t.Fatal(err)
	}
	if !results[0].HasError || results[0].ErrorMsg != "'a1' restart failed" {
		t.Errorf("unexpected result %+v", results[0])
	}
	if !reflect.DeepEqual(cew.restarted, []string{"a2"}) {
		t.Errorf("expected the other container to be restarted anyway, got %v", cew.restarted)
	}
	if n := len(h.runtimeMonitorJobs); n != 0 {
		t.Errorf("expected runtimeMonitorJobs to be empty afterwards, got %d entries", n)
	}
}
