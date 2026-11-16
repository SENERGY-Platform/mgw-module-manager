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
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

// auSecretModfile declares an optional certificate secret that is a user input.
const auSecretModfile = `secrets:
  cert:
    type: certificate
    optional: true
    userInput:
      name: Cert
    targets:
      - mountPoint: /opt/cert
        services:
          - srv
`

func auPlan(t *testing.T, env *auTestEnv, level string) autoUpdatePlan {
	t.Helper()
	plan, err := auPlanCtx(context.Background(), env, level)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

// auPlanCtx plans as prepareAutoUpdate does, with the secret types read before.
func auPlanCtx(ctx context.Context, env *auTestEnv, level string) (autoUpdatePlan, error) {
	return env.srv.planAutoUpdate(ctx, level, env.srv.readSecretTypesOf(ctx, nil))
}

func TestService_planAutoUpdate_DependencyDeployments(t *testing.T) {
	t.Run("deployed module with an undeployed dependency is skipped", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", []string{"mod-b"}, "")
		env.offer("mod-a", "v1.0.1", []string{"mod-b"}, "")
		env.install(t, "mod-b", "v2.0.0", nil, "")
		env.offer("mod-b", "v2.0.0", nil, "")
		env.deploy("mod-a", nil)
		plan := auPlan(t, env, "patch")
		if plan.changeRequest != nil || auSkipped(plan)["mod-a"] != lib_constants.AutoUpdateReasonDependencyNotDeployed {
			t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
		}
	})
	t.Run("new version declares an installed but undeployed dependency", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", nil, "")
		env.offer("mod-a", "v1.0.1", []string{"mod-b"}, "")
		env.install(t, "mod-b", "v2.0.0", nil, "")
		env.offer("mod-b", "v2.0.0", nil, "")
		env.deploy("mod-a", nil)
		plan := auPlan(t, env, "patch")
		if plan.changeRequest != nil || auSkipped(plan)["mod-a"] != lib_constants.AutoUpdateReasonDependencyNotDeployed {
			t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
		}
	})
	t.Run("dependency deployment unreadable", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", []string{"mod-b"}, "")
		env.offer("mod-a", "v1.0.1", []string{"mod-b"}, "")
		env.install(t, "mod-b", "v2.0.0", nil, "")
		env.offer("mod-b", "v2.0.0", nil, "")
		env.deploy("mod-a", nil)
		env.deploy("mod-b", func(d *pkg_models.Deployment) { d.Err = errors.New("broken") })
		plan := auPlan(t, env, "patch")
		if plan.changeRequest != nil || auSkipped(plan)["mod-a"] != lib_constants.AutoUpdateReasonDeploymentError {
			t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
		}
	})
	t.Run("deployed dependency", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", []string{"mod-b"}, "")
		env.offer("mod-a", "v1.0.1", []string{"mod-b"}, "")
		env.install(t, "mod-b", "v2.0.0", nil, "")
		env.offer("mod-b", "v2.0.0", nil, "")
		env.deploy("mod-a", nil)
		env.deploy("mod-b", nil)
		plan := auPlan(t, env, "patch")
		if got := auChanged(plan); !reflect.DeepEqual(got, []string{"mod-a@v1.0.1"}) || len(plan.skipped) != 0 {
			t.Errorf("unexpected plan changed=%v skipped=%+v", got, plan.skipped)
		}
	})
	t.Run("undeployed module needs no deployed dependency", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", []string{"mod-b"}, "")
		env.offer("mod-a", "v1.0.1", []string{"mod-b"}, "")
		env.install(t, "mod-b", "v2.0.0", nil, "")
		env.offer("mod-b", "v2.0.0", nil, "")
		plan := auPlan(t, env, "patch")
		if got := auChanged(plan); !reflect.DeepEqual(got, []string{"mod-a@v1.0.1"}) || len(plan.skipped) != 0 {
			t.Errorf("unexpected plan changed=%v skipped=%+v", got, plan.skipped)
		}
	})
}

// The installed declaration is checked again before the deployment is updated.
func TestService_autoUpdateTick_DependencyNotDeployedAfterModuleUpdate(t *testing.T) {
	env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
	env.install(t, "mod-a", "v1.0.0", nil, "")
	env.offer("mod-a", "v1.0.1", nil, "")
	env.install(t, "mod-c", "v1.0.0", nil, "")
	env.deploy("mod-a", nil)
	env.mods.afterUpdate = func(m *pkg_models.Module) {
		m.Dependencies = map[string]string{"mod-c": ">=v0.0.1"}
	}
	if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateRan {
		t.Fatalf("expected run, got %v", outcome)
	}
	if got := env.deps.updatedIds(); len(got) != 0 {
		t.Errorf("deployment updated without a deployed dependency: %v", got)
	}
	run := env.db.get().LastRun
	want := []lib_models.AutoUpdateFailedModule{{Id: "mod-a", Error: "deployment not updated: dependency 'mod-c' not deployed"}}
	if run == nil || len(run.Updated) != 0 || !reflect.DeepEqual(run.Failed, want) {
		t.Errorf("unexpected run %+v", run)
	}
	auAssertIdle(t, env.srv)
}

