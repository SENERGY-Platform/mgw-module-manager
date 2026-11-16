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
	"io/fs"
	"reflect"
	"sync"
	"testing"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	handler_jobs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

// coMod returns a module wrapper with the given id depending on deps.
func coMod(id string, deps ...string) modWrapper {
	mod := external_models.ModuleLibModule{ID: id, Version: "v1.0.0"}
	if len(deps) > 0 {
		mod.Dependencies = make(map[string]string)
		for _, dep := range deps {
			mod.Dependencies[dep] = ">=v1.0.0"
		}
	}
	return modWrapper{Mod: mod, Source: auSource, Channel: auChannel}
}

func coChange(mod modWrapper) changeItem {
	return changeItem{Previous: lib_models.ModuleAbbreviated{Id: mod.Mod.ID}, Next: mod}
}

func coOpIds(ops []changeOp) []string {
	var ids []string
	for _, op := range ops {
		ids = append(ids, op.action+":"+op.mod.Mod.ID)
	}
	return ids
}

func TestOrderChangeOps(t *testing.T) {
	tests := []struct {
		name    string
		install []modWrapper
		change  []modWrapper
		want    []string
	}{
		{
			name:   "independent modules in id order",
			change: []modWrapper{coMod("c"), coMod("a"), coMod("b")},
			want:   []string{"change:a", "change:b", "change:c"},
		},
		{
			name:   "chain",
			change: []modWrapper{coMod("a", "b"), coMod("b", "c"), coMod("c")},
			want:   []string{"change:c", "change:b", "change:a"},
		},
		{
			name:   "diamond",
			change: []modWrapper{coMod("a", "c", "b"), coMod("c", "z"), coMod("b", "z"), coMod("z")},
			want:   []string{"change:z", "change:b", "change:c", "change:a"},
		},
		{
			name: "cycle between its dependency and its dependent",
			// m and n depend on each other, n depends on z, a depends on m
			change: []modWrapper{coMod("a", "m"), coMod("m", "n"), coMod("n", "m", "z"), coMod("z")},
			want:   []string{"change:z", "change:m", "change:n", "change:a"},
		},
		{
			name:   "cycle of three",
			change: []modWrapper{coMod("c", "a"), coMod("b", "c"), coMod("a", "b")},
			want:   []string{"change:a", "change:b", "change:c"},
		},
		{
			name:    "install depends on change and change on install",
			install: []modWrapper{coMod("a", "b"), coMod("y")},
			change:  []modWrapper{coMod("b", "y"), coMod("x")},
			want:    []string{"change:x", "install:y", "change:b", "install:a"},
		},
		{
			name:   "dependencies outside the request and on itself are ignored",
			change: []modWrapper{coMod("a", "a", "outside"), coMod("b", "outside")},
			want:   []string{"change:a", "change:b"},
		},
		{
			name: "empty",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cr := &modulesChangeRequest{Install: tc.install}
			for _, mod := range tc.change {
				cr.Change = append(cr.Change, coChange(mod))
			}
			if got := coOpIds(orderChangeOps(cr)); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

type coOrderModules struct {
	modulesHandler
	mu    sync.Mutex
	calls []string
	fail  map[string]bool
}

func (m *coOrderModules) record(call, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, call+":"+id)
	if m.fail[id] {
		return errors.New(call + " " + id + " failed")
	}
	return nil
}

func (m *coOrderModules) AddModule(_ context.Context, id, _, _ string, _ fs.FS) error {
	return m.record(lib_constants.ActionInstall, id)
}

func (m *coOrderModules) UpdateModule(_ context.Context, id, _, _ string, _ fs.FS) error {
	return m.record(lib_constants.ActionChange, id)
}

func (m *coOrderModules) DeleteModule(_ context.Context, id string) error {
	return m.record(lib_constants.ActionRemove, id)
}

type coOrderDeployments struct {
	deploymentsHandler
}

func (d *coOrderDeployments) IsDeployed(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func newChangeOrderTestService(mods *coOrderModules) *Service {
	return New(nil, mods, &coOrderDeployments{}, nil, nil, nil, nil, handler_jobs.New(context.Background(), handler_jobs.Config{}), nil)
}

func TestService_execModulesChangeRequest_DependencyOrder(t *testing.T) {
	mods := &coOrderModules{}
	srv := newChangeOrderTestService(mods)
	cr := &modulesChangeRequest{
		Install: []modWrapper{coMod("a", "b")},
		Remove:  []string{"r"},
	}
	for _, mod := range []modWrapper{coMod("b", "c"), coMod("c")} {
		cr.Change = append(cr.Change, coChange(mod))
	}
	report := srv.execModulesChangeRequest(context.Background(), cr)
	want := []string{"remove:r", "change:c", "change:b", "install:a"}
	if !reflect.DeepEqual(mods.calls, want) {
		t.Errorf("applied %v, want %v", mods.calls, want)
	}
	if len(report.Failed) != 0 || len(report.Success) != 4 {
		t.Errorf("unexpected report %+v", report)
	}
}

func TestService_execModulesChangeRequest_FailedDependencyBlocksDependents(t *testing.T) {
	mods := &coOrderModules{fail: map[string]bool{"l": true}}
	srv := newChangeOrderTestService(mods)
	cr := &modulesChangeRequest{Install: []modWrapper{coMod("n", "x")}}
	// a needs l through b, d needs l directly and x, x is independent
	for _, mod := range []modWrapper{coMod("a", "b"), coMod("b", "l"), coMod("d", "l", "x"), coMod("l"), coMod("x")} {
		cr.Change = append(cr.Change, coChange(mod))
	}
	report := srv.execModulesChangeRequest(context.Background(), cr)
	if want := []string{"change:l", "change:x", "install:n"}; !reflect.DeepEqual(mods.calls, want) {
		t.Errorf("applied %v, want %v", mods.calls, want)
	}
	wantSuccess := []lib_models.ChangeReportItem{{Id: "x", Action: lib_constants.ActionChange}, {Id: "n", Action: lib_constants.ActionInstall}}
	if !reflect.DeepEqual(report.Success, wantSuccess) {
		t.Errorf("success %+v, want %+v", report.Success, wantSuccess)
	}
	wantFailed := []lib_models.ChangeReportErrItem{
		{ChangeReportItem: lib_models.ChangeReportItem{Id: "l", Action: lib_constants.ActionChange}, Error: "change l failed"},
		{ChangeReportItem: lib_models.ChangeReportItem{Id: "b", Action: lib_constants.ActionChange}, Error: "not applied, dependency 'l' failed"},
		{ChangeReportItem: lib_models.ChangeReportItem{Id: "a", Action: lib_constants.ActionChange}, Error: "not applied, dependency 'b' not applied because 'l' failed"},
		{ChangeReportItem: lib_models.ChangeReportItem{Id: "d", Action: lib_constants.ActionChange}, Error: "not applied, dependency 'l' failed"},
	}
	if !reflect.DeepEqual(report.Failed, wantFailed) {
		t.Errorf("failed %+v, want %+v", report.Failed, wantFailed)
	}
}

func TestService_execModulesChangeRequest_FailedDependencyInCycle(t *testing.T) {
	mods := &coOrderModules{fail: map[string]bool{"m": true}}
	srv := newChangeOrderTestService(mods)
	cr := &modulesChangeRequest{}
	for _, mod := range []modWrapper{coMod("m", "n"), coMod("n", "m")} {
		cr.Change = append(cr.Change, coChange(mod))
	}
	report := srv.execModulesChangeRequest(context.Background(), cr)
	if want := []string{"change:m"}; !reflect.DeepEqual(mods.calls, want) {
		t.Errorf("applied %v, want %v", mods.calls, want)
	}
	if len(report.Failed) != 2 || report.Failed[1].Id != "n" || report.Failed[1].Error != "not applied, dependency 'm' failed" {
		t.Errorf("unexpected report %+v", report)
	}
}

func TestService_execModulesChangeRequest_FailureInCycleBlocksDependentsOfAppliedMember(t *testing.T) {
	// p and q depend on each other, p is applied before q fails, r depends on p
	mods := &coOrderModules{fail: map[string]bool{"q": true}}
	srv := newChangeOrderTestService(mods)
	cr := &modulesChangeRequest{}
	for _, mod := range []modWrapper{coMod("p", "q"), coMod("q", "p"), coMod("r", "p")} {
		cr.Change = append(cr.Change, coChange(mod))
	}
	report := srv.execModulesChangeRequest(context.Background(), cr)
	if want := []string{"change:p", "change:q"}; !reflect.DeepEqual(mods.calls, want) {
		t.Errorf("applied %v, want %v", mods.calls, want)
	}
	if want := []lib_models.ChangeReportItem{{Id: "p", Action: lib_constants.ActionChange}}; !reflect.DeepEqual(report.Success, want) {
		t.Errorf("success %+v, want %+v", report.Success, want)
	}
	wantFailed := []lib_models.ChangeReportErrItem{
		{ChangeReportItem: lib_models.ChangeReportItem{Id: "q", Action: lib_constants.ActionChange}, Error: "change q failed"},
		{ChangeReportItem: lib_models.ChangeReportItem{Id: "r", Action: lib_constants.ActionChange}, Error: "not applied, dependency 'p' depends on failed 'q'"},
	}
	if !reflect.DeepEqual(report.Failed, wantFailed) {
		t.Errorf("failed %+v, want %+v", report.Failed, wantFailed)
	}
}
