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
	"slices"
	"testing"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

// updTestModule returns a module with the given id depending on deps.
func updTestModule(id string, deps ...string) pkg_models.Module {
	mod := pkg_models.Module{ModuleLibModule: external_models.ModuleLibModule{ID: id, Version: "v1.0.0"}}
	if len(deps) > 0 {
		mod.Dependencies = make(map[string]string)
		for _, dep := range deps {
			mod.Dependencies[dep] = ">=v1.0.0"
		}
	}
	return mod
}

// updTestBatch returns the modules by id and a deployment cache entry "dep-<id>" for each of them.
func updTestBatch(modules ...pkg_models.Module) (map[string]pkg_models.Module, map[string]deploymentsCacheItem) {
	mods := make(map[string]pkg_models.Module)
	cache := make(map[string]deploymentsCacheItem)
	for _, mod := range modules {
		mods[mod.ID] = mod
		cache[mod.ID] = deploymentsCacheItem{DeploymentId: "dep-" + mod.ID}
	}
	return mods, cache
}

// updTestUpdater records the modules it is called for and fails for the ids in fail.
type updTestUpdater struct {
	calls []string
	fail  map[string]bool
}

func (u *updTestUpdater) update(moduleId string, module pkg_models.Module, cacheItem deploymentsCacheItem) error {
	u.calls = append(u.calls, moduleId)
	if module.ID != moduleId || cacheItem.DeploymentId != "dep-"+moduleId {
		return errors.New("wrong module or cache item for " + moduleId)
	}
	if u.fail[moduleId] {
		return errors.New(moduleId + " failed")
	}
	return nil
}

func updTestResult(moduleId, errMsg string) lib_models.DeploymentResult {
	res := lib_models.DeploymentResult{ModuleId: moduleId, Id: "dep-" + moduleId}
	if errMsg != "" {
		res.ErrorResult = lib_models.NewErrorResult(errMsg)
	}
	return res
}

func TestUpdateInDependencyOrder(t *testing.T) {
	t.Run("dependencies first", func(t *testing.T) {
		mods, cache := updTestBatch(
			updTestModule("a", "c", "b"),
			updTestModule("b", "z"),
			updTestModule("c", "z", "outside"),
			updTestModule("x"),
			updTestModule("z"),
		)
		u := &updTestUpdater{}
		results := updateInDependencyOrder(mods, cache, u.update)
		wantCalls := []string{"x", "z", "b", "c", "a"}
		if !reflect.DeepEqual(u.calls, wantCalls) {
			t.Errorf("calls %v, want %v", u.calls, wantCalls)
		}
		var wantResults []lib_models.DeploymentResult
		for _, id := range wantCalls {
			wantResults = append(wantResults, updTestResult(id, ""))
		}
		if !reflect.DeepEqual(results, wantResults) {
			t.Errorf("results %+v, want %+v", results, wantResults)
		}
	})
	t.Run("dependents of a failed deployment are not updated", func(t *testing.T) {
		// a depends on l, b on a, c on l and x, d and x are independent of l
		mods, cache := updTestBatch(
			updTestModule("a", "l"),
			updTestModule("b", "a"),
			updTestModule("c", "x", "l"),
			updTestModule("d"),
			updTestModule("l"),
			updTestModule("x"),
		)
		u := &updTestUpdater{fail: map[string]bool{"l": true}}
		results := updateInDependencyOrder(mods, cache, u.update)
		if wantCalls := []string{"d", "l", "x"}; !reflect.DeepEqual(u.calls, wantCalls) {
			t.Errorf("calls %v, want %v", u.calls, wantCalls)
		}
		want := []lib_models.DeploymentResult{
			updTestResult("d", ""),
			updTestResult("l", "l failed"),
			updTestResult("a", "dependency 'l' failed"),
			updTestResult("b", "dependency 'a' depends on failed 'l'"),
			updTestResult("x", ""),
			updTestResult("c", "dependency 'l' failed"),
		}
		if !reflect.DeepEqual(results, want) {
			t.Errorf("results %+v, want %+v", results, want)
		}
	})
	t.Run("missing deployment blocks its dependents", func(t *testing.T) {
		mods, cache := updTestBatch(updTestModule("a", "n"), updTestModule("n"))
		delete(cache, "n")
		u := &updTestUpdater{}
		results := updateInDependencyOrder(mods, cache, u.update)
		if len(u.calls) != 0 {
			t.Errorf("expected no updates, got %v", u.calls)
		}
		want := []lib_models.DeploymentResult{
			{ModuleId: "n", ErrorResult: lib_models.NewErrorResult("deployment not found")},
			updTestResult("a", "dependency 'n' failed"),
		}
		if !reflect.DeepEqual(results, want) {
			t.Errorf("results %+v, want %+v", results, want)
		}
	})
	t.Run("failure inside a cycle blocks dependents of every cycle member", func(t *testing.T) {
		// p and q depend on each other, p is updated before q fails, r depends on p
		mods, cache := updTestBatch(updTestModule("p", "q"), updTestModule("q", "p"), updTestModule("r", "p"))
		u := &updTestUpdater{fail: map[string]bool{"q": true}}
		results := updateInDependencyOrder(mods, cache, u.update)
		if wantCalls := []string{"p", "q"}; !reflect.DeepEqual(u.calls, wantCalls) {
			t.Errorf("calls %v, want %v", u.calls, wantCalls)
		}
		want := []lib_models.DeploymentResult{
			updTestResult("p", ""),
			updTestResult("q", "q failed"),
			updTestResult("r", "dependency 'p' depends on failed 'q'"),
		}
		if !reflect.DeepEqual(results, want) {
			t.Errorf("results %+v, want %+v", results, want)
		}
	})
	t.Run("empty", func(t *testing.T) {
		if results := updateInDependencyOrder(nil, nil, (&updTestUpdater{}).update); len(results) != 0 {
			t.Errorf("expected no results, got %+v", results)
		}
	})
}

