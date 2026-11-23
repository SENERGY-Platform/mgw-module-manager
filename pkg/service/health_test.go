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
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	handler_jobs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

func runningContainers() map[string]pkg_models.DeploymentContainer {
	return map[string]pkg_models.DeploymentContainer{
		"main": {State: lib_constants.ContainerRunning},
	}
}

func TestGetDeploymentsHealthInfo_RestartThreshold(t *testing.T) {
	lastRestart := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	deployments := map[string]pkg_models.DeploymentReduced{
		"mod-crash": {
			DeploymentBase: pkg_models.DeploymentBase{Id: "dep-crash", ModuleId: "mod-crash"},
			Containers:     runningContainers(),
			State:          lib_constants.DeploymentHealthy,
		},
		"mod-ok": {
			DeploymentBase: pkg_models.DeploymentBase{Id: "dep-ok", ModuleId: "mod-ok"},
			Containers:     runningContainers(),
			State:          lib_constants.DeploymentHealthy,
		},
	}
	restartInfo := map[string]pkg_models.DeploymentRestartInfo{
		"dep-crash": {Restarts: 3, LastRestart: &lastRestart, LastError: "start failed"},
		"dep-ok":    {Restarts: 2},
	}

	t.Run("at threshold is unhealthy and listed", func(t *testing.T) {
		info := getDeploymentsHealthInfo(deployments, nil, restartInfo, 3, false)
		if info.TotalEnabledDeployments != 2 {
			t.Errorf("expected 2 enabled deployments, got %d", info.TotalEnabledDeployments)
		}
		if len(info.Deployments) != 1 {
			t.Fatalf("expected 1 listed deployment, got %d", len(info.Deployments))
		}
		dep := info.Deployments[0]
		if dep.ModuleId != "mod-crash" || dep.State != lib_constants.DeploymentUnhealthy {
			t.Errorf("unexpected deployment %+v", dep)
		}
		if dep.Restarts != 3 || dep.LastRestart == nil || !dep.LastRestart.Equal(lastRestart) || dep.LastError != "start failed" {
			t.Errorf("unexpected restart info %+v", dep)
		}
		if len(dep.Containers) != 0 {
			t.Errorf("running containers must not be listed, got %+v", dep.Containers)
		}
	})

	t.Run("below threshold is healthy and carries counts when included", func(t *testing.T) {
		info := getDeploymentsHealthInfo(deployments, nil, restartInfo, 3, true)
		if len(info.Deployments) != 2 {
			t.Fatalf("expected 2 listed deployments, got %d", len(info.Deployments))
		}
		for _, dep := range info.Deployments {
			if dep.ModuleId == "mod-ok" && (dep.State != lib_constants.DeploymentHealthy || dep.Restarts != 2 || dep.LastRestart != nil) {
				t.Errorf("unexpected deployment %+v", dep)
			}
		}
	})

	t.Run("threshold zero disables", func(t *testing.T) {
		info := getDeploymentsHealthInfo(deployments, nil, restartInfo, 0, false)
		if len(info.Deployments) != 0 {
			t.Errorf("expected no listed deployments, got %+v", info.Deployments)
		}
	})

	t.Run("without restart info", func(t *testing.T) {
		info := getDeploymentsHealthInfo(deployments, nil, nil, 3, false)
		if len(info.Deployments) != 0 {
			t.Errorf("expected no listed deployments, got %+v", info.Deployments)
		}
	})
}

type healthTestDeploymentsHandler struct {
	deploymentsHandler
	deployments map[string]pkg_models.DeploymentReduced
	restartInfo map[string]pkg_models.DeploymentRestartInfo
}

func (m *healthTestDeploymentsHandler) GetReducedDeploymentsByModuleIds(_ context.Context, _ pkg_models.DeploymentsFilterWithState) (map[string]pkg_models.DeploymentReduced, error) {
	return m.deployments, nil
}

func (m *healthTestDeploymentsHandler) GetRestartInfo(_ []string) map[string]pkg_models.DeploymentRestartInfo {
	return m.restartInfo
}

func (m *healthTestDeploymentsHandler) RestartThreshold() int {
	return 3
}

func newHealthTestService(deployments map[string]pkg_models.DeploymentReduced, restartInfo map[string]pkg_models.DeploymentRestartInfo) *Service {
	return &Service{
		deploymentsHandler: &healthTestDeploymentsHandler{deployments: deployments, restartInfo: restartInfo},
		jobsHandler:        handler_jobs.New(context.Background(), handler_jobs.Config{}),
	}
}

