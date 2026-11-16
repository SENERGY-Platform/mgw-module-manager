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
	"fmt"
	"io/fs"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	handler_jobs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
	helper_modfile "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/modfile"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

const (
	auSource  = "src"
	auChannel = "main"
)

// auModfile returns a module file system with a single service; extra is appended to the modfile verbatim. A dependency
// is given as "id" for the range >=v0.0.1 or as "id@range".
func auModfile(id, version string, deps []string, extra string) fstest.MapFS {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "modfileVersion: \"v1\"\nid: %s\nname: Test\nversion: %s\ntype: add-on\ndeploymentType: single\n", id, version)
	b.WriteString("services:\n  srv:\n    name: Srv\n    image: ghcr.io/org/repo:test\n")
	if len(deps) > 0 {
		b.WriteString("dependencies:\n")
		for _, dep := range deps {
			depId, versionRange, ok := strings.Cut(dep, "@")
			if !ok {
				versionRange = ">=v0.0.1"
			}
			_, _ = fmt.Fprintf(&b, "  %s:\n    version: \"%s\"\n", depId, versionRange)
		}
	}
	b.WriteString(extra)
	return fstest.MapFS{"Modfile.yml": {Data: []byte(b.String())}}
}

// auConfigModfile declares a text config with a default that is a user input.
const auConfigModfile = `configs:
  cfg:
    value: "x"
    userInput:
      name: Cfg
      type: text
    targets:
      - refVar: CFG
        services:
          - srv
`

type auTestDB struct {
	mu            sync.Mutex
	row           *pkg_models.AutoUpdate
	runDateWrites int
	onRunDate     func() error
}

func (d *auTestDB) Ping(_ context.Context) error { return nil }

func (d *auTestDB) ReadAutoUpdate(_ context.Context) (pkg_models.AutoUpdate, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.row == nil {
		return pkg_models.AutoUpdate{}, lib_errors.New[lib_errors.ErrNotFound]("not found")
	}
	return *d.row, nil
}

func (d *auTestDB) WriteAutoUpdateSettings(_ context.Context, settings lib_models.AutoUpdateSettings) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.row == nil {
		d.row = &pkg_models.AutoUpdate{}
	}
	d.row.AutoUpdateSettings = settings
	return nil
}

func (d *auTestDB) WriteAutoUpdateRunDate(_ context.Context, date string) error {
	if d.onRunDate != nil {
		if err := d.onRunDate(); err != nil {
			return err
		}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.row.LastRunDate = date
	d.runDateWrites++
	return nil
}

func (d *auTestDB) WriteAutoUpdateLastRun(ctx context.Context, run lib_models.AutoUpdateRun) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.row.LastRun = &run
	return nil
}

func (d *auTestDB) get() pkg_models.AutoUpdate {
	d.mu.Lock()
	defer d.mu.Unlock()
	return *d.row
}

type auTestRepos struct {
	repositoriesHandler
	mu        sync.Mutex
	modules   map[string]fstest.MapFS // id|source|channel
	unlisted  map[string]bool         // module IDs left out of GetModules but still served by GetModuleFS
	refreshes int
	onRefresh func()
}

func (r *auTestRepos) RefreshRepositories(_ context.Context, _ lib_models.RepositoriesRefreshFilter) ([]lib_models.RepositoryResult, error) {
	r.mu.Lock()
	r.refreshes++
	onRefresh := r.onRefresh
	r.mu.Unlock()
	if onRefresh != nil {
		onRefresh()
	}
	return nil, nil
}

func (r *auTestRepos) refreshCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.refreshes
}

func (r *auTestRepos) GetRepositories(_ context.Context) ([]lib_models.Repository, error) {
	return []lib_models.Repository{{
		Source:   auSource,
		Priority: 1,
		Channels: []lib_models.RepositoryChannel{{Name: auChannel, Priority: 1}},
	}}, nil
}

func (r *auTestRepos) GetModules(_ context.Context, filter pkg_models.RepositoryModulesFilter) ([]pkg_models.RepositoryModule, error) {
	var result []pkg_models.RepositoryModule
	for key, fSys := range r.modules {
		parts := strings.Split(key, "|")
		if (len(filter.Ids) > 0 && !slices.Contains(filter.Ids, parts[0])) || r.unlisted[parts[0]] {
			continue
		}
		mod, err := helper_modfile.GetModule(fSys)
		if err != nil {
			return nil, err
		}
		result = append(result, pkg_models.RepositoryModule{
			RepositoryModuleBase: pkg_models.RepositoryModuleBase{Id: parts[0], Source: parts[1], Channel: parts[2]},
			Version:              mod.Version,
		})
	}
	return result, nil
}

func (r *auTestRepos) GetModuleFS(_ context.Context, id, source, channel string) (fs.FS, error) {
	fSys, ok := r.modules[id+"|"+source+"|"+channel]
	if !ok {
		return nil, lib_errors.New[lib_errors.ErrNotFound]("module not found")
	}
	return fSys, nil
}

type auTestModules struct {
	modulesHandler
	mu          sync.Mutex
	installed   map[string]pkg_models.Module
	updates     []string
	afterUpdate func(m *pkg_models.Module)
	getErr      error
	updateErr   map[string]error
}

func (m *auTestModules) GetModules(ctx context.Context, filter pkg_models.ModulesFilterWithName, _ bool) (map[string]pkg_models.Module, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	result := make(map[string]pkg_models.Module)
	for id, mod := range m.installed {
		if len(filter.Ids) > 0 && !slices.Contains(filter.Ids, id) {
			continue
		}
		result[id] = mod
	}
	return result, nil
}

func (m *auTestModules) UpdateModule(_ context.Context, id, source, channel string, fSys fs.FS) error {
	mod, err := helper_modfile.GetModule(fSys)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err = m.updateErr[id]; err != nil {
		return err
	}
	installed := pkg_models.Module{ModuleLibModule: mod, Source: source, Channel: channel}
	if m.afterUpdate != nil {
		m.afterUpdate(&installed)
	}
	m.installed[id] = installed
	m.updates = append(m.updates, id+"@"+mod.Version)
	return nil
}

func (m *auTestModules) AddModule(_ context.Context, id, _, _ string, _ fs.FS) error {
	return errors.New("unexpected install of " + id)
}

func (m *auTestModules) updateList() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.updates)
}

type auTestDeployments struct {
	deploymentsHandler
	mu             sync.Mutex
	deployments    map[string]pkg_models.Deployment // by module ID
	updated        map[string]pkg_models.DeploymentUserInput
	onUpdate       func()
	onGet          func(calls int)
	gets           int
	secretTypes    map[string]string
	secretTypesErr error
	secretReads    int
	onSecretTypes  func()
}