func TestService_planAutoUpdate_SecretTypes(t *testing.T) {
	setup := func(t *testing.T) *auTestEnv {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", nil, auSecretModfile)
		env.offer("mod-a", "v1.0.1", nil, auSecretModfile)
		env.deploy("mod-a", func(d *pkg_models.Deployment) {
			d.Secrets = map[string]pkg_models.DeploymentSecret{"cert": {Id: "s-1"}}
		})
		return env
	}
	cases := []struct {
		name   string
		types  map[string]string
		err    error
		reason string
	}{
		{"declared type", map[string]string{"s-1": "certificate"}, nil, ""},
		{"type changed", map[string]string{"s-1": "basic-auth"}, nil, lib_constants.CarryOverReasonTypeMismatch},
		{"secret deleted", map[string]string{}, nil, lib_constants.CarryOverReasonSecretNotFound},
		{"secret manager unreachable", nil, errors.New("connection refused"), lib_constants.CarryOverReasonSecretTypeUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := setup(t)
			env.deps.secretTypes, env.deps.secretTypesErr = c.types, c.err
			plan := auPlan(t, env, "patch")
			if c.reason == "" {
				if got := auChanged(plan); !reflect.DeepEqual(got, []string{"mod-a@v1.0.1"}) || len(plan.skipped) != 0 {
					t.Errorf("unexpected plan changed=%v skipped=%+v", got, plan.skipped)
				}
				return
			}
			want := []lib_models.CarryOverMissing{{Kind: lib_constants.CarryOverKindSecret, Ref: "cert", Reason: c.reason}}
			if plan.changeRequest != nil || len(plan.skipped) != 1 || plan.skipped[0].Reason != lib_constants.AutoUpdateReasonNeedsInput ||
				!reflect.DeepEqual(plan.skipped[0].Missing, want) {
				t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
			}
		})
	}
}

func TestService_planAutoUpdate_DependentRanges(t *testing.T) {
	t.Run("dependency moved outside the range of an installed dependent", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", []string{"mod-b@<v2.0.1"}, "")
		env.offer("mod-a", "v1.0.0", []string{"mod-b@<v2.0.1"}, "")
		env.install(t, "mod-b", "v2.0.0", nil, "")
		env.offer("mod-b", "v2.0.1", nil, "")
		plan := auPlan(t, env, "patch")
		if plan.changeRequest != nil || auSkipped(plan)["mod-b"] != lib_constants.AutoUpdateReasonDependentRangeViolated {
			t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
		}
	})
	// The dependent sorts before and after its dependency, so its change set is planned first and last.
	for _, dependent := range []string{"mod-a", "mod-z"} {
		t.Run("dependent moved along to a version accepting it, "+dependent, func(t *testing.T) {
			env := newAutoUpdateTestEnv(t, nil)
			env.install(t, dependent, "v1.0.0", []string{"mod-b@<v2.0.1"}, "")
			env.offer(dependent, "v1.0.1", []string{"mod-b@>=v2.0.1"}, "")
			env.install(t, "mod-b", "v2.0.0", nil, "")
			env.offer("mod-b", "v2.0.1", nil, "")
			plan := auPlan(t, env, "patch")
			want := []string{dependent + "@v1.0.1", "mod-b@v2.0.1"}
			if dependent > "mod-b" {
				want = []string{"mod-b@v2.0.1", dependent + "@v1.0.1"}
			}
			if got := auChanged(plan); !reflect.DeepEqual(got, want) || len(plan.skipped) != 0 {
				t.Errorf("unexpected plan changed=%v skipped=%+v", got, plan.skipped)
			}
		})
	}
	t.Run("new range of a moved module rejects its unmoved dependency", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", []string{"mod-b"}, "")
		env.offer("mod-a", "v1.0.1", []string{"mod-b@>=v2.1.0"}, "")
		env.install(t, "mod-b", "v2.0.0", nil, "")
		env.offer("mod-b", "v2.0.0", nil, "")
		plan := auPlan(t, env, "patch")
		if plan.changeRequest != nil || auSkipped(plan)["mod-a"] != lib_constants.AutoUpdateReasonDependentRangeViolated {
			t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
		}
	})
}