func TestService_DeploymentsHealth_ContainerEngineFailure(t *testing.T) {
	engineErr := errors.New("connection refused")
	srv := newHealthTestService(map[string]pkg_models.DeploymentReduced{
		"mod-a": {
			DeploymentBase: pkg_models.DeploymentBase{Id: "dep-a", ModuleId: "mod-a", Enabled: true, Err: engineErr},
		},
	}, nil)
	_, err := srv.DeploymentsHealth(context.Background(), lib_models.DeploymentsHealthInfoFilter{})
	if err == nil {
		t.Fatal("expected an error if the container states could not be determined")
	}
	if !errors.Is(err, engineErr) {
		t.Errorf("expected error to wrap the engine error, got %v", err)
	}
	// errors without a typed mapping are answered with status 500 by the API
	if lib_errors.IsOf[lib_errors.ErrNotFound](err) || lib_errors.IsOf[lib_errors.ErrExists](err) || lib_errors.IsOf[lib_errors.ErrInvalidInput](err) || lib_errors.IsOf[lib_errors.ErrActiveJob](err) {
		t.Errorf("expected an untyped error, got %T", err)
	}
}

func TestService_DeploymentsHealth_ContainerNotFound(t *testing.T) {
	srv := newHealthTestService(map[string]pkg_models.DeploymentReduced{
		"mod-a": {
			DeploymentBase: pkg_models.DeploymentBase{Id: "dep-a", ModuleId: "mod-a", Enabled: true, Err: errors.New("'main' container 'x' not found")},
			Containers:     map[string]pkg_models.DeploymentContainer{"main": {}},
			State:          lib_constants.DeploymentUnhealthy,
		},
	}, nil)
	info, err := srv.DeploymentsHealth(context.Background(), lib_models.DeploymentsHealthInfoFilter{})
	if err != nil {
		t.Fatalf("a missing container is not an engine failure: %v", err)
	}
	if len(info.Deployments) != 1 || info.Deployments[0].State != lib_constants.DeploymentUnhealthy {
		t.Errorf("expected the deployment to be listed as unhealthy, got %+v", info.Deployments)
	}
}

func TestService_DeploymentsHealth_Restarts(t *testing.T) {
	srv := newHealthTestService(
		map[string]pkg_models.DeploymentReduced{
			"mod-a": {
				DeploymentBase: pkg_models.DeploymentBase{Id: "dep-a", ModuleId: "mod-a", Enabled: true},
				Containers:     runningContainers(),
				State:          lib_constants.DeploymentHealthy,
			},
		},
		map[string]pkg_models.DeploymentRestartInfo{"dep-a": {Restarts: 4, LastError: "boom"}},
	)
	info, err := srv.DeploymentsHealth(context.Background(), lib_models.DeploymentsHealthInfoFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Deployments) != 1 {
		t.Fatalf("expected 1 listed deployment, got %d", len(info.Deployments))
	}
	dep := info.Deployments[0]
	if dep.State != lib_constants.DeploymentUnhealthy || dep.Restarts != 4 || dep.LastError != "boom" {
		t.Errorf("unexpected deployment %+v", dep)
	}
}

func TestGetDeploymentsHealthInfo_HealthOutput(t *testing.T) {
	deployments := map[string]pkg_models.DeploymentReduced{
		"mod": {
			DeploymentBase: pkg_models.DeploymentBase{Id: "dep", ModuleId: "mod"},
			Containers: map[string]pkg_models.DeploymentContainer{
				"main":  {DeploymentContainerBase: pkg_models.DeploymentContainerBase{Reference: "main"}, State: lib_constants.ContainerRunning, Health: lib_constants.ContainerUnhealthy, HealthOutput: "connection refused"},
				"plain": {DeploymentContainerBase: pkg_models.DeploymentContainerBase{Reference: "plain"}, State: lib_constants.ContainerStopped},
			},
			State: lib_constants.DeploymentUnhealthy,
		},
	}
	auxDeployments := map[string]map[string]lib_models.AuxiliaryDeploymentReduced{
		"mod": {"ref": {
			AuxiliaryDeploymentBase: lib_models.AuxiliaryDeploymentBase{Id: "aux"},
			Container:               lib_models.Container{State: lib_constants.ContainerRunning, Health: lib_constants.ContainerUnhealthy, HealthOutput: "aux failed"},
		}}}
	info := getDeploymentsHealthInfo(deployments, auxDeployments, nil, 0, false)
	if len(info.Deployments) != 1 {
		t.Fatalf("expected 1 listed deployment, got %d", len(info.Deployments))
	}
	dep := info.Deployments[0]
	outputs := make(map[string]string)
	for _, c := range dep.Containers {
		outputs[c.Reference] = c.HealthOutput
	}
	if outputs["main"] != "connection refused" || outputs["plain"] != "" {
		t.Errorf("unexpected container health outputs %v", outputs)
	}
	if len(dep.AuxiliaryDeployments) != 1 || dep.AuxiliaryDeployments[0].Container.HealthOutput != "aux failed" {
		t.Errorf("unexpected auxiliary deployments %+v", dep.AuxiliaryDeployments)
	}
	empty, err := json.Marshal(lib_models.ContainerHealthInfo{State: lib_constants.ContainerRunning})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(empty), "health_output") {
		t.Errorf("empty health output must be omitted, got %s", empty)
	}
	full, _ := json.Marshal(lib_models.ContainerHealthInfo{HealthOutput: "x"})
	if !strings.Contains(string(full), `"health_output":"x"`) {
		t.Errorf("health output missing in %s", full)
	}
}