func (d *auTestDeployments) GetSecretTypes(_ context.Context) (map[string]string, error) {
	if d.onSecretTypes != nil {
		d.onSecretTypes()
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.secretReads++
	return d.secretTypes, d.secretTypesErr
}

func (d *auTestDeployments) GetDeploymentsByModuleIds(ctx context.Context, filter pkg_models.DeploymentsFilterWithState) (map[string]pkg_models.Deployment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.gets++
	if d.onGet != nil {
		d.onGet(d.gets)
	}
	result := make(map[string]pkg_models.Deployment)
	for id, dep := range d.deployments {
		if len(filter.ModuleIds) > 0 && !slices.Contains(filter.ModuleIds, id) {
			continue
		}
		result[id] = dep
	}
	return result, nil
}

func (d *auTestDeployments) GetReducedDeploymentsByModuleIds(ctx context.Context, filter pkg_models.DeploymentsFilterWithState) (map[string]pkg_models.DeploymentReduced, error) {
	deployments, err := d.GetDeploymentsByModuleIds(ctx, filter)
	if err != nil {
		return nil, err
	}
	result := make(map[string]pkg_models.DeploymentReduced)
	for id, dep := range deployments {
		result[id] = pkg_models.DeploymentReduced{DeploymentBase: dep.DeploymentBase}
	}
	return result, nil
}

func (d *auTestDeployments) UpdateDeployments(
	_ context.Context,
	selectedModules map[string]pkg_models.Module,
	userInputs map[string]pkg_models.DeploymentUserInput,
) ([]lib_models.DeploymentResult, error) {
	if d.onUpdate != nil {
		d.onUpdate()
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.updated == nil {
		d.updated = make(map[string]pkg_models.DeploymentUserInput)
	}
	var results []lib_models.DeploymentResult
	for id, mod := range selectedModules {
		d.updated[id] = userInputs[id]
		results = append(results, lib_models.DeploymentResult{ModuleId: id, Id: d.deployments[id].Id})
		// Deployments created without a module variant keep it empty, so they are never seen as lagging.
		if dep := d.deployments[id]; dep.ModuleVersion != "" {
			dep.ModuleSource, dep.ModuleChannel, dep.ModuleVersion = mod.Source, mod.Channel, mod.Version
			d.deployments[id] = dep
		}
	}
	return results, nil
}

func (d *auTestDeployments) GetDeployment(_ context.Context, id string) (pkg_models.Deployment, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, dep := range d.deployments {
		if dep.Id == id {
			return dep, nil
		}
	}
	return pkg_models.Deployment{}, lib_errors.New[lib_errors.ErrNotFound]("deployment not found")
}

func (d *auTestDeployments) updatedIds() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	var ids []string
	for id := range d.updated {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

type auTestAux struct {
	auxiliaryDeploymentsHandler
}

func (a *auTestAux) RecreateDeployments(
	_ context.Context,
	_ pkg_models.Module,
	_ pkg_models.Deployment,
	_ map[string]pkg_models.DeploymentReduced,
	_ lib_models.AuxiliaryDeploymentsFilterWithState,
) ([]lib_models.AuxiliaryDeploymentBatchResult, error) {
	return nil, nil
}

type auTestEnv struct {
	srv   *Service
	db    *auTestDB
	repos *auTestRepos
	mods  *auTestModules
	deps  *auTestDeployments
	gcs   *carryOverTestGlobalConfigsHandler
	mu    sync.Mutex
	now   time.Time
}

func (e *auTestEnv) setNow(t time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.now = t
}

func (e *auTestEnv) getNow() time.Time {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.now
}

// install adds an installed module parsed from the same modfile a repository would provide.
func (e *auTestEnv) install(t *testing.T, id, version string, deps []string, extra string) {
	t.Helper()
	mod, err := helper_modfile.GetModule(auModfile(id, version, deps, extra))
	if err != nil {
		t.Fatal(err)
	}
	e.mods.installed[id] = pkg_models.Module{ModuleLibModule: mod, Source: auSource, Channel: auChannel}
}

func (e *auTestEnv) offer(id, version string, deps []string, extra string) {
	e.repos.modules[id+"|"+auSource+"|"+auChannel] = auModfile(id, version, deps, extra)
}

// deploy deploys an installed module as created for its installed variant.
func (e *auTestEnv) deploy(id string, mutate func(d *pkg_models.Deployment)) {
	d := pkg_models.Deployment{DeploymentBase: pkg_models.DeploymentBase{Id: "dep-" + id, ModuleId: id}}
	if mod, ok := e.mods.installed[id]; ok {
		d.ModuleSource, d.ModuleChannel, d.ModuleVersion = mod.Source, mod.Channel, mod.Version
	}
	if mutate != nil {
		mutate(&d)
	}
	e.deps.deployments[id] = d
}

// deployAt deploys an installed module as created for version of its source and channel.
func (e *auTestEnv) deployAt(id, version string, mutate func(d *pkg_models.Deployment)) {
	e.deploy(id, func(d *pkg_models.Deployment) {
		d.ModuleSource, d.ModuleChannel, d.ModuleVersion = auSource, auChannel, version
		if mutate != nil {
			mutate(d)
		}
	})
}

func newAutoUpdateTestEnv(t *testing.T, settings *lib_models.AutoUpdateSettings) *auTestEnv {
	t.Helper()
	env := &auTestEnv{
		db:    &auTestDB{},
		repos: &auTestRepos{modules: make(map[string]fstest.MapFS)},
		mods:  &auTestModules{installed: make(map[string]pkg_models.Module)},
		deps:  &auTestDeployments{deployments: make(map[string]pkg_models.Deployment)},
		gcs:   &carryOverTestGlobalConfigsHandler{configs: make(map[string]pkg_models.Config)},
		now:   time.Date(2026, 11, 16, 3, 0, 0, 0, time.UTC),
	}
	if settings != nil {
		env.db.row = &pkg_models.AutoUpdate{AutoUpdateSettings: *settings}
	}
	ctx, cf := context.WithCancel(context.Background())
	t.Cleanup(cf)
	env.srv = New(env.repos, env.mods, env.deps, &auTestAux{}, env.gcs, nil, env.db, handler_jobs.New(ctx, handler_jobs.Config{}), nil)
	env.srv.now = env.getNow
	return env
}

func auEnabled(level, start, end string) *lib_models.AutoUpdateSettings {
	return &lib_models.AutoUpdateSettings{Enabled: true, Level: level, WindowStart: start, WindowEnd: end, TimeZone: "UTC"}
}

func TestAutoUpdateLevel(t *testing.T) {
	cases := []struct {
		version, next string
		level         string
		newer         bool
		err           bool
	}{
		{"v1.0.0", "v1.0.1", lib_constants.AutoUpdateLevelPatch, true, false},
		{"v1.0.9", "v1.1.0", lib_constants.AutoUpdateLevelMinor, true, false},
		{"v1.9.9", "v2.0.0", lib_constants.AutoUpdateLevelMajor, true, false},
		{"v0.4.3", "v0.4.4", lib_constants.AutoUpdateLevelPatch, true, false},
		{"v0.4.3", "v0.5.0", lib_constants.AutoUpdateLevelMinor, true, false},
		{"v0.9.0", "v1.0.0", lib_constants.AutoUpdateLevelMajor, true, false},
		{"v1.2.3-rc.1", "v1.2.3", lib_constants.AutoUpdateLevelPatch, true, false},
		{"v1.2.3", "v1.2.4-rc.1", lib_constants.AutoUpdateLevelPatch, true, false},
		{"v1.2.3", "v1.3.0-beta", lib_constants.AutoUpdateLevelMinor, true, false},
		{"v1.2.3", "v1.2.3-rc.1", "", false, false},
		{"v1.0.1", "v1.0.0", "", false, false},
		{"v1.0.0", "v1.0.0", "", false, false},
		{"v1.0.0+a", "v1.0.0+b", "", false, false},
		{"v1", "v1.0.1", lib_constants.AutoUpdateLevelPatch, true, false},
		{"1.0.0", "v1.0.1", "", false, true},
		{"v1.0.0", "latest", "", false, true},
	}
	for _, c := range cases {
		t.Run(c.version+"->"+c.next, func(t *testing.T) {
			level, newer, err := autoUpdateLevel(c.version, c.next)
			if (err != nil) != c.err || newer != c.newer || level != c.level {
				t.Errorf("got level=%q newer=%v err=%v, want level=%q newer=%v err=%v", level, newer, err, c.level, c.newer, c.err)
			}
		})
	}
}

func TestExceedsAutoUpdateLevel(t *testing.T) {
	cases := []struct {
		level, maxLevel string
		exceeds         bool
	}{
		{lib_constants.AutoUpdateLevelPatch, lib_constants.AutoUpdateLevelPatch, false},
		{lib_constants.AutoUpdateLevelMinor, lib_constants.AutoUpdateLevelPatch, true},
		{lib_constants.AutoUpdateLevelMinor, lib_constants.AutoUpdateLevelMinor, false},
		{lib_constants.AutoUpdateLevelMajor, lib_constants.AutoUpdateLevelMinor, true},
		{lib_constants.AutoUpdateLevelMajor, lib_constants.AutoUpdateLevelMajor, false},
		{lib_constants.AutoUpdateLevelPatch, "", true},
	}
	for _, c := range cases {
		if got := exceedsAutoUpdateLevel(c.level, c.maxLevel); got != c.exceeds {
			t.Errorf("exceedsAutoUpdateLevel(%q, %q) = %v, want %v", c.level, c.maxLevel, got, c.exceeds)
		}
	}
}

func TestValidateAutoUpdateSettings(t *testing.T) {
	valid := []lib_models.AutoUpdateSettings{
		defaultAutoUpdateSettings,
		{Level: lib_constants.AutoUpdateLevelMinor, WindowStart: "00:00", WindowEnd: "23:59", TimeZone: "UTC"},
		{Level: lib_constants.AutoUpdateLevelMajor, WindowStart: "23:00", WindowEnd: "01:00", TimeZone: "America/New_York"},
	}
	for _, settings := range valid {
		if err := validateAutoUpdateSettings(settings); err != nil {
			t.Errorf("%+v: unexpected error %v", settings, err)
		}
	}
	invalid := []lib_models.AutoUpdateSettings{
		{Level: "", WindowStart: "02:00", WindowEnd: "05:00", TimeZone: "UTC"},
		{Level: "PATCH", WindowStart: "02:00", WindowEnd: "05:00", TimeZone: "UTC"},
		{Level: "patch", WindowStart: "2:00", WindowEnd: "05:00", TimeZone: "UTC"},
		{Level: "patch", WindowStart: "24:00", WindowEnd: "05:00", TimeZone: "UTC"},
		{Level: "patch", WindowStart: "02:60", WindowEnd: "05:00", TimeZone: "UTC"},
		{Level: "patch", WindowStart: "+2:00", WindowEnd: "05:00", TimeZone: "UTC"},
		{Level: "patch", WindowStart: "02-00", WindowEnd: "05:00", TimeZone: "UTC"},
		{Level: "patch", WindowStart: "0::00", WindowEnd: "05:00", TimeZone: "UTC"},
		{Level: "patch", WindowStart: "02:00", WindowEnd: "05:0;", TimeZone: "UTC"},
		{Level: "patch", WindowStart: "02:00:00", WindowEnd: "05:00", TimeZone: "UTC"},
		{Level: "patch", WindowStart: "02:00", WindowEnd: "", TimeZone: "UTC"},
		{Level: "patch", WindowStart: "02:00", WindowEnd: "02:00", TimeZone: "UTC"},
		{Level: "patch", WindowStart: "02:00", WindowEnd: "05:00", TimeZone: ""},
		{Level: "patch", WindowStart: "02:00", WindowEnd: "05:00", TimeZone: "Local"},
		{Level: "patch", WindowStart: "02:00", WindowEnd: "05:00", TimeZone: "Europe/Atlantis"},
		{Level: "patch", WindowStart: "02:00", WindowEnd: "05:00", TimeZone: "../zoneinfo/UTC"},
		{Level: "patch", WindowStart: "02:00", WindowEnd: "05:00", TimeZone: "+01:00"},
		{Level: "patch", WindowStart: "02:00", WindowEnd: "05:00", TimeZone: strings.Repeat("A", 65)},
	}
	for _, settings := range invalid {
		if err := validateAutoUpdateSettings(settings); err == nil {
			t.Errorf("%+v: expected error", settings)
		}
	}
}

func TestService_AutoUpdateSettings(t *testing.T) {
	env := newAutoUpdateTestEnv(t, nil)
	ctx := context.Background()
	settings, err := env.srv.GetAutoUpdateSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := lib_models.AutoUpdateSettings{Enabled: false, Level: "patch", WindowStart: "02:00", WindowEnd: "05:00", TimeZone: "Europe/Berlin"}
	if settings != want {
		t.Errorf("defaults: got %+v, want %+v", settings, want)
	}
	err = env.srv.UpdateAutoUpdateSettings(ctx, lib_models.AutoUpdateSettings{Enabled: true, Level: "all", WindowStart: "02:00", WindowEnd: "05:00"})
	if !lib_errors.IsOf[lib_errors.ErrInvalidInput](err) {
		t.Errorf("expected invalid input error, got %v", err)
	}
	err = env.srv.UpdateAutoUpdateSettings(ctx, lib_models.AutoUpdateSettings{Enabled: true, Level: "patch", WindowStart: "02:00", WindowEnd: "05:00", TimeZone: "Mars/Olympus"})
	if !lib_errors.IsOf[lib_errors.ErrInvalidInput](err) {
		t.Errorf("unknown time zone: expected invalid input error, got %v", err)
	}
	if env.db.row != nil {
		t.Error("invalid settings must not be stored")
	}
	next := *auEnabled("minor", "23:30", "01:15")
	if err = env.srv.UpdateAutoUpdateSettings(ctx, next); err != nil {
		t.Fatal(err)
	}
	status, err := env.srv.GetAutoUpdateStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.AutoUpdateSettings != next || status.LastRun != nil {
		t.Errorf("unexpected status %+v", status)
	}
}

func TestAutoUpdateWindow_Date(t *testing.T) {
	at := func(y int, mo time.Month, d, h, m int) time.Time { return time.Date(y, mo, d, h, m, 0, 0, time.UTC) }
	cases := []struct {
		name       string
		start, end string
		tz         string
		t          time.Time
		date       string
		inside     bool
	}{
		{"before", "02:00", "05:00", "UTC", at(2026, 11, 16, 1, 59), "", false},
		{"start inclusive", "02:00", "05:00", "UTC", at(2026, 11, 16, 2, 0), "2026-11-16", true},
		{"last minute", "02:00", "05:00", "UTC", time.Date(2026, 11, 16, 4, 59, 59, 999, time.UTC), "2026-11-16", true},
		{"end exclusive", "02:00", "05:00", "UTC", at(2026, 11, 16, 5, 0), "", false},
		{"crossing before start", "23:00", "01:00", "UTC", at(2026, 11, 16, 22, 59), "", false},
		{"crossing evening", "23:00", "01:00", "UTC", at(2026, 11, 16, 23, 0), "2026-11-16", true},
		{"crossing after midnight keeps start date", "23:00", "01:00", "UTC", at(2026, 11, 17, 0, 30), "2026-11-16", true},
		{"crossing end exclusive", "23:00", "01:00", "UTC", at(2026, 11, 17, 1, 0), "", false},
		{"crossing month boundary", "23:00", "01:00", "UTC", at(2026, 3, 1, 0, 30), "2026-02-28", true},
		{"crossing year boundary", "23:00", "01:00", "UTC", at(2027, 1, 1, 0, 10), "2026-12-31", true},
		{"converted to the configured zone", "23:00", "01:00", "UTC", time.Date(2026, 11, 17, 0, 30, 0, 0, time.FixedZone("X", 3600)), "2026-11-16", true},
		// 2026-11-16 01:30 UTC is 02:30 in Berlin
		{"wall clock of the configured zone", "02:00", "05:00", "Europe/Berlin", at(2026, 11, 16, 1, 30), "2026-11-16", true},
		{"configured zone before start", "02:00", "05:00", "Europe/Berlin", at(2026, 11, 16, 0, 59), "", false},
		{"configured zone ahead of UTC across midnight", "23:00", "01:00", "Europe/Berlin", at(2026, 11, 16, 23, 30), "2026-11-16", true},
		{"configured zone behind UTC on the previous date", "02:00", "05:00", "America/New_York", at(2026, 11, 17, 8, 0), "2026-11-17", true},
		// Berlin 2026-03-29: 02:00 CET is followed by 03:00 CEST
		{"spring forward, last minute before the gap", "02:00", "04:00", "Europe/Berlin", at(2026, 3, 29, 0, 59), "", false},
		{"spring forward, first minute after the gap", "02:00", "04:00", "Europe/Berlin", at(2026, 3, 29, 1, 0), "2026-03-29", true},
		{"spring forward, window inside the gap", "02:00", "02:30", "Europe/Berlin", at(2026, 3, 29, 1, 0), "", false},
		// Berlin 2026-10-25: 03:00 CEST is followed by 02:00 CET, 02:xx occurs twice
		{"fall back, first 02:30", "02:00", "03:00", "Europe/Berlin", at(2026, 10, 25, 0, 30), "2026-10-25", true},
		{"fall back, second 02:30", "02:00", "03:00", "Europe/Berlin", at(2026, 10, 25, 1, 30), "2026-10-25", true},
		{"fall back, window crossing midnight", "23:00", "02:30", "Europe/Berlin", at(2026, 10, 25, 1, 15), "2026-10-24", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, err := newAutoUpdateWindow(lib_models.AutoUpdateSettings{WindowStart: c.start, WindowEnd: c.end, TimeZone: c.tz})
			if err != nil {
				t.Fatal(err)
			}
			date, inside := w.date(c.t)
			if date != c.date || inside != c.inside {
				t.Errorf("got %q %v, want %q %v", date, inside, c.date, c.inside)
			}
		})
	}
}

func auSkipped(plan autoUpdatePlan) map[string]string {
	result := make(map[string]string)
	for _, item := range plan.skipped {
		result[item.Id] = item.Reason
	}
	return result
}

func auChanged(plan autoUpdatePlan) []string {
	var result []string
	if plan.changeRequest == nil {
		return nil
	}
	for _, item := range plan.changeRequest.Change {
		result = append(result, item.Next.Mod.ID+"@"+item.Next.Mod.Version)
	}
	return result
}

func TestService_planAutoUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("patch update with deployment", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", nil, "")
		env.offer("mod-a", "v1.0.1", nil, "")
		env.deploy("mod-a", nil)
		plan, err := auPlanCtx(ctx, env, "patch")
		if err != nil {
			t.Fatal(err)
		}
		if _, deployed := plan.deployed["mod-a"]; !deployed || !reflect.DeepEqual(auChanged(plan), []string{"mod-a@v1.0.1"}) || len(plan.skipped) != 0 {
			t.Errorf("unexpected plan changed=%v skipped=%+v deployed=%v", auChanged(plan), plan.skipped, plan.deployed)
		}
	})

	t.Run("downgrade and equal version ignored", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.1", nil, "")
		env.offer("mod-a", "v1.0.0", nil, "")
		env.install(t, "mod-b", "v1.0.0", nil, "")
		env.offer("mod-b", "v1.0.0", nil, "")
		plan, err := auPlanCtx(ctx, env, "major")
		if err != nil {
			t.Fatal(err)
		}
		if plan.changeRequest != nil || len(plan.skipped) != 0 || len(plan.failed) != 0 {
			t.Errorf("expected nothing, got changed=%v skipped=%+v failed=%+v", auChanged(plan), plan.skipped, plan.failed)
		}
	})

	t.Run("other source or channel ignored", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", nil, "")
		env.repos.modules["mod-a|"+auSource+"|dev"] = auModfile("mod-a", "v1.0.1", nil, "")
		plan, err := auPlanCtx(ctx, env, "major")
		if err != nil {
			t.Fatal(err)
		}
		if plan.changeRequest != nil || len(plan.skipped) != 0 || len(plan.failed) != 0 {
			t.Errorf("expected nothing, got changed=%v skipped=%+v failed=%+v", auChanged(plan), plan.skipped, plan.failed)
		}
	})

	t.Run("non semver skipped", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "1.0.0", nil, "")
		env.offer("mod-a", "v1.0.1", nil, "")
		plan, err := auPlanCtx(ctx, env, "major")
		if err != nil {
			t.Fatal(err)
		}
		if plan.changeRequest != nil || auSkipped(plan)["mod-a"] != lib_constants.AutoUpdateReasonInvalidVersion {
			t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
		}
		if s := plan.skipped[0]; s.Version != "1.0.0" || s.NextVersion != "v1.0.1" {
			t.Errorf("unexpected versions %+v", s)
		}
	})

	t.Run("level exceeded skipped, v0 minor literal", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v0.4.2", nil, "")
		env.offer("mod-a", "v0.5.0", nil, "")
		plan, err := auPlanCtx(ctx, env, "patch")
		if err != nil {
			t.Fatal(err)
		}
		if plan.changeRequest != nil || auSkipped(plan)["mod-a"] != lib_constants.AutoUpdateReasonLevelExceeded {
			t.Errorf("patch: unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
		}
		plan, err = auPlanCtx(ctx, env, "minor")
		if err != nil {
			t.Fatal(err)
		}
		if got := auChanged(plan); !reflect.DeepEqual(got, []string{"mod-a@v0.5.0"}) {
			t.Errorf("minor: unexpected plan changed=%v skipped=%+v", got, plan.skipped)
		}
	})

	t.Run("dependency within level is updated together", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", []string{"mod-b"}, "")
		env.offer("mod-a", "v1.0.1", []string{"mod-b"}, "")
		env.install(t, "mod-b", "v2.0.0", nil, "")
		env.offer("mod-b", "v2.0.3", nil, "")
		plan, err := auPlanCtx(ctx, env, "patch")
		if err != nil {
			t.Fatal(err)
		}
		if got := auChanged(plan); !reflect.DeepEqual(got, []string{"mod-a@v1.0.1", "mod-b@v2.0.3"}) || len(plan.skipped) != 0 {
			t.Errorf("unexpected plan changed=%v skipped=%+v", got, plan.skipped)
		}
	})

	t.Run("dependency is part of the change set without being a candidate itself", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", []string{"mod-b"}, "")
		env.offer("mod-a", "v1.0.1", []string{"mod-b"}, "")
		env.install(t, "mod-b", "v2.0.0", nil, "")
		env.offer("mod-b", "v2.0.3", nil, "")
		env.repos.unlisted = map[string]bool{"mod-b": true}
		plan, err := auPlanCtx(ctx, env, "patch")
		if err != nil {
			t.Fatal(err)
		}
		if got := auChanged(plan); !reflect.DeepEqual(got, []string{"mod-a@v1.0.1", "mod-b@v2.0.3"}) || len(plan.skipped) != 0 {
			t.Errorf("unexpected plan changed=%v skipped=%+v", got, plan.skipped)
		}
	})

	t.Run("dependency over level skipped", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", []string{"mod-b"}, "")
		env.offer("mod-a", "v1.0.1", []string{"mod-b"}, "")
		env.install(t, "mod-b", "v2.0.0", nil, "")
		env.offer("mod-b", "v2.1.0", nil, "")
		plan, err := auPlanCtx(ctx, env, "patch")
		if err != nil {
			t.Fatal(err)
		}
		skipped := auSkipped(plan)
		if plan.changeRequest != nil ||
			skipped["mod-a"] != lib_constants.AutoUpdateReasonDependencyLevelExceeded ||
			skipped["mod-b"] != lib_constants.AutoUpdateReasonLevelExceeded {
			t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
		}
	})

	t.Run("dependency from another source skipped", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", []string{"mod-b"}, "")
		env.offer("mod-a", "v1.0.1", []string{"mod-b"}, "")
		env.install(t, "mod-b", "v2.0.0", nil, "")
		env.mods.installed["mod-b"] = func() pkg_models.Module {
			m := env.mods.installed["mod-b"]
			m.Source = "other"
			return m
		}()
		env.offer("mod-b", "v2.0.1", nil, "")
		plan, err := auPlanCtx(ctx, env, "major")
		if err != nil {
			t.Fatal(err)
		}
		if plan.changeRequest != nil || auSkipped(plan)["mod-a"] != lib_constants.AutoUpdateReasonDependencyLevelExceeded {
			t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
		}
	})

	t.Run("new dependency skipped", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", nil, "")
		env.offer("mod-a", "v1.0.1", []string{"mod-c"}, "")
		env.offer("mod-c", "v1.0.0", nil, "")
		plan, err := auPlanCtx(ctx, env, "patch")
		if err != nil {
			t.Fatal(err)
		}
		if plan.changeRequest != nil || auSkipped(plan)["mod-a"] != lib_constants.AutoUpdateReasonNewDependency {
			t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
		}
	})

	t.Run("needs input skipped with missing list", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", nil, auConfigModfile)
		env.offer("mod-a", "v1.0.1", nil, auConfigModfile)
		env.deploy("mod-a", func(d *pkg_models.Deployment) {
			d.Configs = map[string]pkg_models.DeploymentUserConfig{"cfg": {Value: coStringValue("")}}
		})
		plan, err := auPlanCtx(ctx, env, "patch")
		if err != nil {
			t.Fatal(err)
		}
		if plan.changeRequest != nil || len(plan.skipped) != 1 || plan.skipped[0].Reason != lib_constants.AutoUpdateReasonNeedsInput {
			t.Fatalf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
		}
		want := []lib_models.CarryOverMissing{{Kind: "config", Ref: "cfg", Reason: lib_constants.CarryOverReasonEmptyValue}}
		if !reflect.DeepEqual(plan.skipped[0].Missing, want) {
			t.Errorf("got missing %+v, want %+v", plan.skipped[0].Missing, want)
		}
	})

	t.Run("dependency deployment needs input skips the dependent module", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", []string{"mod-b"}, "")
		env.offer("mod-a", "v1.0.1", []string{"mod-b"}, "")
		env.install(t, "mod-b", "v2.0.0", nil, auConfigModfile)
		env.offer("mod-b", "v2.0.1", nil, auConfigModfile)
		env.deploy("mod-b", func(d *pkg_models.Deployment) {
			d.Configs = map[string]pkg_models.DeploymentUserConfig{"cfg": {Value: coStringValue("")}}
		})
		plan, err := auPlanCtx(ctx, env, "patch")
		if err != nil {
			t.Fatal(err)
		}
		skipped := auSkipped(plan)
		if plan.changeRequest != nil ||
			skipped["mod-a"] != lib_constants.AutoUpdateReasonNeedsInput ||
			skipped["mod-b"] != lib_constants.AutoUpdateReasonNeedsInput {
			t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
		}
	})

	t.Run("dependency module error skips the dependent module", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", []string{"mod-b"}, "")
		env.offer("mod-a", "v1.0.1", []string{"mod-b"}, "")
		env.install(t, "mod-b", "v2.0.0", nil, "")
		env.offer("mod-b", "v2.0.1", nil, "")
		m := env.mods.installed["mod-b"]
		m.Err = errors.New("broken")
		env.mods.installed["mod-b"] = m
		plan, err := auPlanCtx(ctx, env, "patch")
		if err != nil {
			t.Fatal(err)
		}
		skipped := auSkipped(plan)
		if plan.changeRequest != nil ||
			skipped["mod-a"] != lib_constants.AutoUpdateReasonModuleError ||
			skipped["mod-b"] != lib_constants.AutoUpdateReasonModuleError {
			t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
		}
	})

	t.Run("module and deployment errors skipped", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		env.install(t, "mod-a", "v1.0.0", nil, "")
		env.offer("mod-a", "v1.0.1", nil, "")
		// an unreadable module has no declaration to take the version from
		env.mods.installed["mod-a"] = pkg_models.Module{Source: auSource, Channel: auChannel, Err: errors.New("broken")}
		env.install(t, "mod-b", "v1.0.0", nil, "")
		env.offer("mod-b", "v1.0.1", nil, "")
		env.deploy("mod-b", func(d *pkg_models.Deployment) { d.Err = errors.New("broken") })
		plan, err := auPlanCtx(ctx, env, "patch")
		if err != nil {
			t.Fatal(err)
		}
		skipped := auSkipped(plan)
		if plan.changeRequest != nil ||
			skipped["mod-a"] != lib_constants.AutoUpdateReasonModuleError ||
			skipped["mod-b"] != lib_constants.AutoUpdateReasonDeploymentError {
			t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
		}
	})
}