func TestAutoUpdateRangeViolated(t *testing.T) {
	mod := func(version string, deps map[string]string) pkg_models.Module {
		m := pkg_models.Module{}
		m.Version = version
		m.Dependencies = deps
		return m
	}
	change := func(id, version string, deps map[string]string) changeItem {
		item := changeItem{}
		item.Next.Mod.ID = id
		item.Next.Mod.Version = version
		item.Next.Mod.Dependencies = deps
		return item
	}
	cases := []struct {
		name      string
		installed map[string]pkg_models.Module
		changes   map[string]changeItem
		want      bool
	}{
		{
			name: "violation that the changes do not touch is left alone",
			installed: map[string]pkg_models.Module{
				"a": mod("v1.0.0", map[string]string{"b": "<v1.0.0"}),
				"b": mod("v1.0.0", nil),
				"c": mod("v1.0.0", nil),
			},
			changes: map[string]changeItem{"c": change("c", "v1.0.1", nil)},
			want:    false,
		},
		{
			name: "upper bound of an installed dependent",
			installed: map[string]pkg_models.Module{
				"a": mod("v1.0.0", map[string]string{"b": ">=v1.0.0;<v1.1.0"}),
				"b": mod("v1.0.0", nil),
			},
			changes: map[string]changeItem{"b": change("b", "v1.1.0", nil)},
			want:    true,
		},
		{
			name: "within the range of an installed dependent",
			installed: map[string]pkg_models.Module{
				"a": mod("v1.0.0", map[string]string{"b": ">=v1.0.0;<v1.1.0"}),
				"b": mod("v1.0.0", nil),
			},
			changes: map[string]changeItem{"b": change("b", "v1.0.9", nil)},
			want:    false,
		},
		{
			name: "invalid range counts as violated",
			installed: map[string]pkg_models.Module{
				"a": mod("v1.0.0", map[string]string{"b": "latest"}),
				"b": mod("v1.0.0", nil),
			},
			changes: map[string]changeItem{"b": change("b", "v1.0.1", nil)},
			want:    true,
		},
		{
			name: "unreadable dependency of a moved module counts as violated",
			installed: map[string]pkg_models.Module{
				"a": mod("v1.0.0", nil),
				"b": {Err: errors.New("broken")},
			},
			changes: map[string]changeItem{"a": change("a", "v1.0.1", map[string]string{"b": ">=v0.0.1"})},
			want:    true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := autoUpdateRangeViolated(c.installed, c.changes); got != c.want {
				t.Errorf("want %v, got %v", c.want, got)
			}
		})
	}
}

func TestService_planAutoUpdate_LaggingDeployments(t *testing.T) {
	t.Run("lagging deployment within level is repaired", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.1", nil, "")
		env.deployAt("mod-a", "v1.0.0", nil)
		plan := auPlan(t, env, "patch")
		want := []autoUpdateRepair{{id: "mod-a", from: "v1.0.0", to: "v1.0.1", source: auSource, channel: auChannel}}
		if plan.changeRequest != nil || !reflect.DeepEqual(plan.repairs, want) || len(plan.skipped)+len(plan.failed) != 0 {
			t.Errorf("unexpected plan changed=%v repairs=%+v skipped=%+v failed=%+v", auChanged(plan), plan.repairs, plan.skipped, plan.failed)
		}
	})
	t.Run("current, newer or other channel deployments are left alone", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.1", nil, "")
		env.deployAt("mod-a", "v1.0.1", nil)
		env.install(t, "mod-b", "v1.0.1", nil, "")
		env.deployAt("mod-b", "v1.0.2", nil)
		env.install(t, "mod-c", "v1.0.1", nil, "")
		env.deployAt("mod-c", "v1.0.0", func(d *pkg_models.Deployment) { d.ModuleChannel = "dev" })
		plan := auPlan(t, env, "major")
		if len(plan.repairs)+len(plan.skipped)+len(plan.failed) != 0 {
			t.Errorf("unexpected plan repairs=%+v skipped=%+v failed=%+v", plan.repairs, plan.skipped, plan.failed)
		}
	})
	cases := []struct {
		name   string
		setup  func(env *auTestEnv)
		reason string
	}{
		{"level exceeded", func(env *auTestEnv) {
			env.install(t, "mod-a", "v1.1.0", nil, "")
			env.deployAt("mod-a", "v1.0.0", nil)
		}, lib_constants.AutoUpdateReasonLevelExceeded},
		{"invalid deployment version", func(env *auTestEnv) {
			env.install(t, "mod-a", "v1.0.1", nil, "")
			env.deployAt("mod-a", "1.0.0", nil)
		}, lib_constants.AutoUpdateReasonInvalidVersion},
		{"deployment error", func(env *auTestEnv) {
			env.install(t, "mod-a", "v1.0.1", nil, "")
			env.deployAt("mod-a", "v1.0.0", func(d *pkg_models.Deployment) { d.Err = errors.New("broken") })
		}, lib_constants.AutoUpdateReasonDeploymentError},
		{"needs input against the installed declaration", func(env *auTestEnv) {
			env.install(t, "mod-a", "v1.0.1", nil, auConfigModfile)
			env.deployAt("mod-a", "v1.0.0", func(d *pkg_models.Deployment) {
				d.Configs = map[string]pkg_models.DeploymentUserConfig{"cfg": {Value: coStringValue("")}}
			})
		}, lib_constants.AutoUpdateReasonNeedsInput},
		{"dependency not deployed", func(env *auTestEnv) {
			env.install(t, "mod-a", "v1.0.1", []string{"mod-b"}, "")
			env.install(t, "mod-b", "v1.0.0", nil, "")
			env.deployAt("mod-a", "v1.0.0", nil)
		}, lib_constants.AutoUpdateReasonDependencyNotDeployed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := newAutoUpdateTestEnv(t, nil)
			c.setup(env)
			plan := auPlan(t, env, "patch")
			if len(plan.repairs) != 0 || len(plan.skipped) != 1 || plan.skipped[0].Id != "mod-a" || plan.skipped[0].Reason != c.reason {
				t.Fatalf("unexpected plan repairs=%+v skipped=%+v failed=%+v", plan.repairs, plan.skipped, plan.failed)
			}
			if s := plan.skipped[0]; s.Version != env.deps.deployments["mod-a"].ModuleVersion || s.NextVersion != env.mods.installed["mod-a"].Version {
				t.Errorf("skipped entry must name the deployment and the installed version, got %+v", s)
			}
		})
	}
	t.Run("update of a module whose deployment lags beyond level is skipped", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.1", nil, "")
		env.offer("mod-a", "v1.0.2", nil, "")
		env.deployAt("mod-a", "v0.9.0", nil)
		plan := auPlan(t, env, "patch")
		if plan.changeRequest != nil || len(plan.repairs) != 0 {
			t.Fatalf("unexpected plan changed=%v repairs=%+v", auChanged(plan), plan.repairs)
		}
		for _, s := range plan.skipped {
			if s.Reason != lib_constants.AutoUpdateReasonLevelExceeded {
				t.Errorf("unexpected skipped %+v", s)
			}
		}
	})
	t.Run("lagging dependency deployment beyond level skips the dependent", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", []string{"mod-b"}, "")
		env.offer("mod-a", "v1.0.1", []string{"mod-b"}, "")
		env.install(t, "mod-b", "v2.0.0", nil, "")
		env.offer("mod-b", "v2.0.1", nil, "")
		env.deployAt("mod-b", "v1.9.0", nil)
		plan := auPlan(t, env, "patch")
		if plan.changeRequest != nil || auSkipped(plan)["mod-a"] != lib_constants.AutoUpdateReasonDependencyLevelExceeded {
			t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
		}
	})
}

