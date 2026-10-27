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
	"strings"
	"sync"
	"testing"
	"time"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

type testClock struct {
	t time.Time
}

func (c *testClock) now() time.Time { return c.t }

func (c *testClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newRestartTestHandler(window time.Duration, threshold int) (*Handler, *testClock) {
	clock := &testClock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	h := New(nil, nil, nil, nil, nil, Config{RestartWindow: window, RestartThreshold: threshold})
	h.now = clock.now
	return h, clock
}

func TestHandler_RestartInfo_Window(t *testing.T) {
	h, clock := newRestartTestHandler(10*time.Minute, 3)
	t0 := clock.t
	h.recordRestart("a")
	clock.advance(4 * time.Minute)
	h.recordRestart("a")
	clock.advance(4 * time.Minute)
	h.recordRestart("a")
	t2 := clock.t

	info := h.GetRestartInfo([]string{"a"})["a"]
	if info.Restarts != 3 {
		t.Fatalf("expected 3 restarts, got %d", info.Restarts)
	}
	if info.LastRestart == nil || !info.LastRestart.Equal(t2) {
		t.Fatalf("expected last restart %v, got %v", t2, info.LastRestart)
	}

	// t0 is exactly at the window edge and still counts, one nanosecond later it is gone
	clock.t = t0.Add(10 * time.Minute)
	if n := h.GetRestartInfo([]string{"a"})["a"].Restarts; n != 3 {
		t.Fatalf("expected 3 restarts at the window edge, got %d", n)
	}
	clock.t = t0.Add(10*time.Minute + time.Nanosecond)
	if n := h.GetRestartInfo([]string{"a"})["a"].Restarts; n != 2 {
		t.Fatalf("expected 2 restarts after the oldest left the window, got %d", n)
	}

	clock.t = t2.Add(10*time.Minute + time.Nanosecond)
	infos := h.GetRestartInfo([]string{"a"})
	if _, ok := infos["a"]; ok {
		t.Fatalf("expected no entry once all restarts left the window, got %+v", infos["a"])
	}
	if _, ok := h.restarts["a"]; ok {
		t.Fatal("expected record to be removed once empty")
	}
}

func TestHandler_RestartInfo_PrunedOnWrite(t *testing.T) {
	h, clock := newRestartTestHandler(10*time.Minute, 3)
	h.recordRestart("a")
	h.recordRestart("a")
	clock.advance(11 * time.Minute)
	h.recordRestart("a")
	if n := len(h.restarts["a"].starts); n != 1 {
		t.Fatalf("expected stale timestamps to be dropped on write, got %d stored", n)
	}
}

func TestHandler_RestartInfo_PerDeployment(t *testing.T) {
	h, _ := newRestartTestHandler(10*time.Minute, 3)
	h.recordRestart("a")
	h.recordRestart("a")
	h.recordRestart("b")
	infos := h.GetRestartInfo([]string{"a", "b", "c"})
	if infos["a"].Restarts != 2 {
		t.Errorf("expected 2 restarts for a, got %d", infos["a"].Restarts)
	}
	if infos["b"].Restarts != 1 {
		t.Errorf("expected 1 restart for b, got %d", infos["b"].Restarts)
	}
	if _, ok := infos["c"]; ok {
		t.Error("expected no entry for c")
	}
	infos = h.GetRestartInfo([]string{"b"})
	if len(infos) != 1 {
		t.Errorf("expected only requested ids, got %d entries", len(infos))
	}
}

func TestHandler_RestartInfo_LastError(t *testing.T) {
	h, clock := newRestartTestHandler(10*time.Minute, 3)
	h.recordRestart("a")
	h.recordStartError("a", errors.New("first"))
	clock.advance(time.Minute)
	h.recordRestart("a")
	h.recordStartError("a", errors.New("second"))
	h.recordRestart("b")

	infos := h.GetRestartInfo([]string{"a", "b"})
	if infos["a"].LastError != "second" {
		t.Errorf("expected last error 'second', got '%s'", infos["a"].LastError)
	}
	if infos["b"].LastError != "" {
		t.Errorf("expected no error for b, got '%s'", infos["b"].LastError)
	}

	// the old error and restarts leave the window, the new restart keeps the record alive
	clock.advance(11 * time.Minute)
	h.recordRestart("a")
	info := h.GetRestartInfo([]string{"a"})["a"]
	if info.LastError != "" {
		t.Errorf("expected error to leave the window, got '%s'", info.LastError)
	}
	if info.Restarts != 1 {
		t.Errorf("expected 1 restart, got %d", info.Restarts)
	}
}

func TestHandler_RestartInfo_ErrorWithoutRestart(t *testing.T) {
	h, clock := newRestartTestHandler(10*time.Minute, 3)
	h.recordStartError("a", errors.New("boom"))
	info := h.GetRestartInfo([]string{"a"})["a"]
	if info.LastError != "boom" || info.Restarts != 0 || info.LastRestart != nil {
		t.Errorf("unexpected info %+v", info)
	}
	clock.advance(11 * time.Minute)
	if _, ok := h.GetRestartInfo([]string{"a"})["a"]; ok {
		t.Error("expected record to be removed once the error left the window")
	}
}

func TestHandler_RestartInfo_Forget(t *testing.T) {
	h, _ := newRestartTestHandler(10*time.Minute, 3)
	h.recordRestart("a")
	h.recordRestart("b")
	h.forgetRestarts("a")
	infos := h.GetRestartInfo([]string{"a", "b"})
	if _, ok := infos["a"]; ok {
		t.Error("expected a to be forgotten")
	}
	if infos["b"].Restarts != 1 {
		t.Error("expected b to be kept")
	}
}

func TestHandler_RestartThreshold(t *testing.T) {
	h, _ := newRestartTestHandler(time.Minute, 5)
	if h.RestartThreshold() != 5 {
		t.Errorf("expected threshold 5, got %d", h.RestartThreshold())
	}
}

type restartTestDbMock struct {
	databaseHandler
	deployments map[string]pkg_models.DeploymentBase
	containers  map[string]map[string]pkg_models.DeploymentContainerBase
}

func (m *restartTestDbMock) ReadDeployments(_ context.Context, _ pkg_models.DeploymentsFilter) (map[string]pkg_models.DeploymentBase, error) {
	return m.deployments, nil
}

func (m *restartTestDbMock) ReadDeploymentsContainers(_ context.Context, _ []string) (map[string]map[string]pkg_models.DeploymentContainerBase, error) {
	return m.containers, nil
}

func (m *restartTestDbMock) ReadDeploymentsSecrets(_ context.Context, _ pkg_models.DeploymentsSecretsFilter) (map[string]map[string]pkg_models.DeploymentSecret, error) {
	return nil, nil
}

type restartTestCewMock struct {
	containerEngineWrapperClient
	containers []external_models.CewContainer
	startErr   error
	mu         sync.Mutex
	started    []string
}

func (m *restartTestCewMock) GetContainers(_ context.Context, _ external_models.CewContainersFilter) ([]external_models.CewContainer, error) {
	return m.containers, nil
}

func (m *restartTestCewMock) StartContainer(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = append(m.started, id)
	return m.startErr
}

func TestHandler_checkDeployments_RecordsRestarts(t *testing.T) {
	db := &restartTestDbMock{
		deployments: map[string]pkg_models.DeploymentBase{
			"stopped-enabled":  {Id: "stopped-enabled", Enabled: true},
			"running-enabled":  {Id: "running-enabled", Enabled: true},
			"stopped-disabled": {Id: "stopped-disabled", Enabled: false},
		},
		containers: map[string]map[string]pkg_models.DeploymentContainerBase{
			"stopped-enabled":  {"main": {Name: "c1", DeploymentId: "stopped-enabled", Reference: "main"}},
			"running-enabled":  {"main": {Name: "c2", DeploymentId: "running-enabled", Reference: "main"}},
			"stopped-disabled": {"main": {Name: "c3", DeploymentId: "stopped-disabled", Reference: "main"}},
		},
	}
	cew := &restartTestCewMock{
		containers: []external_models.CewContainer{
			{Name: "c1", State: lib_constants.ContainerStopped},
			{Name: "c2", State: lib_constants.ContainerRunning},
			{Name: "c3", State: lib_constants.ContainerStopped},
		},
		startErr: errors.New("start failed"),
	}
	h := New(db, cew, nil, nil, nil, Config{RestartWindow: time.Minute * 10, RestartThreshold: 3})
	ids := []string{"stopped-enabled", "running-enabled", "stopped-disabled"}

	h.checkDeployments(context.Background())

	// startDeployment runs in a goroutine and removes its job entry last
	deadline := time.Now().Add(5 * time.Second)
	for {
		h.runtimeMonitorJobsMu.RLock()
		n := len(h.runtimeMonitorJobs)
		h.runtimeMonitorJobsMu.RUnlock()
		if n == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timeout waiting for runtime monitor jobs")
		}
		time.Sleep(5 * time.Millisecond)
	}

	infos := h.GetRestartInfo(ids)
	info, ok := infos["stopped-enabled"]
	if !ok || info.Restarts != 1 || info.LastRestart == nil {
		t.Fatalf("expected one recorded restart for stopped-enabled, got %+v", info)
	}
	if !strings.Contains(info.LastError, "start failed") {
		t.Errorf("expected start error to be recorded, got '%s'", info.LastError)
	}
	if _, ok = infos["running-enabled"]; ok {
		t.Error("running deployment must not be recorded")
	}
	if _, ok = infos["stopped-disabled"]; ok {
		t.Error("disabled deployment must not be recorded")
	}
	if len(cew.started) != 1 || cew.started[0] != "c1" {
		t.Errorf("expected only c1 to be started, got %v", cew.started)
	}
}