func TestService_execModulesChangeRequest_ClearsOnlyItsRequest(t *testing.T) {
	env := newAutoUpdateTestEnv(t, nil)
	own := &modulesChangeRequest{}
	other := &modulesChangeRequest{}
	env.srv.changeRequest = other
	env.srv.execModulesChangeRequest(context.Background(), own)
	if env.srv.changeRequest != other {
		t.Error("a change request created during the execution was cleared")
	}
	env.srv.changeRequest = own
	env.srv.execModulesChangeRequest(context.Background(), own)
	if env.srv.changeRequest != nil {
		t.Error("the executed change request was not cleared")
	}
}

func auAssertIdle(t *testing.T, srv *Service) {
	t.Helper()
	if jobs := srv.jobsHandler.CurrentSlotJobs(nil); len(jobs) != 0 {
		t.Errorf("job slots still occupied: %v", jobs)
	}
}

func TestService_autoUpdateTick_Run(t *testing.T) {
	env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
	env.install(t, "mod-a", "v1.0.0", nil, auConfigModfile)
	env.offer("mod-a", "v1.0.1", nil, auConfigModfile)
	env.deploy("mod-a", func(d *pkg_models.Deployment) {
		d.Configs = map[string]pkg_models.DeploymentUserConfig{"cfg": {Value: coStringValue("kept")}}
	})
	env.install(t, "mod-n", "v1.0.0", nil, "")
	env.offer("mod-n", "v1.0.5", nil, "")
	env.install(t, "mod-x", "v1.0.0", nil, "")
	env.offer("mod-x", "v2.0.0", nil, "")
	ctx := context.Background()
	var slotsDuringDeploymentUpdate []int
	env.deps.onUpdate = func() {
		for slot := range env.srv.jobsHandler.CurrentSlotJobs(nil) {
			slotsDuringDeploymentUpdate = append(slotsDuringDeploymentUpdate, slot)
		}
	}

	if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateRan {
		t.Fatalf("expected run, got %v", outcome)
	}
	if !reflect.DeepEqual(slotsDuringDeploymentUpdate, []int{deploymentJobSlotNum}) {
		t.Errorf("deployments must be updated in the deployment job alone, occupied slots: %v", slotsDuringDeploymentUpdate)
	}
	if got := env.mods.updateList(); !reflect.DeepEqual(got, []string{"mod-a@v1.0.1", "mod-n@v1.0.5"}) {
		t.Errorf("unexpected module updates %v", got)
	}
	if got := env.deps.updatedIds(); !reflect.DeepEqual(got, []string{"mod-a"}) {
		t.Errorf("unexpected deployment updates %v", got)
	}
	if v := env.deps.updated["mod-a"].Configs["cfg"]; v.String != "kept" {
		t.Errorf("deployment setup not carried over, got %+v", v)
	}
	row := env.db.get()
	if row.LastRunDate != "2026-11-16" || row.LastRun == nil {
		t.Fatalf("run not recorded: %+v", row)
	}
	wantUpdated := []lib_models.AutoUpdateUpdatedModule{{Id: "mod-a", From: "v1.0.0", To: "v1.0.1"}, {Id: "mod-n", From: "v1.0.0", To: "v1.0.5"}}
	if !reflect.DeepEqual(row.LastRun.Updated, wantUpdated) || len(row.LastRun.Failed) != 0 || row.LastRun.HasError {
		t.Errorf("unexpected run %+v", row.LastRun)
	}
	if len(row.LastRun.Skipped) != 1 || row.LastRun.Skipped[0].Id != "mod-x" || row.LastRun.Skipped[0].Reason != lib_constants.AutoUpdateReasonLevelExceeded {
		t.Errorf("unexpected skipped %+v", row.LastRun.Skipped)
	}
	if env.srv.changeRequest != nil {
		t.Error("change request of the run was not cleared")
	}
	auAssertIdle(t, env.srv)

	// once per window, also for a restarted service reading the same database
	refreshes := env.repos.refreshCount()
	env.setNow(time.Date(2026, 11, 16, 4, 30, 0, 0, time.UTC))
	if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateAlreadyRun {
		t.Errorf("expected already run, got %v", outcome)
	}
	restarted := New(env.repos, env.mods, env.deps, &auTestAux{}, nil, nil, env.db, handler_jobs.New(ctx, handler_jobs.Config{}), nil)
	restarted.now = env.getNow
	if outcome := restarted.autoUpdateTick(ctx); outcome != autoUpdateAlreadyRun {
		t.Errorf("restarted: expected already run, got %v", outcome)
	}
	if env.repos.refreshCount() != refreshes || env.db.runDateWrites != 1 {
		t.Error("a second run was started in the same window")
	}
}