func TestService_autoUpdateTick_RepairsLaggingDeployment(t *testing.T) {
	env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
	env.install(t, "mod-a", "v1.0.1", nil, auConfigModfile)
	env.deployAt("mod-a", "v1.0.0", func(d *pkg_models.Deployment) {
		d.Configs = map[string]pkg_models.DeploymentUserConfig{"cfg": {Value: coStringValue("kept")}}
	})
	var slotsDuringDeploymentUpdate []int
	env.deps.onUpdate = func() {
		for slot := range env.srv.jobsHandler.CurrentSlotJobs(nil) {
			slotsDuringDeploymentUpdate = append(slotsDuringDeploymentUpdate, slot)
		}
	}
	if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateRan {
		t.Fatalf("expected run, got %v", outcome)
	}
	if got := env.mods.updateList(); len(got) != 0 {
		t.Errorf("modules changed by a deployment repair: %v", got)
	}
	if !reflect.DeepEqual(slotsDuringDeploymentUpdate, []int{deploymentJobSlotNum}) {
		t.Errorf("expected the deployment job alone, occupied slots: %v", slotsDuringDeploymentUpdate)
	}
	if v := env.deps.updated["mod-a"].Configs["cfg"]; v.String != "kept" {
		t.Errorf("deployment setup not carried over, got %+v", v)
	}
	run := env.db.get().LastRun
	want := []lib_models.AutoUpdateUpdatedModule{{Id: "mod-a", From: "v1.0.0", To: "v1.0.1"}}
	if run == nil || !reflect.DeepEqual(run.Updated, want) || len(run.Failed)+len(run.Skipped) != 0 {
		t.Errorf("unexpected run %+v", run)
	}
	if env.srv.changeRequest != nil {
		t.Error("a deployment repair set a change request")
	}
	auAssertIdle(t, env.srv)
}