// updTestDb serves deployments by module id, the volumes and containers of every deployment are empty.
type updTestDb struct {
	databaseHandler
	deployments map[string]pkg_models.DeploymentBase
}

func (d *updTestDb) ReadDeployments(_ context.Context, filter pkg_models.DeploymentsFilter) (map[string]pkg_models.DeploymentBase, error) {
	res := make(map[string]pkg_models.DeploymentBase)
	for _, deployment := range d.deployments {
		if slices.Contains(filter.ModuleIds, deployment.ModuleId) {
			res[deployment.Id] = deployment
		}
	}
	return res, nil
}

func (d *updTestDb) ReadDeploymentsContainers(_ context.Context, _ []string) (map[string]map[string]pkg_models.DeploymentContainerBase, error) {
	return nil, nil
}

func (d *updTestDb) ReadDeploymentsVolumes(_ context.Context, _ []string) (map[string]map[string]pkg_models.DeploymentVolume, error) {
	return nil, nil
}

func TestHandler_UpdateDeployments_FailedDependency(t *testing.T) {
	// l and b depend on the module "missing", which is not deployed, so their updates fail before anything is
	// changed. a depends on l and must not be attempted.
	mods, _ := updTestBatch(updTestModule("a", "l"), updTestModule("b", "missing"), updTestModule("l", "missing"))
	db := &updTestDb{deployments: make(map[string]pkg_models.DeploymentBase)}
	for id := range mods {
		db.deployments["dep-"+id] = pkg_models.DeploymentBase{Id: "dep-" + id, ModuleId: id}
	}
	h := New(db, nil, nil, nil, nil, Config{})
	results, err := h.UpdateDeployments(context.Background(), mods, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []lib_models.DeploymentResult{
		updTestResult("b", "dependencies not found: missing"),
		updTestResult("l", "dependencies not found: missing"),
		updTestResult("a", "dependency 'l' failed"),
	}
	if !reflect.DeepEqual(results, want) {
		t.Errorf("results %+v, want %+v", results, want)
	}
}