func TestService_autoUpdateTick_CrossingMidnightOncePerWindow(t *testing.T) {
	env := newAutoUpdateTestEnv(t, auEnabled("patch", "23:00", "01:00"))
	ctx := context.Background()
	env.setNow(time.Date(2026, 11, 16, 23, 30, 0, 0, time.UTC))
	if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateRan {
		t.Fatalf("expected run, got %v", outcome)
	}
	if date := env.db.get().LastRunDate; date != "2026-11-16" {
		t.Errorf("got run date %q", date)
	}
	env.setNow(time.Date(2026, 11, 17, 0, 30, 0, 0, time.UTC))
	if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateAlreadyRun {
		t.Errorf("after midnight: expected already run, got %v", outcome)
	}
	env.setNow(time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC))
	if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateOutsideWindow {
		t.Errorf("noon: expected outside window, got %v", outcome)
	}
	env.setNow(time.Date(2026, 11, 17, 23, 0, 0, 0, time.UTC))
	if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateRan {
		t.Errorf("next window: expected run, got %v", outcome)
	}
	if env.db.runDateWrites != 2 {
		t.Errorf("expected 2 runs, got %d", env.db.runDateWrites)
	}
}

func TestService_autoUpdateTick_NotDue(t *testing.T) {
	ctx := context.Background()
	t.Run("disabled by default", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, nil)
		if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateDisabled {
			t.Errorf("expected disabled, got %v", outcome)
		}
	})
	t.Run("outside window", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, auEnabled("patch", "04:00", "05:00"))
		if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateOutsideWindow {
			t.Errorf("expected outside window, got %v", outcome)
		}
		if env.repos.refreshCount() != 0 {
			t.Error("refresh started outside the window")
		}
	})
	t.Run("window ended during refresh", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
		env.install(t, "mod-a", "v1.0.0", nil, "")
		env.offer("mod-a", "v1.0.1", nil, "")
		env.setNow(time.Date(2026, 11, 16, 4, 59, 0, 0, time.UTC))
		env.repos.onRefresh = func() { env.setNow(time.Date(2026, 11, 16, 5, 0, 0, 0, time.UTC)) }
		if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateOutsideWindow {
			t.Errorf("expected outside window, got %v", outcome)
		}
		if env.db.runDateWrites != 0 || len(env.mods.updateList()) != 0 {
			t.Error("run started after the window end")
		}
	})
	t.Run("disabled during refresh", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
		env.repos.onRefresh = func() {
			if err := env.srv.UpdateAutoUpdateSettings(ctx, lib_models.AutoUpdateSettings{Level: "patch", WindowStart: "02:00", WindowEnd: "05:00", TimeZone: "UTC"}); err != nil {
				t.Error(err)
			}
		}
		if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateDisabled {
			t.Errorf("expected disabled, got %v", outcome)
		}
		if env.db.runDateWrites != 0 {
			t.Error("run started after it was disabled")
		}
	})
}