func TestService_autoUpdateTick_CancelledDeploymentJob(t *testing.T) {
	env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
	env.install(t, "mod-a", "v1.0.0", nil, "")
	env.offer("mod-a", "v1.0.1", nil, "")
	env.deployAt("mod-a", "v1.0.0", nil)
	env.install(t, "mod-b", "v1.0.0", nil, "")
	env.offer("mod-b", "v1.0.1", nil, "")
	env.deployAt("mod-b", "v1.0.0", nil)
	ctx := context.Background()
	env.mods.afterUpdate = func(_ *pkg_models.Module) {
		if job, ok := env.srv.jobsHandler.CurrentSlotJob(deploymentJobSlotNum); ok {
			job.Cancel()
		}
	}
	if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateRan {
		t.Fatalf("expected run, got %v", outcome)
	}
	if got := env.mods.updateList(); !reflect.DeepEqual(got, []string{"mod-a@v1.0.1"}) {
		t.Errorf("modules applied after the deployment job was cancelled: %v", got)
	}
	if got := env.deps.updatedIds(); len(got) != 0 {
		t.Errorf("deployment update started after the deployment job was cancelled: %v", got)
	}
	run := env.db.get().LastRun
	wantFailed := []lib_models.AutoUpdateFailedModule{
		{Id: "mod-a", Error: autoUpdateDeploymentCancelled},
		{Id: "mod-b", Error: "not applied: " + autoUpdateDeploymentCancelled},
	}
	if run == nil || len(run.Updated) != 0 || !reflect.DeepEqual(run.Failed, wantFailed) {
		t.Errorf("unexpected run %+v", run)
	}
	auAssertIdle(t, env.srv)

	// The next window completes the update: the deployment of mod-a lags its module, mod-b is still a candidate.
	env.mods.afterUpdate = nil
	env.setNow(env.getNow().Add(24 * time.Hour))
	if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateRan {
		t.Fatalf("next window: expected run, got %v", outcome)
	}
	if got := env.deps.updatedIds(); !reflect.DeepEqual(got, []string{"mod-a", "mod-b"}) {
		t.Errorf("next window: unexpected deployment updates %v", got)
	}
	run = env.db.get().LastRun
	wantUpdated := []lib_models.AutoUpdateUpdatedModule{{Id: "mod-a", From: "v1.0.0", To: "v1.0.1"}, {Id: "mod-b", From: "v1.0.0", To: "v1.0.1"}}
	if run == nil || !reflect.DeepEqual(run.Updated, wantUpdated) || len(run.Failed)+len(run.Skipped) != 0 {
		t.Errorf("next window: unexpected run %+v", run)
	}
	for _, id := range []string{"mod-a", "mod-b"} {
		if v := env.deps.deployments[id].ModuleVersion; v != "v1.0.1" {
			t.Errorf("%s: deployment still at %s", id, v)
		}
	}
	auAssertIdle(t, env.srv)
}

func TestService_autoUpdateTick_CancelledModuleJob(t *testing.T) {
	env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
	env.install(t, "mod-a", "v1.0.0", nil, "")
	env.offer("mod-a", "v1.0.1", nil, "")
	env.deploy("mod-a", nil)
	env.install(t, "mod-b", "v1.0.0", nil, "")
	env.offer("mod-b", "v1.0.1", nil, "")
	env.deploy("mod-b", nil)
	env.mods.afterUpdate = func(_ *pkg_models.Module) {
		if job, ok := env.srv.jobsHandler.CurrentSlotJob(moduleJobSlotNum); ok {
			job.Cancel()
		}
	}
	if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateRan {
		t.Fatalf("expected run, got %v", outcome)
	}
	if got := env.mods.updateList(); !reflect.DeepEqual(got, []string{"mod-a@v1.0.1"}) {
		t.Errorf("modules applied after the module job was cancelled: %v", got)
	}
	if got := env.deps.updatedIds(); !reflect.DeepEqual(got, []string{"mod-a"}) {
		t.Errorf("the deployment of the applied module must still be updated, got %v", got)
	}
	run := env.db.get().LastRun
	wantUpdated := []lib_models.AutoUpdateUpdatedModule{{Id: "mod-a", From: "v1.0.0", To: "v1.0.1"}}
	wantFailed := []lib_models.AutoUpdateFailedModule{{Id: "mod-b", Error: "not applied: context canceled"}}
	if run == nil || !reflect.DeepEqual(run.Updated, wantUpdated) || !reflect.DeepEqual(run.Failed, wantFailed) {
		t.Errorf("unexpected run %+v", run)
	}
	auAssertIdle(t, env.srv)
}

func TestService_autoUpdateTick_PrepareFailureReleasesJobs(t *testing.T) {
	for _, c := range []struct {
		name  string
		hook  func() error
		panic bool
	}{
		{"error", func() error { return errors.New("database unavailable") }, false},
		{"panic", func() error { panic("database driver") }, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
			env.install(t, "mod-a", "v1.0.0", nil, "")
			env.offer("mod-a", "v1.0.1", nil, "")
			env.deploy("mod-a", nil)
			var reserved int
			env.db.onRunDate = func() error {
				reserved = len(env.srv.jobsHandler.CurrentSlotJobs(nil))
				return c.hook()
			}
			if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateError {
				t.Fatalf("expected error, got %v", outcome)
			}
			if reserved != 2 {
				t.Fatalf("expected both slots reserved before the run date is written, got %d", reserved)
			}
			auAssertIdle(t, env.srv)
			if env.srv.changeRequest != nil {
				t.Error("change request left behind")
			}
			if len(env.mods.updateList()) != 0 {
				t.Error("modules updated")
			}
			env.db.onRunDate = nil
			if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateRetryLater {
				t.Errorf("expected the next attempt to wait, got %v", outcome)
			}
		})
	}
}

