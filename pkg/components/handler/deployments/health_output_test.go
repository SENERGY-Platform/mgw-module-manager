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
	"testing"

	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func TestGetContainers_HealthOutput(t *testing.T) {
	output := "connection refused"
	stg := map[string]pkg_models.DeploymentContainerBase{
		"a": {Name: "ctr-a", Reference: "a"},
		"b": {Name: "ctr-b", Reference: "b"},
	}
	cew := map[string]external_models.CewContainer{
		"ctr-a": {HealthOutput: &output},
		"ctr-b": {},
	}
	containers, _ := getContainers(context.Background(), stg, cew)
	if containers["a"].HealthOutput != output {
		t.Errorf("expected health output %q, got %q", output, containers["a"].HealthOutput)
	}
	if containers["b"].HealthOutput != "" {
		t.Errorf("expected empty health output, got %q", containers["b"].HealthOutput)
	}
}