func TestService_autoUpdateTick_Busy(t *testing.T) {
	ctx := context.Background()
	setup := func(t *testing.T) *auTestEnv {
		env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
		env.install(t, "mod-a", "v1.0.0", nil, "")
		env.offer("mod-a", "v1.0.1", nil, "")
		return env
	}
	assertNothingDone := func(t *testing.T, env *auTestEnv) {
		t.Helper()
		if env.db.runDateWrites != 0 || env.db.get().LastRun != nil {
			t.Error("an aborted run was recorded")
		}
		if len(env.mods.updateList()) != 0 {
			t.Error("modules were updated")
		}
	}

	t.Run("pending change request", func(t *testing.T) {
		env := setup(t)
		cr := &modulesChangeRequest{Remove: []string{"mod"}}
		env.srv.changeRequest = cr
		if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateBusy {
			t.Errorf("expected busy, got %v", outcome)
		}
		if env.srv.changeRequest != cr {
			t.Error("pending change request was replaced")
		}
		if env.repos.refreshCount() != 0 {
			t.Error("refresh must not run while a change request is pending")
		}
		assertNothingDone(t, env)
	})

	for _, slot := range []int{moduleJobSlotNum, deploymentJobSlotNum, repositoryJobSlotNum} {
		t.Run(fmt.Sprintf("active job in slot %d", slot), func(t *testing.T) {
			env := setup(t)
			job, err := env.srv.jobsHandler.CreateSlotJob(slot, "test")
			if err != nil {
				t.Fatal(err)
			}
			defer job.Done()
			if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateBusy {
				t.Errorf("expected busy, got %v", outcome)
			}
			if env.repos.refreshCount() != 0 {
				t.Error("refresh must not start while a job is active")
			}
			assertNothingDone(t, env)
		})
	}

	t.Run("change request created during refresh", func(t *testing.T) {
		env := setup(t)
		cr := &modulesChangeRequest{Remove: []string{"mod"}}
		env.repos.onRefresh = func() {
			env.srv.mu.Lock()
			env.srv.changeRequest = cr
			env.srv.mu.Unlock()
		}
		if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateBusy {
			t.Errorf("expected busy, got %v", outcome)
		}
		if env.srv.changeRequest != cr {
			t.Error("pending change request was replaced")
		}
		assertNothingDone(t, env)
		// retried by a later tick once the change request is gone
		env.srv.mu.Lock()
		env.srv.changeRequest = nil
		env.srv.mu.Unlock()
		env.repos.onRefresh = nil
		if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateRan {
			t.Errorf("retry: expected run, got %v", outcome)
		}
	})

	t.Run("job started during refresh", func(t *testing.T) {
		env := setup(t)
		var job *handler_jobs.Job
		env.repos.onRefresh = func() {
			var err error
			job, err = env.srv.jobsHandler.CreateSlotJob(deploymentJobSlotNum, "test")
			if err != nil {
				t.Error(err)
			}
		}
		if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateBusy {
			t.Errorf("expected busy, got %v", outcome)
		}
		job.Done()
		assertNothingDone(t, env)
	})
}