// A module of the change request has its deployment updated with the change, not repaired on its own.
func TestService_planAutoUpdate_NoRepairForModulesOfTheChangeRequest(t *testing.T) {
	env := newAutoUpdateTestEnv(t, nil)
	env.install(t, "mod-a", "v1.0.1", nil, "")
	env.offer("mod-a", "v1.0.2", nil, "")
	env.deployAt("mod-a", "v1.0.0", nil)
	plan := auPlan(t, env, "patch")
	if got := auChanged(plan); !reflect.DeepEqual(got, []string{"mod-a@v1.0.2"}) || len(plan.repairs) != 0 || plan.deployed["mod-a"] != "v1.0.0" {
		t.Errorf("unexpected plan changed=%v repairs=%+v", got, plan.repairs)
	}
}

func TestService_autoUpdateTick_RepairedModuleChangedMeanwhile(t *testing.T) {
	env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
	env.install(t, "mod-a", "v1.0.1", nil, "")
	env.deployAt("mod-a", "v1.0.0", nil)
	// runs after planning, before the deployment phase
	env.db.onRunDate = func() error {
		env.mods.mu.Lock()
		defer env.mods.mu.Unlock()
		m := env.mods.installed["mod-a"]
		m.Version = "v1.0.2"
		env.mods.installed["mod-a"] = m
		return nil
	}
	if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateRan {
		t.Fatalf("expected run, got %v", outcome)
	}
	if got := env.deps.updatedIds(); len(got) != 0 {
		t.Errorf("deployment updated to a module version that was not checked: %v", got)
	}
	run := env.db.get().LastRun
	if run == nil || len(run.Updated) != 0 || len(run.Failed) != 1 || !strings.Contains(run.Failed[0].Error, "module changed") {
		t.Errorf("unexpected run %+v", run)
	}
	auAssertIdle(t, env.srv)
}

// A cancellation that arrives after the deployment phase started still keeps the deployments from being updated.
func TestService_autoUpdateTick_DeploymentJobCancelledInDeploymentPhase(t *testing.T) {
	env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
	env.install(t, "mod-a", "v1.0.1", nil, "")
	env.deployAt("mod-a", "v1.0.0", nil)
	// The deployment job is reserved after planning, the first read while it exists starts the deployment phase.
	env.deps.onGet = func(_ int) {
		if job, ok := env.srv.jobsHandler.CurrentSlotJob(deploymentJobSlotNum); ok {
			job.Cancel()
		}
	}
	if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateRan {
		t.Fatalf("expected run, got %v", outcome)
	}
	if got := env.deps.updatedIds(); len(got) != 0 {
		t.Errorf("deployment updated after the deployment job was cancelled: %v", got)
	}
	run := env.db.get().LastRun
	want := []lib_models.AutoUpdateFailedModule{{Id: "mod-a", Error: "deployment not updated: " + autoUpdateDeploymentCancelled}}
	if run == nil || len(run.Updated) != 0 || !reflect.DeepEqual(run.Failed, want) {
		t.Errorf("unexpected run %+v", run)
	}
	auAssertIdle(t, env.srv)
}

// A deployment the run cannot move forward within its source and channel is left to the user, as in the repair.
func TestService_planAutoUpdate_DeploymentVariantDiffers(t *testing.T) {
	cases := []struct {
		name  string
		setup func(env *auTestEnv)
	}{
		{"deployment of another channel", func(env *auTestEnv) {
			env.install(t, "mod-a", "v1.0.0", nil, "")
			env.offer("mod-a", "v1.0.1", nil, "")
			env.deployAt("mod-a", "v1.0.0", func(d *pkg_models.Deployment) { d.ModuleChannel = "dev" })
		}},
		{"deployment of another source", func(env *auTestEnv) {
			env.install(t, "mod-a", "v1.0.0", nil, "")
			env.offer("mod-a", "v1.0.1", nil, "")
			env.deployAt("mod-a", "v1.0.0", func(d *pkg_models.Deployment) { d.ModuleSource = "other" })
		}},
		{"dependency deployment of another channel", func(env *auTestEnv) {
			env.install(t, "mod-a", "v1.0.0", []string{"mod-b"}, "")
			env.offer("mod-a", "v1.0.1", []string{"mod-b"}, "")
			env.install(t, "mod-b", "v2.0.0", nil, "")
			env.offer("mod-b", "v2.0.1", nil, "")
			env.deployAt("mod-a", "v1.0.0", nil)
			env.deployAt("mod-b", "v2.0.0", func(d *pkg_models.Deployment) { d.ModuleChannel = "dev" })
		}},
		{"deployment at the new version", func(env *auTestEnv) {
			env.install(t, "mod-a", "v1.0.1", nil, "")
			env.offer("mod-a", "v1.0.2", nil, "")
			env.deployAt("mod-a", "v1.0.2", nil)
		}},
		{"deployment ahead of the new version", func(env *auTestEnv) {
			env.install(t, "mod-a", "v1.0.1", nil, "")
			env.offer("mod-a", "v1.0.2", nil, "")
			env.deployAt("mod-a", "v1.0.3", nil)
		}},
		{"dependency deployment at the new version", func(env *auTestEnv) {
			env.install(t, "mod-a", "v1.0.0", []string{"mod-b"}, "")
			env.offer("mod-a", "v1.0.1", []string{"mod-b"}, "")
			env.install(t, "mod-b", "v2.0.0", nil, "")
			env.offer("mod-b", "v2.0.1", nil, "")
			env.deployAt("mod-a", "v1.0.0", nil)
			env.deployAt("mod-b", "v2.0.1", nil)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := newAutoUpdateTestEnv(t, nil)
			c.setup(env)
			plan := auPlan(t, env, "patch")
			if plan.changeRequest != nil || len(plan.repairs) != 0 || auSkipped(plan)["mod-a"] != lib_constants.AutoUpdateReasonDeploymentVariantDiffers {
				t.Fatalf("unexpected plan changed=%v repairs=%+v skipped=%+v", auChanged(plan), plan.repairs, plan.skipped)
			}
			for _, s := range plan.skipped {
				if s.Reason != lib_constants.AutoUpdateReasonDeploymentVariantDiffers {
					t.Errorf("unexpected skipped %+v", s)
				}
			}
		})
	}
}

