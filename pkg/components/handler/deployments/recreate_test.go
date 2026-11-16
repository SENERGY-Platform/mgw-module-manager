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
	"reflect"
	"testing"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

// recTestDb extends updTestDb with empty user data for every deployment.
type recTestDb struct {
	updTestDb
}

func (d *recTestDb) ReadDeploymentsHostResources(_ context.Context, _ pkg_models.DeploymentsHostResourcesFilter) (map[string]map[string]pkg_models.DeploymentHostResource, error) {
	return nil, nil
}

func (d *recTestDb) ReadDeploymentsSecrets(_ context.Context, _ pkg_models.DeploymentsSecretsFilter) (map[string]map[string]pkg_models.DeploymentSecret, error) {
	return nil, nil
}

func (d *recTestDb) ReadDeploymentsConfigs(_ context.Context, _ []string) (map[string]map[string]pkg_models.DeploymentUserConfig, error) {
	return nil, nil
}

func (d *recTestDb) ReadDeploymentsGlobalConfigs(_ context.Context, _ pkg_models.DeploymentGlobalConfigsFilter) (map[string]map[string]pkg_models.DeploymentGlobalConfig, error) {
	return nil, nil
}

func (d *recTestDb) ReadDeploymentsFiles(_ context.Context, _ []string) (map[string]map[string]pkg_models.DeploymentFile, error) {
	return nil, nil
}

func (d *recTestDb) ReadDeploymentsFileGroups(_ context.Context, _ []string) (map[string]map[string]pkg_models.DeploymentFileGroup, error) {
	return nil, nil
}

func TestHandler_RecreateDeployments_DependencyOrder(t *testing.T) {
	// Every deployment was made from an older module version, so each recreate fails before anything is changed.
	// a depends on l, c on n, which is not deployed.
	mods, _ := updTestBatch(updTestModule("a", "l"), updTestModule("c", "n"), updTestModule("l"), updTestModule("n"))
	db := &recTestDb{updTestDb{deployments: make(map[string]pkg_models.DeploymentBase)}}
	for _, id := range []string{"a", "c", "l"} {
		db.deployments["dep-"+id] = pkg_models.DeploymentBase{Id: "dep-" + id, ModuleId: id, ModuleVersion: "v0.9.0"}
	}
	h := New(db, nil, nil, nil, nil, Config{})
	results, err := h.RecreateDeployments(context.Background(), mods)
	if err != nil {
		t.Fatal(err)
	}
	want := []lib_models.DeploymentResult{
		updTestResult("l", "module 'l' has changed, deployment must be updated first"),
		updTestResult("a", "dependency 'l' failed"),
		{ModuleId: "n", ErrorResult: lib_models.NewErrorResult("deployment not found")},
		updTestResult("c", "dependency 'n' not deployed"),
	}
	if !reflect.DeepEqual(results, want) {
		t.Errorf("results %+v, want %+v", results, want)
	}
}