func TestService_autoUpdateTick_NeedsInputLeavesModuleUntouched(t *testing.T) {
	env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
	env.install(t, "mod-a", "v1.0.0", nil, auConfigModfile)
	env.offer("mod-a", "v1.0.1", nil, auConfigModfile)
	env.deploy("mod-a", func(d *pkg_models.Deployment) {
		d.Configs = map[string]pkg_models.DeploymentUserConfig{"cfg": {Value: coStringValue("")}}
	})
	if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateRan {
		t.Fatalf("expected run, got %v", outcome)
	}
	if got := env.mods.updateList(); len(got) != 0 {
		t.Errorf("module updated although its deployment needs input: %v", got)
	}
	run := env.db.get().LastRun
	if run == nil || len(run.Skipped) != 1 || run.Skipped[0].Reason != lib_constants.AutoUpdateReasonNeedsInput {
		t.Errorf("unexpected run %+v", run)
	}
	auAssertIdle(t, env.srv)
}

func TestService_autoUpdateTick_DeploymentFailsAfterModuleUpdate(t *testing.T) {
	env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
	env.install(t, "mod-a", "v1.0.0", nil, auConfigModfile)
	env.offer("mod-a", "v1.0.1", nil, auConfigModfile)
	env.deploy("mod-a", nil)
	// the installed declaration differs from the repository one the pre-check used
	env.mods.afterUpdate = func(m *pkg_models.Module) {
		m.Configs["cfg"] = coTextConfig(true, nil)
	}
	if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateRan {
		t.Fatalf("expected run, got %v", outcome)
	}
	if got := env.deps.updatedIds(); len(got) != 0 {
		t.Errorf("deployment updated without a complete setup: %v", got)
	}
	run := env.db.get().LastRun
	if run == nil || len(run.Updated) != 0 || len(run.Failed) != 1 || run.Failed[0].Id != "mod-a" ||
		!strings.Contains(run.Failed[0].Error, "needs input") {
		t.Errorf("unexpected run %+v", run)
	}
	auAssertIdle(t, env.srv)
}