func TestService_autoUpdateTick_FailedModuleUpdateFallback(t *testing.T) {
	t.Run("lagging deployment updated to the installed version", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
		env.install(t, "mod-a", "v1.0.1", nil, "")
		env.offer("mod-a", "v1.0.2", nil, "")
		env.deployAt("mod-a", "v1.0.0", nil)
		env.mods.updateErr = map[string]error{"mod-a": errors.New("disk full")}
		if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateRan {
			t.Fatalf("expected run, got %v", outcome)
		}
		if got := env.mods.updateList(); len(got) != 0 {
			t.Errorf("unexpected module updates %v", got)
		}
		if v := env.deps.deployments["mod-a"].ModuleVersion; v != "v1.0.1" {
			t.Errorf("deployment at %s, want the installed version", v)
		}
		run := env.db.get().LastRun
		wantUpdated := []lib_models.AutoUpdateUpdatedModule{{Id: "mod-a", From: "v1.0.0", To: "v1.0.1"}}
		wantFailed := []lib_models.AutoUpdateFailedModule{{Id: "mod-a", Error: "disk full"}}
		if run == nil || !reflect.DeepEqual(run.Updated, wantUpdated) || !reflect.DeepEqual(run.Failed, wantFailed) || len(run.Skipped) != 0 {
			t.Errorf("unexpected run %+v", run)
		}
		auAssertIdle(t, env.srv)
	})
	t.Run("lagging deployment skipped against the installed declaration", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
		// The installed version needs a deployed mod-b, the new one does not.
		env.install(t, "mod-a", "v1.0.1", []string{"mod-b"}, "")
		env.install(t, "mod-b", "v1.0.0", nil, "")
		env.offer("mod-a", "v1.0.2", nil, "")
		env.deployAt("mod-a", "v1.0.0", nil)
		env.mods.updateErr = map[string]error{"mod-a": errors.New("disk full")}
		if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateRan {
			t.Fatalf("expected run, got %v", outcome)
		}
		if got := env.deps.updatedIds(); len(got) != 0 {
			t.Errorf("unexpected deployment updates %v", got)
		}
		run := env.db.get().LastRun
		wantSkipped := []lib_models.AutoUpdateSkippedModule{{
			Id: "mod-a", Version: "v1.0.0", NextVersion: "v1.0.1", Reason: lib_constants.AutoUpdateReasonDependencyNotDeployed,
		}}
		wantFailed := []lib_models.AutoUpdateFailedModule{{Id: "mod-a", Error: "disk full"}}
		if run == nil || len(run.Updated) != 0 || !reflect.DeepEqual(run.Skipped, wantSkipped) || !reflect.DeepEqual(run.Failed, wantFailed) {
			t.Errorf("unexpected run %+v", run)
		}
		auAssertIdle(t, env.srv)
	})
	// The update reports the version the deployment moves from, the fallback is not reported.
	t.Run("not applied after a successful update", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
		env.install(t, "mod-a", "v1.0.1", []string{"mod-b"}, "")
		env.install(t, "mod-b", "v1.0.0", nil, "")
		env.offer("mod-a", "v1.0.2", nil, "")
		env.deployAt("mod-a", "v1.0.0", nil)
		if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateRan {
			t.Fatalf("expected run, got %v", outcome)
		}
		run := env.db.get().LastRun
		wantUpdated := []lib_models.AutoUpdateUpdatedModule{{Id: "mod-a", From: "v1.0.0", To: "v1.0.2"}}
		if run == nil || !reflect.DeepEqual(run.Updated, wantUpdated) || len(run.Skipped)+len(run.Failed) != 0 {
			t.Errorf("unexpected run %+v", run)
		}
		if v := env.deps.deployments["mod-a"].ModuleVersion; v != "v1.0.2" {
			t.Errorf("deployment at %s", v)
		}
		auAssertIdle(t, env.srv)
	})
}

// auHeldLock reports whether s.mu is locked, for reading or writing, by anyone else.
func auHeldLock(srv *Service) bool {
	if srv.mu.TryLock() {
		srv.mu.Unlock()
		return false
	}
	return true
}

func TestService_autoUpdateTick_SecretTypesReadWithoutLock(t *testing.T) {
	env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
	env.install(t, "mod-a", "v1.0.0", nil, auSecretModfile)
	env.offer("mod-a", "v1.0.1", nil, auSecretModfile)
	env.deploy("mod-a", func(d *pkg_models.Deployment) {
		d.Secrets = map[string]pkg_models.DeploymentSecret{"cert": {Id: "s-1"}}
	})
	env.deps.secretTypes = map[string]string{"s-1": "certificate"}
	var underLock int
	env.deps.onSecretTypes = func() {
		if auHeldLock(env.srv) {
			underLock++
		}
	}
	if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateRan {
		t.Fatalf("expected run, got %v", outcome)
	}
	if underLock != 0 || env.deps.secretReads == 0 {
		t.Errorf("secret types read %d times, %d of them under the service lock", env.deps.secretReads, underLock)
	}
	if got := env.deps.updatedIds(); !reflect.DeepEqual(got, []string{"mod-a"}) {
		t.Errorf("unexpected deployment updates %v", got)
	}
	auAssertIdle(t, env.srv)
}

// A secret stored after the secret types were read has an unknown type.
func TestService_planAutoUpdate_SecretStoredAfterSecretTypesRead(t *testing.T) {
	env := newAutoUpdateTestEnv(t, nil)
	env.install(t, "mod-a", "v1.0.0", nil, auSecretModfile)
	env.offer("mod-a", "v1.0.1", nil, auSecretModfile)
	env.deploy("mod-a", nil)
	env.deps.secretTypes = map[string]string{"s-1": "certificate"}
	// the first read is taken for the secret types, the second one plans
	env.deps.onGet = func(calls int) {
		if calls == 2 {
			d := env.deps.deployments["mod-a"]
			d.Secrets = map[string]pkg_models.DeploymentSecret{"cert": {Id: "s-1"}}
			env.deps.deployments["mod-a"] = d
		}
	}
	plan := auPlan(t, env, "patch")
	want := []lib_models.CarryOverMissing{{Kind: lib_constants.CarryOverKindSecret, Ref: "cert", Reason: lib_constants.CarryOverReasonSecretTypeUnknown}}
	if plan.changeRequest != nil || len(plan.skipped) != 1 || !reflect.DeepEqual(plan.skipped[0].Missing, want) || env.deps.secretReads != 0 {
		t.Errorf("unexpected plan changed=%v skipped=%+v, %d reads of the secret types", auChanged(plan), plan.skipped, env.deps.secretReads)
	}
}

func TestService_autoUpdateTick_DeploymentJobResultStored(t *testing.T) {
	setup := func(t *testing.T) (*auTestEnv, *string) {
		env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
		env.install(t, "mod-a", "v1.0.0", nil, "")
		env.offer("mod-a", "v1.0.1", nil, "")
		env.deploy("mod-a", nil)
		var jobId string
		env.db.onRunDate = func() error {
			if job, ok := env.srv.jobsHandler.CurrentSlotJob(deploymentJobSlotNum); ok {
				jobId = job.Id
			}
			return nil
		}
		return env, &jobId
	}
	result := func(t *testing.T, env *auTestEnv, jobId *string) lib_models.DeploymentUpdateJobResult {
		t.Helper()
		if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateRan {
			t.Fatalf("expected run, got %v", outcome)
		}
		auAssertIdle(t, env.srv)
		if *jobId == "" {
			t.Fatal("no deployment job reserved")
		}
		res, err := env.srv.GetUpdateDeploymentsJobResult(context.Background(), *jobId)
		if err != nil {
			t.Fatalf("no result for the deployment job: %v", err)
		}
		return res
	}
	t.Run("cancelled before the deployment phase", func(t *testing.T) {
		env, jobId := setup(t)
		env.mods.afterUpdate = func(_ *pkg_models.Module) {
			if job, ok := env.srv.jobsHandler.CurrentSlotJob(deploymentJobSlotNum); ok {
				job.Cancel()
			}
		}
		if res := result(t, env, jobId); !res.HasError || res.ErrorMsg != autoUpdateDeploymentCancelled || len(res.Results) != 0 {
			t.Errorf("unexpected result %+v", res)
		}
	})
	t.Run("unused", func(t *testing.T) {
		env, jobId := setup(t)
		env.mods.updateErr = map[string]error{"mod-a": errors.New("disk full")}
		if res := result(t, env, jobId); res.HasError || len(res.Results) != 0 || res.JobId != *jobId {
			t.Errorf("unexpected result %+v", res)
		}
	})
}