func TestService_autoUpdateTick_FailedDependencyBlocksDependent(t *testing.T) {
	env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
	// mod-a sorts before its dependency mod-l, so id order alone would update it first
	env.install(t, "mod-l", "v1.0.0", nil, "")
	env.offer("mod-l", "v1.0.1", nil, "")
	env.install(t, "mod-a", "v1.0.0", []string{"mod-l"}, "")
	env.offer("mod-a", "v1.0.1", []string{"mod-l"}, "")
	// mod-b depends on mod-n, which succeeds
	env.install(t, "mod-n", "v1.0.0", nil, "")
	env.offer("mod-n", "v1.0.1", nil, "")
	env.install(t, "mod-b", "v1.0.0", []string{"mod-n"}, "")
	env.offer("mod-b", "v1.0.1", []string{"mod-n"}, "")
	for _, id := range []string{"mod-a", "mod-b", "mod-l", "mod-n"} {
		env.deploy(id, nil)
	}
	env.mods.updateErr = map[string]error{"mod-l": errors.New("disk full")}
	if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateRan {
		t.Fatalf("expected run, got %v", outcome)
	}
	if got := env.mods.updateList(); !reflect.DeepEqual(got, []string{"mod-n@v1.0.1", "mod-b@v1.0.1"}) {
		t.Errorf("unexpected module updates %v", got)
	}
	if got := env.deps.updatedIds(); !reflect.DeepEqual(got, []string{"mod-b", "mod-n"}) {
		t.Errorf("unexpected deployment updates %v", got)
	}
	run := env.db.get().LastRun
	if run == nil {
		t.Fatal("run not recorded")
	}
	wantFailed := []lib_models.AutoUpdateFailedModule{
		{Id: "mod-a", Error: "not applied, dependency 'mod-l' failed"},
		{Id: "mod-l", Error: "disk full"},
	}
	if !reflect.DeepEqual(run.Failed, wantFailed) {
		t.Errorf("failed %+v, want %+v", run.Failed, wantFailed)
	}
	if len(run.Updated) != 2 || run.Updated[0].Id != "mod-b" || run.Updated[1].Id != "mod-n" {
		t.Errorf("unexpected updated %+v", run.Updated)
	}
	auAssertIdle(t, env.srv)
}

func TestService_AutoUpdateLoop(t *testing.T) {
	env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
	ctx, cf := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		env.srv.AutoUpdateLoop(ctx, time.Millisecond)
		close(done)
	}()
	deadline := time.After(time.Second)
	for env.db.get().LastRun == nil {
		select {
		case <-deadline:
			t.Fatal("loop did not run the auto-update")
		case <-time.After(time.Millisecond):
		}
	}
	cf()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("loop did not stop on context cancel")
	}
	if env.repos.refreshCount() != 1 {
		t.Errorf("expected one refresh, got %d", env.repos.refreshCount())
	}
}

// auFileModfile declares a required file with a default source.
const auFileModfile = `files:
  conf:
    source: conf.json
    userInput:
      name: Conf
      type: json
    targets:
      - mountPoint: /etc/conf.json
        services:
          - srv
`

func TestService_planAutoUpdate_CarryOverChecks(t *testing.T) {
	ctx := context.Background()

	t.Run("global config", func(t *testing.T) {
		cases := []struct {
			name   string
			config *pkg_models.Config
			reason string
		}{
			{"valid for the new declaration", &pkg_models.Config{Id: "gc-1", Value: coStringValue("v")}, ""},
			{"invalid for the new declaration", &pkg_models.Config{Id: "gc-1", Value: coIntValue(1)}, lib_constants.CarryOverReasonTypeMismatch},
			{"deleted", nil, lib_constants.CarryOverReasonGlobalConfigNotFound},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				env := newAutoUpdateTestEnv(t, nil)
				env.install(t, "mod-a", "v1.0.0", nil, auConfigModfile)
				env.offer("mod-a", "v1.0.1", nil, auConfigModfile)
				env.deploy("mod-a", func(d *pkg_models.Deployment) {
					d.GlobalConfigs = map[string]pkg_models.DeploymentGlobalConfig{"cfg": {Id: "gc-1", Reference: "cfg"}}
				})
				if c.config != nil {
					env.gcs.configs["gc-1"] = *c.config
				}
				plan, err := auPlanCtx(ctx, env, "patch")
				if err != nil {
					t.Fatal(err)
				}
				if c.reason == "" {
					if got := auChanged(plan); !reflect.DeepEqual(got, []string{"mod-a@v1.0.1"}) || len(plan.skipped) != 0 {
						t.Errorf("unexpected plan changed=%v skipped=%+v", got, plan.skipped)
					}
					return
				}
				want := []lib_models.CarryOverMissing{{Kind: "config", Ref: "cfg", Reason: c.reason}}
				if plan.changeRequest != nil || len(plan.skipped) != 1 || plan.skipped[0].Reason != lib_constants.AutoUpdateReasonNeedsInput ||
					!reflect.DeepEqual(plan.skipped[0].Missing, want) {
					t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
				}
			})
		}
	})

	t.Run("default file", func(t *testing.T) {
		cases := []struct {
			name    string
			content []byte // nil: missing in the repository
			skipped bool
			failed  bool
		}{
			{"non-empty default answers the required file", []byte("{}"), false, false},
			{"empty default does not, as after the installation", []byte{}, true, false},
			{"unreadable default fails the module", nil, false, true},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				env := newAutoUpdateTestEnv(t, nil)
				env.install(t, "mod-a", "v1.0.0", nil, auFileModfile)
				env.offer("mod-a", "v1.0.1", nil, auFileModfile)
				if c.content != nil {
					env.repos.modules["mod-a|"+auSource+"|"+auChannel]["conf.json"] = &fstest.MapFile{Data: c.content}
				}
				env.deploy("mod-a", nil)
				plan, err := auPlanCtx(ctx, env, "patch")
				if err != nil {
					t.Fatal(err)
				}
				switch {
				case c.skipped:
					want := []lib_models.CarryOverMissing{{Kind: "file", Ref: "conf", Reason: lib_constants.CarryOverReasonNotSet}}
					if plan.changeRequest != nil || len(plan.skipped) != 1 || !reflect.DeepEqual(plan.skipped[0].Missing, want) {
						t.Errorf("unexpected plan changed=%v skipped=%+v", auChanged(plan), plan.skipped)
					}
				case c.failed:
					if plan.changeRequest != nil || len(plan.failed) != 1 || plan.failed[0].Id != "mod-a" {
						t.Errorf("unexpected plan changed=%v failed=%+v", auChanged(plan), plan.failed)
					}
				default:
					if got := auChanged(plan); !reflect.DeepEqual(got, []string{"mod-a@v1.0.1"}) || len(plan.skipped)+len(plan.failed) != 0 {
						t.Errorf("unexpected plan changed=%v skipped=%+v failed=%+v", got, plan.skipped, plan.failed)
					}
				}
			})
		}
	})
}

func TestService_autoUpdateTick_DeploymentSlotReservedDuringModuleUpdate(t *testing.T) {
	ctx := context.Background()
	t.Run("deployed module", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
		env.install(t, "mod-a", "v1.0.0", nil, "")
		env.offer("mod-a", "v1.0.1", nil, "")
		env.deploy("mod-a", nil)
		var slotHeld bool
		var deleteErr error
		env.mods.afterUpdate = func(_ *pkg_models.Module) {
			_, slotHeld = env.srv.jobsHandler.CurrentSlotJob(deploymentJobSlotNum)
			if slotHeld {
				_, deleteErr = env.srv.DeleteDeployments(ctx, []string{"mod-a"}, false)
			}
		}
		if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateRan {
			t.Fatalf("expected run, got %v", outcome)
		}
		if !slotHeld || !lib_errors.IsOf[lib_errors.ErrActiveJob](deleteErr) {
			t.Errorf("deployment slot free while the module job ran: held=%v delete err=%v", slotHeld, deleteErr)
		}
		if got := env.deps.updatedIds(); !reflect.DeepEqual(got, []string{"mod-a"}) {
			t.Errorf("unexpected deployment updates %v", got)
		}
		auAssertIdle(t, env.srv)
	})
	t.Run("no deployed module", func(t *testing.T) {
		env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
		env.install(t, "mod-a", "v1.0.0", nil, "")
		env.offer("mod-a", "v1.0.1", nil, "")
		slotHeld := true
		env.mods.afterUpdate = func(_ *pkg_models.Module) {
			_, slotHeld = env.srv.jobsHandler.CurrentSlotJob(deploymentJobSlotNum)
		}
		if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateRan {
			t.Fatalf("expected run, got %v", outcome)
		}
		if slotHeld {
			t.Error("deployment slot reserved without an affected deployment")
		}
		auAssertIdle(t, env.srv)
	})
}

func TestService_autoUpdateTick_ShutdownDuringModuleUpdate(t *testing.T) {
	env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
	env.install(t, "mod-a", "v1.0.0", nil, "")
	env.offer("mod-a", "v1.0.1", nil, "")
	env.deploy("mod-a", nil)
	env.install(t, "mod-n", "v1.0.0", nil, "")
	env.offer("mod-n", "v1.0.1", nil, "")
	ctx, cf := context.WithCancel(context.Background())
	defer cf()
	env.mods.afterUpdate = func(_ *pkg_models.Module) { cf() }
	if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateRan {
		t.Fatalf("expected run, got %v", outcome)
	}
	row := env.db.get()
	if row.LastRun == nil {
		t.Fatal("run interrupted by the shutdown was not recorded")
	}
	wantUpdated := []lib_models.AutoUpdateUpdatedModule{{Id: "mod-n", From: "v1.0.0", To: "v1.0.1"}}
	if !reflect.DeepEqual(row.LastRun.Updated, wantUpdated) {
		t.Errorf("updated: want %+v, got %+v", wantUpdated, row.LastRun.Updated)
	}
	if f := row.LastRun.Failed; len(f) != 1 || f[0].Id != "mod-a" || !strings.HasPrefix(f[0].Error, "deployment not updated: run interrupted") {
		t.Errorf("unexpected failed %+v", f)
	}
	if got := env.deps.updatedIds(); len(got) != 0 {
		t.Errorf("deployment update started during shutdown: %v", got)
	}
	auAssertIdle(t, env.srv)
}

func TestService_autoUpdateTick_PlanningErrorRetriedAfterBackoff(t *testing.T) {
	env := newAutoUpdateTestEnv(t, auEnabled("patch", "02:00", "05:00"))
	env.install(t, "mod-a", "v1.0.0", nil, "")
	env.offer("mod-a", "v1.0.1", nil, "")
	setGetErr := func(err error) {
		env.mods.mu.Lock()
		env.mods.getErr = err
		env.mods.mu.Unlock()
	}
	ctx := context.Background()
	start := env.getNow()

	setGetErr(errors.New("modules unavailable"))
	if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateError {
		t.Fatalf("expected error, got %v", outcome)
	}
	row := env.db.get()
	if env.db.runDateWrites != 0 || row.LastRunDate != "" {
		t.Errorf("planning error claimed the window date: %+v", row)
	}
	if row.LastRun == nil || !row.LastRun.HasError {
		t.Errorf("planning error not reported: %+v", row.LastRun)
	}
	auAssertIdle(t, env.srv)

	setGetErr(nil)
	env.setNow(start.Add(autoUpdateRetryDelay - time.Minute))
	if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateRetryLater {
		t.Errorf("before the retry delay: expected retry later, got %v", outcome)
	}
	if env.repos.refreshCount() != 1 {
		t.Errorf("repositories refreshed during the retry delay: %d refreshes", env.repos.refreshCount())
	}

	env.setNow(start.Add(autoUpdateRetryDelay))
	if outcome := env.srv.autoUpdateTick(ctx); outcome != autoUpdateRan {
		t.Fatalf("after the retry delay: expected run, got %v", outcome)
	}
	row = env.db.get()
	if row.LastRunDate != "2026-11-16" || row.LastRun == nil || row.LastRun.HasError || len(row.LastRun.Updated) != 1 {
		t.Errorf("retry did not run: %+v %+v", row, row.LastRun)
	}
	auAssertIdle(t, env.srv)
}

func TestService_autoUpdateTick_InvalidStoredTimeZone(t *testing.T) {
	settings := auEnabled("patch", "02:00", "05:00")
	settings.TimeZone = "Nowhere/Zone"
	env := newAutoUpdateTestEnv(t, settings)
	if outcome := env.srv.autoUpdateTick(context.Background()); outcome != autoUpdateError {
		t.Errorf("expected error, got %v", outcome)
	}
	if env.repos.refreshCount() != 0 || env.db.runDateWrites != 0 {
		t.Error("run started with an invalid time zone")
	}
}

// Steps through the DST transition days of Europe/Berlin and counts the runs per window date.
func TestService_autoUpdateTick_DSTTransitionDays(t *testing.T) {
	cases := []struct {
		name       string
		start, end string
		from       time.Time // UTC
		want       []string
	}{
		{"spring forward, window across the gap", "02:00", "04:00", time.Date(2026, 3, 28, 22, 0, 0, 0, time.UTC), []string{"2026-03-29"}},
		{"spring forward, window inside the gap", "02:00", "02:30", time.Date(2026, 3, 28, 22, 0, 0, 0, time.UTC), nil},
		{"spring forward, window crossing midnight into the gap", "23:00", "02:30", time.Date(2026, 3, 28, 21, 0, 0, 0, time.UTC), []string{"2026-03-28", "2026-03-29"}},
		{"fall back, repeated window", "02:00", "03:00", time.Date(2026, 10, 24, 22, 0, 0, 0, time.UTC), []string{"2026-10-25"}},
		{"fall back, window ending in the repeated hour", "01:00", "02:30", time.Date(2026, 10, 24, 22, 0, 0, 0, time.UTC), []string{"2026-10-25"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			settings := auEnabled("patch", c.start, c.end)
			settings.TimeZone = "Europe/Berlin"
			env := newAutoUpdateTestEnv(t, settings)
			var got []string
			for now := c.from; now.Before(c.from.Add(26 * time.Hour)); now = now.Add(5 * time.Minute) {
				env.setNow(now)
				if outcome := env.srv.autoUpdateTick(context.Background()); outcome == autoUpdateRan {
					got = append(got, env.db.get().LastRunDate)
				}
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("runs: want %v, got %v", c.want, got)
			}
		})
	}
}
