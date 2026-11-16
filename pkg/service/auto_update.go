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
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/SENERGY-Platform/mgw-module-lib/util/sem_ver"
	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	handler_jobs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	"golang.org/x/mod/semver"
)

const (
	autoUpdateJobPollInterval = 100 * time.Millisecond
	// autoUpdateRetryDelay is the earliest retry after an attempt that failed before anything was modified.
	autoUpdateRetryDelay = 30 * time.Minute
	// autoUpdateTimeZoneMaxLen matches the time_zone column.
	autoUpdateTimeZoneMaxLen = 64
)

var defaultAutoUpdateSettings = lib_models.AutoUpdateSettings{
	Level:       lib_constants.AutoUpdateLevelPatch,
	WindowStart: "02:00",
	WindowEnd:   "05:00",
	TimeZone:    "Europe/Berlin",
}

var autoUpdateLevelRank = map[string]int{
	lib_constants.AutoUpdateLevelPatch: 1,
	lib_constants.AutoUpdateLevelMinor: 2,
	lib_constants.AutoUpdateLevelMajor: 3,
}

type autoUpdateOutcome int

const (
	autoUpdateDisabled autoUpdateOutcome = iota
	autoUpdateOutsideWindow
	autoUpdateAlreadyRun
	autoUpdateDue
	autoUpdateBusy
	autoUpdateRetryLater
	autoUpdateError
	autoUpdateRan
)

// autoUpdateBackoff holds the earliest time of the next attempt. It has its own lock, autoUpdateDue reads it with and
// without s.mu held.
type autoUpdateBackoff struct {
	mu    sync.Mutex
	until time.Time
}

func (b *autoUpdateBackoff) set(until time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.until = until
}

func (b *autoUpdateBackoff) active(now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return now.Before(b.until)
}

func (s *Service) GetAutoUpdateSettings(ctx context.Context) (lib_models.AutoUpdateSettings, error) {
	autoUpdate, err := s.readAutoUpdate(ctx)
	if err != nil {
		return lib_models.AutoUpdateSettings{}, err
	}
	return autoUpdate.AutoUpdateSettings, nil
}

func (s *Service) UpdateAutoUpdateSettings(ctx context.Context, settings lib_models.AutoUpdateSettings) error {
	if err := validateAutoUpdateSettings(settings); err != nil {
		return lib_errors.Wrap[lib_errors.ErrInvalidInput](err)
	}
	return s.databaseHandler.WriteAutoUpdateSettings(ctx, settings)
}

func (s *Service) GetAutoUpdateStatus(ctx context.Context) (lib_models.AutoUpdateStatus, error) {
	autoUpdate, err := s.readAutoUpdate(ctx)
	if err != nil {
		return lib_models.AutoUpdateStatus{}, err
	}
	return lib_models.AutoUpdateStatus{
		AutoUpdateSettings: autoUpdate.AutoUpdateSettings,
		LastRun:            autoUpdate.LastRun,
	}, nil
}

func (s *Service) readAutoUpdate(ctx context.Context) (pkg_models.AutoUpdate, error) {
	autoUpdate, err := s.databaseHandler.ReadAutoUpdate(ctx)
	if err != nil {
		if lib_errors.IsOf[lib_errors.ErrNotFound](err) {
			return pkg_models.AutoUpdate{AutoUpdateSettings: defaultAutoUpdateSettings}, nil
		}
		return pkg_models.AutoUpdate{}, err
	}
	return autoUpdate, nil
}

func validateAutoUpdateSettings(settings lib_models.AutoUpdateSettings) error {
	if _, ok := autoUpdateLevelRank[settings.Level]; !ok {
		return fmt.Errorf("invalid level '%s'", settings.Level)
	}
	_, err := newAutoUpdateWindow(settings)
	return err
}

// autoUpdateWindow holds minutes after midnight on the wall clock of loc, start inclusive and end exclusive. start > end
// crosses midnight.
type autoUpdateWindow struct {
	start int
	end   int
	loc   *time.Location
}

func newAutoUpdateWindow(settings lib_models.AutoUpdateSettings) (autoUpdateWindow, error) {
	loc, err := loadTimeZone(settings.TimeZone)
	if err != nil {
		return autoUpdateWindow{}, err
	}
	start, err := parseClock(settings.WindowStart)
	if err != nil {
		return autoUpdateWindow{}, err
	}
	end, err := parseClock(settings.WindowEnd)
	if err != nil {
		return autoUpdateWindow{}, err
	}
	if start == end {
		return autoUpdateWindow{}, errors.New("window start and end must differ")
	}
	return autoUpdateWindow{start: start, end: end, loc: loc}, nil
}

// loadTimeZone accepts IANA names only, LoadLocation would map "" to UTC and "Local" to the zone of the host.
func loadTimeZone(name string) (*time.Location, error) {
	if name == "" || name == "Local" || len(name) > autoUpdateTimeZoneMaxLen {
		return nil, fmt.Errorf("invalid time zone '%s'", name)
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("invalid time zone '%s'", name)
	}
	return loc, nil
}

// parseClock parses HH:MM strictly, two digits each, into minutes after midnight.
func parseClock(s string) (int, error) {
	if len(s) != 5 || s[2] != ':' {
		return 0, fmt.Errorf("invalid time '%s', expected HH:MM", s)
	}
	for _, i := range []int{0, 1, 3, 4} {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("invalid time '%s', expected HH:MM", s)
		}
	}
	h := int(s[0]-'0')*10 + int(s[1]-'0')
	m := int(s[3]-'0')*10 + int(s[4]-'0')
	if h > 23 || m > 59 {
		return 0, fmt.Errorf("invalid time '%s', expected HH:MM", s)
	}
	return h*60 + m, nil
}

// date reports whether t lies within the window and returns the date the window containing t started on, so both
// parts of a window that crosses midnight belong to the same date. Only the wall clock counts: on a DST change, window
// minutes that are skipped do not occur and repeated ones map to the same date.
func (w autoUpdateWindow) date(t time.Time) (string, bool) {
	t = t.In(w.loc)
	m := t.Hour()*60 + t.Minute()
	if w.start < w.end {
		if m >= w.start && m < w.end {
			return t.Format(time.DateOnly), true
		}
		return "", false
	}
	if m >= w.start {
		return t.Format(time.DateOnly), true
	}
	if m < w.end {
		y, mo, d := t.Date()
		// Calendar arithmetic in UTC, a DST shift in t's location must not move the date.
		return time.Date(y, mo, d-1, 12, 0, 0, 0, time.UTC).Format(time.DateOnly), true
	}
	return "", false
}

// autoUpdateLevel classifies the change from version to next by the first differing SemVer component, taken literally
// below v1.0.0. newer is false if next is not SemVer-greater than version.
func autoUpdateLevel(version, next string) (level string, newer bool, err error) {
	c, err := sem_ver.CompareSemVer(next, version)
	if err != nil {
		return "", false, err
	}
	if c <= 0 {
		return "", false, nil
	}
	switch {
	case semver.Major(version) != semver.Major(next):
		return lib_constants.AutoUpdateLevelMajor, true, nil
	case semver.MajorMinor(version) != semver.MajorMinor(next):
		return lib_constants.AutoUpdateLevelMinor, true, nil
	default:
		return lib_constants.AutoUpdateLevelPatch, true, nil
	}
}

// exceedsAutoUpdateLevel treats an unknown maximum as exceeded by every change.
func exceedsAutoUpdateLevel(level, maxLevel string) bool {
	maxRank, ok := autoUpdateLevelRank[maxLevel]
	return !ok || autoUpdateLevelRank[level] > maxRank
}

// AutoUpdateLoop checks every interval whether an auto-update run is due and executes it, until ctx is done.
func (s *Service) AutoUpdateLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.autoUpdateTick(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// autoUpdateTick refreshes the repositories and runs the auto-update if it is enabled, inside its window and has not
// run in this window yet. While a change request is pending or a job is active nothing is recorded, so a later tick
// in the same window tries again.
func (s *Service) autoUpdateTick(ctx context.Context) (outcome autoUpdateOutcome) {
	defer func() {
		if st := recover(); st != nil {
			outcome = autoUpdateError
			logger.ErrorContext(ctx, "auto-update", slog_keys.Error, "panic", slog_keys.StackTrace, st)
		}
	}()
	if _, _, outcome = s.autoUpdateDue(ctx); outcome != autoUpdateDue {
		return outcome
	}
	s.mu.RLock()
	busy := s.autoUpdateBusy()
	s.mu.RUnlock()
	if busy {
		return autoUpdateBusy
	}
	job, skipped, err := s.refreshRepositoriesIfIdle(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "auto-update, refresh repositories", slog_keys.Error, err)
		s.autoUpdateBackoff.set(s.now().Add(autoUpdateRetryDelay))
		return autoUpdateError
	}
	if skipped {
		return autoUpdateBusy
	}
	if err = s.waitSlotJobDone(ctx, repositoryJobSlotNum, job.Id); err != nil {
		return autoUpdateError
	}
	if res, err := s.GetRefreshRepositoriesJobResult(ctx, job.Id); err == nil && (res.HasError || res.ResultsErrNum > 0) {
		logger.WarnContext(ctx, "auto-update, repositories refresh incomplete, using cached repositories", slog_keys.JobId, job.Id)
	}
	return s.runAutoUpdate(ctx)
}

// autoUpdateDue reads the settings and returns the date of the current window if a run is due. The window is evaluated
// in the configured time zone, independent of the zone the service uses for timestamps.
func (s *Service) autoUpdateDue(ctx context.Context) (pkg_models.AutoUpdate, string, autoUpdateOutcome) {
	autoUpdate, err := s.readAutoUpdate(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "auto-update, read settings", slog_keys.Error, err)
		return pkg_models.AutoUpdate{}, "", autoUpdateError
	}
	if !autoUpdate.Enabled {
		return autoUpdate, "", autoUpdateDisabled
	}
	window, err := newAutoUpdateWindow(autoUpdate.AutoUpdateSettings)
	if err != nil {
		logger.ErrorContext(ctx, "auto-update, invalid settings", slog_keys.Error, err)
		return autoUpdate, "", autoUpdateError
	}
	now := s.now()
	date, ok := window.date(now)
	if !ok {
		return autoUpdate, "", autoUpdateOutsideWindow
	}
	if date == autoUpdate.LastRunDate {
		return autoUpdate, date, autoUpdateAlreadyRun
	}
	if s.autoUpdateBackoff.active(now) {
		return autoUpdate, date, autoUpdateRetryLater
	}
	return autoUpdate, date, autoUpdateDue
}

// autoUpdateBusy requires s.mu to be locked by the caller.
func (s *Service) autoUpdateBusy() bool {
	return s.changeRequest != nil ||
		len(s.jobsHandler.CurrentSlotJobs([]int{moduleJobSlotNum, deploymentJobSlotNum, repositoryJobSlotNum})) > 0
}

// waitSlotJobDone returns once the job no longer occupies the slot. The context of a canceled job is done before the
// job has finished, so the slot is checked as well.
func (s *Service) waitSlotJobDone(ctx context.Context, slotNum int, jobId string) error {
	job, ok := s.jobsHandler.Job(jobId)
	if !ok {
		return nil
	}
	select {
	case <-job.Context().Done():
	case <-ctx.Done():
		return ctx.Err()
	}
	ticker := time.NewTicker(autoUpdateJobPollInterval)
	defer ticker.Stop()
	for {
		current, ok := s.jobsHandler.CurrentSlotJob(slotNum)
		if !ok || current.Id != jobId {
			return nil
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

type autoUpdatePlan struct {
	changeRequest *modulesChangeRequest // nil if no module is updated
	// deployed holds the modules of the change request that had a deployment when the plan was made, with the version
	// the deployment was created for.
	deployed  map[string]string
	repairs   []autoUpdateRepair // deployments that lag their installed module and are updated without a module change
	fallbacks map[string]autoUpdateFallback
	skipped   []lib_models.AutoUpdateSkippedModule
	failed    []lib_models.AutoUpdateFailedModule
}

// autoUpdateFallback is the repair check of the lagging deployment of a module of the change request. It is applied
// only if the update of that module fails and leaves the installed module the plan was made against.
type autoUpdateFallback struct {
	repairs []autoUpdateRepair // at most one
	skipped []lib_models.AutoUpdateSkippedModule
	failed  []lib_models.AutoUpdateFailedModule
}

// autoUpdateRepair is a deployment that was created for an older version of its installed module, in the same source
// and channel, and is updated to the installed version.
type autoUpdateRepair struct {
	id      string
	from    string // version the deployment was created for
	to      string // installed version
	source  string
	channel string
}

func (s *Service) runAutoUpdate(ctx context.Context) autoUpdateOutcome {
	run := lib_models.AutoUpdateRun{Start: s.now()}
	prepared, date, outcome := s.prepareAutoUpdate(ctx)
	if outcome != autoUpdateRan && prepared.err == nil {
		return outcome
	}
	logger.InfoContext(ctx, "auto-update run start", slog_keys.Date, date)
	if prepared.err != nil {
		run.ErrorResult = lib_models.NewErrorResult(prepared.err.Error())
	}
	run.Skipped = prepared.skipped
	run.Failed = prepared.failed
	if prepared.modJob != nil || prepared.depJob != nil {
		s.execAutoUpdate(ctx, prepared, &run)
	}
	run.End = s.now()
	// Stable, a module can be listed for its update and for its lagging deployment.
	slices.SortStableFunc(run.Updated, func(a, b lib_models.AutoUpdateUpdatedModule) int { return strings.Compare(a.Id, b.Id) })
	slices.SortStableFunc(run.Skipped, func(a, b lib_models.AutoUpdateSkippedModule) int { return strings.Compare(a.Id, b.Id) })
	slices.SortStableFunc(run.Failed, func(a, b lib_models.AutoUpdateFailedModule) int { return strings.Compare(a.Id, b.Id) })
	// A shutdown during the run must not lose its outcome.
	if err := s.databaseHandler.WriteAutoUpdateLastRun(context.WithoutCancel(ctx), run); err != nil {
		logger.ErrorContext(ctx, "auto-update, write last run", slog_keys.Error, err)
	}
	if run.HasError {
		logger.ErrorContext(ctx, "auto-update run aborted", slog_keys.Date, date, slog_keys.Error, run.ErrorMsg)
	}
	logger.InfoContext(
		ctx,
		"auto-update run end",
		slog_keys.Date, date,
		slog_keys.Updated, len(run.Updated),
		slog_keys.Skipped, len(run.Skipped),
		slog_keys.Failed, len(run.Failed),
	)
	return outcome
}

type preparedAutoUpdate struct {
	autoUpdatePlan
	modJob *handler_jobs.Job // nil if no module is updated
	depJob *handler_jobs.Job // nil if no deployment is affected
	err    error             // set if the run failed before anything was modified, the window date is then not claimed
}

func (p *preparedAutoUpdate) releaseJobs() {
	for _, job := range []*handler_jobs.Job{p.modJob, p.depJob} {
		if job != nil {
			job.Done()
		}
	}
	p.modJob, p.depJob = nil, nil
}

// prepareAutoUpdate checks under s.mu that the run is still due and nothing else is pending or running, plans the run
// and claims the window date. If modules are updated it occupies the module job slot and sets the change request, if a
// deployment is affected the deployment job slot, both before the lock is released, so no other change request or job
// can start in between. If planning fails, the date stays unclaimed and the next attempt waits for autoUpdateRetryDelay;
// the outcome is then autoUpdateError with the error in the returned plan.
func (s *Service) prepareAutoUpdate(ctx context.Context) (preparedAutoUpdate, string, autoUpdateOutcome) {
	// Read before the lock, a slow secret manager must not block the service. Planning treats a secret stored after
	// this read as of unknown type.
	secrets := s.readSecretTypesOf(ctx, nil)
	s.mu.Lock()
	defer s.mu.Unlock()
	autoUpdate, date, outcome := s.autoUpdateDue(ctx)
	if outcome != autoUpdateDue {
		return preparedAutoUpdate{}, "", outcome
	}
	if s.autoUpdateBusy() {
		return preparedAutoUpdate{}, "", autoUpdateBusy
	}
	var prepared preparedAutoUpdate
	handedOver := false
	// Also on a panic, the slots must not stay occupied and the next attempt must wait.
	defer func() {
		if !handedOver {
			prepared.releaseJobs()
			s.autoUpdateBackoff.set(s.now().Add(autoUpdateRetryDelay))
		}
	}()
	prepared.autoUpdatePlan, prepared.err = s.planAutoUpdate(ctx, autoUpdate.Level, secrets)
	if prepared.err == nil && (prepared.changeRequest != nil || len(prepared.repairs) > 0) {
		prepared.err = s.reserveAutoUpdateJobs(&prepared)
	}
	if prepared.err != nil {
		prepared.releaseJobs()
		s.autoUpdateBackoff.set(s.now().Add(autoUpdateRetryDelay))
		return prepared, date, autoUpdateError
	}
	// Claimed before anything is modified, so a restart within the window does not run again.
	if err := s.databaseHandler.WriteAutoUpdateRunDate(ctx, date); err != nil {
		prepared.releaseJobs()
		logger.ErrorContext(ctx, "auto-update, write run date", slog_keys.Error, err)
		s.autoUpdateBackoff.set(s.now().Add(autoUpdateRetryDelay))
		return preparedAutoUpdate{}, "", autoUpdateError
	}
	if prepared.changeRequest != nil {
		s.changeRequest = prepared.changeRequest
	}
	handedOver = true
	return prepared, date, autoUpdateRan
}

// reserveAutoUpdateJobs requires s.mu to be locked by the caller.
func (s *Service) reserveAutoUpdateJobs(prepared *preparedAutoUpdate) error {
	var err error
	if prepared.changeRequest != nil {
		if prepared.modJob, err = s.jobsHandler.CreateSlotJob(moduleJobSlotNum, "auto-update modules"); err != nil {
			return err
		}
	}
	// Taken now, so no deployment can be deleted or changed while its module is updated.
	if len(prepared.deployed) > 0 || len(prepared.repairs) > 0 {
		prepared.depJob, err = s.jobsHandler.CreateSlotJob(deploymentJobSlotNum, "auto-update deployments")
		if err != nil {
			return err
		}
	}
	return nil
}

// planAutoUpdate selects the installed modules with a newer version in their repository channel that can be updated
// within level, together with their dependencies, and checks that every affected deployment can keep its setup. It also
// selects the deployments that lag their installed module, so a run that updated a module but not its deployment is
// completed. secrets is read before s.mu was taken. Requires s.mu to be locked by the caller.
func (s *Service) planAutoUpdate(ctx context.Context, level string, secrets secretTypesSnapshot) (autoUpdatePlan, error) {
	plan := autoUpdatePlan{deployed: make(map[string]string), fallbacks: make(map[string]autoUpdateFallback)}
	installedMods, err := s.modulesHandler.GetModules(ctx, pkg_models.ModulesFilterWithName{}, false)
	if err != nil {
		return autoUpdatePlan{}, err
	}
	if len(installedMods) == 0 {
		return plan, nil
	}
	ids := slices.Sorted(maps.Keys(installedMods))
	deployments, err := s.deploymentsHandler.GetDeploymentsByModuleIds(ctx, pkg_models.DeploymentsFilterWithState{
		DeploymentsFilter: pkg_models.DeploymentsFilter{ModuleIds: ids},
	})
	if err != nil {
		return autoUpdatePlan{}, err
	}
	refs, err := s.readCarryOverRefs(ctx, deployments, secrets)
	if err != nil {
		return autoUpdatePlan{}, err
	}
	repoMods, err := s.repositoriesHandler.GetModules(ctx, pkg_models.RepositoryModulesFilter{Ids: ids})
	if err != nil {
		return autoUpdatePlan{}, err
	}
	available := make(map[string]string)
	for _, repoMod := range repoMods {
		installedMod, ok := installedMods[repoMod.Id]
		if ok && installedMod.Source == repoMod.Source && installedMod.Channel == repoMod.Channel {
			available[repoMod.Id] = repoMod.Version
		}
	}
	changes := make(map[string]changeItem)
	for _, id := range ids {
		installedMod := installedMods[id]
		nextVersion, ok := available[id]
		if !ok || nextVersion == installedMod.Version {
			continue
		}
		skip := func(reason string, missing []lib_models.CarryOverMissing) {
			plan.skipped = append(plan.skipped, lib_models.AutoUpdateSkippedModule{
				Id:          id,
				Version:     installedMod.Version,
				NextVersion: nextVersion,
				Reason:      reason,
				Missing:     missing,
			})
		}
		if installedMod.Err != nil {
			skip(lib_constants.AutoUpdateReasonModuleError, nil)
			continue
		}
		modLevel, newer, err := autoUpdateLevel(installedMod.Version, nextVersion)
		if err != nil {
			skip(lib_constants.AutoUpdateReasonInvalidVersion, nil)
			continue
		}
		if !newer {
			continue
		}
		if exceedsAutoUpdateLevel(modLevel, level) {
			skip(lib_constants.AutoUpdateReasonLevelExceeded, nil)
			continue
		}
		items, reason, missing, err := s.autoUpdateChangeSet(ctx, id, installedMods, deployments, refs, level)
		if err != nil {
			plan.failed = append(plan.failed, lib_models.AutoUpdateFailedModule{Id: id, Error: err.Error()})
			continue
		}
		if reason != "" {
			skip(reason, missing)
			continue
		}
		// Dependencies have to stay in their installed source and channel, so two change sets agree on a shared
		// module; a disagreement is treated like a dependency that cannot be updated.
		if slices.ContainsFunc(items, func(item changeItem) bool {
			other, ok := changes[item.Next.Mod.ID]
			return ok && (other.Next.Source != item.Next.Source || other.Next.Channel != item.Next.Channel || other.Next.Mod.Version != item.Next.Mod.Version)
		}) {
			skip(lib_constants.AutoUpdateReasonDependencyLevelExceeded, nil)
			continue
		}
		merged := maps.Clone(changes)
		for _, item := range items {
			if _, ok := merged[item.Next.Mod.ID]; !ok {
				merged[item.Next.Mod.ID] = item
			}
		}
		if autoUpdateRangeViolated(installedMods, merged) {
			skip(lib_constants.AutoUpdateReasonDependentRangeViolated, nil)
			continue
		}
		changes = merged
	}
	// A candidate skipped on its own can still be updated as part of a later change set, e.g. together with a dependent
	// whose new range accepts it.
	plan.skipped = slices.DeleteFunc(plan.skipped, func(item lib_models.AutoUpdateSkippedModule) bool {
		_, ok := changes[item.Id]
		return ok
	})
	for _, id := range ids {
		if _, ok := changes[id]; !ok {
			s.planAutoUpdateRepair(id, installedMods, deployments, refs, level, &plan)
			continue
		}
		var fallback autoUpdatePlan
		s.planAutoUpdateRepair(id, installedMods, deployments, refs, level, &fallback)
		if len(fallback.repairs)+len(fallback.skipped)+len(fallback.failed) > 0 {
			plan.fallbacks[id] = autoUpdateFallback{repairs: fallback.repairs, skipped: fallback.skipped, failed: fallback.failed}
		}
	}
	if len(changes) == 0 {
		return plan, nil
	}
	plan.changeRequest = &modulesChangeRequest{Created: s.now()}
	for _, id := range slices.Sorted(maps.Keys(changes)) {
		plan.changeRequest.Change = append(plan.changeRequest.Change, changes[id])
		if deployment, ok := deployments[id]; ok {
			plan.deployed[id] = deployment.ModuleVersion
		}
	}
	return plan, nil
}

// planAutoUpdateRepair adds the deployment of the installed module id to the plan if it lags the module, or the reason
// why it is left as it is. The checks match those of an update, against the installed declaration.
func (s *Service) planAutoUpdateRepair(
	id string,
	installedMods map[string]pkg_models.Module,
	deployments map[string]pkg_models.Deployment,
	refs carryOverRefs,
	level string,
	plan *autoUpdatePlan,
) {
	installedMod := installedMods[id]
	deployment, ok := deployments[id]
	// Without a readable declaration the installed version is unknown. A deployment of another source or channel is
	// left to the user, as the update of a module is.
	if !ok || installedMod.Err != nil || deployment.ModuleSource != installedMod.Source ||
		deployment.ModuleChannel != installedMod.Channel || deployment.ModuleVersion == installedMod.Version {
		return
	}
	skip := func(reason string, missing []lib_models.CarryOverMissing) {
		plan.skipped = append(plan.skipped, lib_models.AutoUpdateSkippedModule{
			Id:          id,
			Version:     deployment.ModuleVersion,
			NextVersion: installedMod.Version,
			Reason:      reason,
			Missing:     missing,
		})
	}
	if deployment.Err != nil {
		skip(lib_constants.AutoUpdateReasonDeploymentError, nil)
		return
	}
	depLevel, newer, err := autoUpdateLevel(deployment.ModuleVersion, installedMod.Version)
	switch {
	case err != nil:
		skip(lib_constants.AutoUpdateReasonInvalidVersion, nil)
		return
	case !newer:
		return
	case exceedsAutoUpdateLevel(depLevel, level):
		skip(lib_constants.AutoUpdateReasonLevelExceeded, nil)
		return
	}
	if reason, _ := autoUpdateDependenciesDeployed(installedMod.Dependencies, deployments); reason != "" {
		skip(reason, nil)
		return
	}
	_, missing, err := carryOverInstalled(installedMod, deployment, refs)
	switch {
	case err != nil:
		plan.failed = append(plan.failed, lib_models.AutoUpdateFailedModule{Id: id, Error: err.Error()})
		return
	case len(missing) > 0:
		skip(lib_constants.AutoUpdateReasonNeedsInput, missing)
		return
	}
	plan.repairs = append(plan.repairs, autoUpdateRepair{
		id:      id,
		from:    deployment.ModuleVersion,
		to:      installedMod.Version,
		source:  installedMod.Source,
		channel: installedMod.Channel,
	})
}

// autoUpdateDependenciesDeployed returns the reason and the dependency if a dependency has no readable deployment, the
// deployment handler requires one for every dependency of a deployment.
func autoUpdateDependenciesDeployed(dependencies map[string]string, deployments map[string]pkg_models.Deployment) (string, string) {
	for _, dep := range slices.Sorted(maps.Keys(dependencies)) {
		deployment, ok := deployments[dep]
		switch {
		case !ok:
			return lib_constants.AutoUpdateReasonDependencyNotDeployed, dep
		case deployment.Err != nil:
			return lib_constants.AutoUpdateReasonDeploymentError, dep
		}
	}
	return "", ""
}

// autoUpdateRangeViolated reports whether a version in changes lies outside the range an installed module declares for
// it in the version it has after changes, or a module in changes declares a range its dependency is outside of. Pairs
// that changes does not touch are not checked, the run does not cause them.
func autoUpdateRangeViolated(installedMods map[string]pkg_models.Module, changes map[string]changeItem) bool {
	version := func(id string) (string, bool) {
		if item, ok := changes[id]; ok {
			return item.Next.Mod.Version, true
		}
		mod, ok := installedMods[id]
		return mod.Version, ok && mod.Err == nil
	}
	for id, mod := range installedMods {
		dependencies := mod.Dependencies
		item, moved := changes[id]
		if moved {
			dependencies = item.Next.Mod.Dependencies
		}
		for dep, versionRange := range dependencies {
			if _, depMoved := changes[dep]; !moved && !depMoved {
				continue
			}
			v, ok := version(dep)
			if !ok {
				return true
			}
			if in, err := sem_ver.InSemVerRange(versionRange, v); err != nil || !in {
				return true
			}
		}
	}
	return false
}

// autoUpdateChangeSet resolves the modules an update of the installed module id changes and returns them, or the
// reason why the update is skipped. No items and no reason means there is nothing to update.
func (s *Service) autoUpdateChangeSet(
	ctx context.Context,
	id string,
	installedMods map[string]pkg_models.Module,
	deployments map[string]pkg_models.Deployment,
	refs carryOverRefs,
	level string,
) ([]changeItem, string, []lib_models.CarryOverMissing, error) {
	selectedRepoMods, err := s.selectRepoModules(ctx, []lib_models.ChangeRequestItem{{Id: id, Update: true}}, installedMods)
	if err != nil {
		return nil, "", nil, err
	}
	changeRequest := newModulesChangeRequest(selectedRepoMods, installedMods, nil)
	if len(changeRequest.Install) > 0 {
		return nil, lib_constants.AutoUpdateReasonNewDependency, nil, nil
	}
	items := changeRequest.Change
	slices.SortFunc(items, func(a, b changeItem) int { return strings.Compare(a.Next.Mod.ID, b.Next.Mod.ID) })
	i := slices.IndexFunc(items, func(item changeItem) bool { return item.Next.Mod.ID == id })
	if i < 0 {
		return nil, "", nil, nil
	}
	// The modfile is authoritative over the repository listing the candidate was found with.
	modLevel, newer, err := autoUpdateLevel(items[i].Previous.Version, items[i].Next.Mod.Version)
	switch {
	case err != nil:
		return nil, lib_constants.AutoUpdateReasonInvalidVersion, nil, nil
	case !newer:
		return nil, "", nil, nil
	case exceedsAutoUpdateLevel(modLevel, level):
		return nil, lib_constants.AutoUpdateReasonLevelExceeded, nil, nil
	}
	for _, item := range items {
		installedMod := installedMods[item.Next.Mod.ID]
		if installedMod.Err != nil {
			return nil, lib_constants.AutoUpdateReasonModuleError, nil, nil
		}
		if item.Next.Mod.ID == id {
			continue
		}
		if item.Next.Source != installedMod.Source || item.Next.Channel != installedMod.Channel {
			return nil, lib_constants.AutoUpdateReasonDependencyLevelExceeded, nil, nil
		}
		depLevel, newer, err := autoUpdateLevel(item.Previous.Version, item.Next.Mod.Version)
		if err != nil || !newer || exceedsAutoUpdateLevel(depLevel, level) {
			return nil, lib_constants.AutoUpdateReasonDependencyLevelExceeded, nil, nil
		}
	}
	for _, item := range items {
		deployment, ok := deployments[item.Next.Mod.ID]
		if !ok {
			continue
		}
		if deployment.Err != nil {
			return nil, lib_constants.AutoUpdateReasonDeploymentError, nil, nil
		}
		if reason := autoUpdateDeploymentLag(id, item, installedMods[item.Next.Mod.ID], deployment, level); reason != "" {
			return nil, reason, nil, nil
		}
		if reason, _ := autoUpdateDependenciesDeployed(item.Next.Mod.Dependencies, deployments); reason != "" {
			return nil, reason, nil, nil
		}
	}
	var missing []lib_models.CarryOverMissing
	for _, item := range items {
		deployment, ok := deployments[item.Next.Mod.ID]
		if !ok {
			continue
		}
		_, itemMissing, err := carryOverRepository(item.Next, deployment, refs)
		if err != nil {
			return nil, "", nil, fmt.Errorf("module '%s': %w", item.Next.Mod.ID, err)
		}
		missing = append(missing, itemMissing...)
	}
	if len(missing) > 0 {
		return nil, lib_constants.AutoUpdateReasonNeedsInput, missing, nil
	}
	return items, "", nil, nil
}

// autoUpdateDeploymentLag returns the reason if the deployment was created for another source or channel than its
// installed module, which is left to the user as in the repair, or if it lags its installed module and the change from
// the deployment's version to the new one is not an update within level.
func autoUpdateDeploymentLag(candidate string, item changeItem, installedMod pkg_models.Module, deployment pkg_models.Deployment, level string) string {
	if deployment.ModuleSource != installedMod.Source || deployment.ModuleChannel != installedMod.Channel {
		return lib_constants.AutoUpdateReasonDeploymentVariantDiffers
	}
	if deployment.ModuleVersion == installedMod.Version {
		return ""
	}
	depLevel, newer, err := autoUpdateLevel(deployment.ModuleVersion, item.Next.Mod.Version)
	switch {
	case err != nil:
		return lib_constants.AutoUpdateReasonInvalidVersion
	case !newer:
		// The deployment is at or ahead of the new version, the update would not move it forward.
		return lib_constants.AutoUpdateReasonDeploymentVariantDiffers
	case !exceedsAutoUpdateLevel(depLevel, level):
		return ""
	case item.Next.Mod.ID == candidate:
		return lib_constants.AutoUpdateReasonLevelExceeded
	default:
		return lib_constants.AutoUpdateReasonDependencyLevelExceeded
	}
}

// autoUpdateDeploymentCancelled is the error of a module whose deployment was not updated because the deployment job
// was cancelled; the next run updates the deployment.
const autoUpdateDeploymentCancelled = "deployment update cancelled"

// execAutoUpdate executes the change request, if any, in the module job slot it was prepared with and then updates the
// deployments of the changed modules and the lagging deployments in the deployment job slot that was reserved with it.
// The deployment of a module whose update failed is updated to its installed version if its fallback allows it.
func (s *Service) execAutoUpdate(ctx context.Context, prepared preparedAutoUpdate, run *lib_models.AutoUpdateRun) {
	modJob, depJob := prepared.modJob, prepared.depJob
	// Both slots are released on every path, a panic included.
	releaseModJob := sync.OnceFunc(func() {
		if modJob != nil {
			modJob.Done()
			logJobDone(ctx, modJob)
		}
	})
	defer releaseModJob()
	depJobHandedOver := false
	// Replaced once the deployment phase is decided, it remains if a panic ends the run before.
	depJobErr := lib_models.NewErrorResult("deployment phase not reached")
	defer func() {
		if depJob != nil && !depJobHandedOver {
			s.setUpdateDeploymentsJobResult(depJob.Id, lib_models.DeploymentUpdateJobResult{
				JobResult: lib_models.JobResult{JobId: depJob.Id, ErrorResult: depJobErr},
			})
			depJob.Done()
			logJobDone(ctx, depJob)
		}
	}()
	items := make(map[string]changeItem)
	reportErrs := make(map[string]string)
	if prepared.changeRequest != nil {
		for _, item := range prepared.changeRequest.Change {
			items[item.Next.Mod.ID] = item
		}
		// Cancelling the deployment job stops the module phase too, so no further deployed module is left behind.
		var stopOn *handler_jobs.Job
		if len(prepared.deployed) > 0 {
			stopOn = depJob
		}
		logJobStart(ctx, modJob)
		report := s.autoUpdateExecModules(ctx, modJob, stopOn, prepared.changeRequest)
		for _, item := range report.Failed {
			reportErrs[item.Id] = item.Error
		}
	}
	repairs := make(map[string]autoUpdateRepair)
	for _, repair := range prepared.repairs {
		repairs[repair.id] = repair
	}
	// Read regardless of a shutdown, which cancels ctx while the job ends, so updated modules are reported as updated.
	detachedCtx := context.WithoutCancel(ctx)
	// The installed state decides what was updated, the report is lost if the job panics.
	installedMods, err := s.modulesHandler.GetModules(
		detachedCtx,
		pkg_models.ModulesFilterWithName{ModulesFilter: pkg_models.ModulesFilter{
			Ids: slices.Concat(slices.Collect(maps.Keys(items)), slices.Collect(maps.Keys(repairs))),
		}},
		false,
	)
	from := make(map[string]string)
	to := make(map[string]string)
	var updated, toDeploy []string
	installed := func(id string, variant lib_models.ModuleVariant) bool {
		installedMod, ok := installedMods[id]
		return err == nil && ok && installedMod.Err == nil && installedMod.Version == variant.Version &&
			installedMod.Source == variant.Source && installedMod.Channel == variant.Channel
	}
	for _, id := range slices.Sorted(maps.Keys(items)) {
		item := items[id]
		if installed(id, lib_models.ModuleVariant{Source: item.Next.Source, Channel: item.Next.Channel, Version: item.Next.Mod.Version}) {
			from[id], to[id] = item.Previous.Version, item.Next.Mod.Version
			if depVersion, deployed := prepared.deployed[id]; deployed {
				if depVersion != "" {
					from[id] = depVersion
				}
				toDeploy = append(toDeploy, id)
			} else {
				updated = append(updated, id)
			}
			continue
		}
		msg, ok := reportErrs[id]
		switch {
		case ok:
		case err != nil:
			msg = "read modules: " + err.Error()
		default:
			msg = "module not updated"
		}
		run.Failed = append(run.Failed, lib_models.AutoUpdateFailedModule{Id: id, Error: msg})
		// The fallback was checked against the installed module, it only holds while that is still installed.
		fallback, ok := prepared.fallbacks[id]
		if !ok || !installed(id, item.Previous.ModuleVariant) {
			continue
		}
		run.Skipped = append(run.Skipped, fallback.skipped...)
		run.Failed = append(run.Failed, fallback.failed...)
		for _, repair := range fallback.repairs {
			from[id], to[id] = repair.from, repair.to
			toDeploy = append(toDeploy, id)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(repairs)) {
		repair := repairs[id]
		if installed(id, lib_models.ModuleVariant{Source: repair.source, Channel: repair.channel, Version: repair.to}) {
			from[id], to[id] = repair.from, repair.to
			toDeploy = append(toDeploy, id)
			continue
		}
		msg := "deployment not updated: module changed since the run was planned"
		if err != nil {
			msg = "read modules: " + err.Error()
		}
		run.Failed = append(run.Failed, lib_models.AutoUpdateFailedModule{Id: id, Error: msg})
	}
	releaseModJob()
	if len(toDeploy) > 0 {
		var msg string
		switch {
		case ctx.Err() != nil:
			// No deployment update is started during a shutdown.
			msg = "deployment not updated: run interrupted: " + ctx.Err().Error()
		case depJob == nil:
			msg = "deployment not updated: no deployment job"
		case depJob.Context().Err() != nil:
			msg = autoUpdateDeploymentCancelled
		}
		if msg != "" {
			for _, id := range toDeploy {
				run.Failed = append(run.Failed, lib_models.AutoUpdateFailedModule{Id: id, Error: msg})
			}
			toDeploy = nil
			depJobErr = lib_models.NewErrorResult(msg)
		}
	} else {
		depJobErr = lib_models.ErrorResult{}
	}
	if len(toDeploy) > 0 {
		depJobHandedOver = true
		deployed := s.autoUpdateExecDeployments(detachedCtx, depJob, toDeploy, installedMods, run)
		updated = append(updated, deployed...)
	}
	for _, id := range updated {
		run.Updated = append(run.Updated, lib_models.AutoUpdateUpdatedModule{Id: id, From: from[id], To: to[id]})
	}
}

// autoUpdateExecModules executes the change request in job. If stopOn is set, cancelling it stops the execution as
// cancelling job does: modules not applied yet are not applied.
func (s *Service) autoUpdateExecModules(
	ctx context.Context,
	job *handler_jobs.Job,
	stopOn *handler_jobs.Job,
	changeRequest *modulesChangeRequest,
) lib_models.ModulesChangeReport {
	jobResult := lib_models.ModulesChangeJobResult{
		JobResult: lib_models.JobResult{JobId: job.Id},
	}
	defer func() {
		if st := recover(); st != nil {
			jobResult.ErrorResult = lib_models.NewErrorResult(fmt.Sprintf("%v", st))
			logger.ErrorContext(ctx, "auto-update modules", slog_keys.JobId, job.Id, slog_keys.Error, "panic", slog_keys.StackTrace, st)
		}
		s.setModuleChangeJobResult(job.Id, jobResult)
	}()
	stopped := job.Context().Err
	if stopOn != nil {
		stopped = func() error {
			if stopOn.Context().Err() != nil {
				return errors.New(autoUpdateDeploymentCancelled)
			}
			return job.Context().Err()
		}
	}
	jobResult.ModulesChangeReport = s.execModulesChangeRequestUntil(job.Context(), changeRequest, stopped)
	return jobResult.ModulesChangeReport
}

// autoUpdateExecDeployments updates the deployments of the given updated modules with their carried over setup in the
// deployment job and returns the modules that are done, failures are added to run.
func (s *Service) autoUpdateExecDeployments(
	ctx context.Context,
	job *handler_jobs.Job,
	moduleIds []string,
	installedMods map[string]pkg_models.Module,
	run *lib_models.AutoUpdateRun,
) (done []string) {
	jobResult := lib_models.DeploymentUpdateJobResult{
		JobResult: lib_models.JobResult{JobId: job.Id},
	}
	errs := make(map[string]string)
	notDeployed := make(map[string]bool)
	defer func() {
		if st := recover(); st != nil {
			jobResult.ErrorResult = lib_models.NewErrorResult(fmt.Sprintf("%v", st))
			logger.ErrorContext(ctx, "auto-update deployments", slog_keys.JobId, job.Id, slog_keys.Error, "panic", slog_keys.StackTrace, st)
		}
		s.setUpdateDeploymentsJobResult(job.Id, jobResult)
		job.Done()
		logJobDone(ctx, job)
		results := make(map[string]lib_models.DeploymentUpdateResult)
		for _, result := range jobResult.Results {
			results[result.ModuleId] = result
		}
		for _, id := range moduleIds {
			msg, failed := errs[id]
			if !failed && !notDeployed[id] {
				msg, failed = deploymentUpdateErr(results, id, jobResult.ErrorResult)
			}
			if failed {
				run.Failed = append(run.Failed, lib_models.AutoUpdateFailedModule{Id: id, Error: msg})
				continue
			}
			done = append(done, id)
		}
	}()
	logJobStart(ctx, job)
	// The dependencies are read as well, the installed declaration is checked again for them.
	ids := slices.Clone(moduleIds)
	for _, id := range moduleIds {
		ids = append(ids, slices.Collect(maps.Keys(installedMods[id].Dependencies))...)
	}
	slices.Sort(ids)
	allDeployments, err := s.deploymentsHandler.GetDeploymentsByModuleIds(ctx, pkg_models.DeploymentsFilterWithState{
		DeploymentsFilter: pkg_models.DeploymentsFilter{ModuleIds: slices.Compact(ids)},
	})
	if err != nil {
		jobResult.ErrorResult = lib_models.NewErrorResult(err.Error())
		return
	}
	deployments := make(map[string]pkg_models.Deployment)
	for _, id := range moduleIds {
		if deployment, ok := allDeployments[id]; ok {
			deployments[id] = deployment
		}
	}
	refs, err := s.readCarryOverRefs(ctx, deployments, s.readSecretTypes(ctx, deployments))
	if err != nil {
		jobResult.ErrorResult = lib_models.NewErrorResult(err.Error())
		return
	}
	var userInputs []lib_models.DeploymentUserInput
	for _, id := range moduleIds {
		deployment, ok := deployments[id]
		if !ok {
			// removed while the module was updated
			notDeployed[id] = true
			continue
		}
		switch reason, dep := autoUpdateDependenciesDeployed(installedMods[id].Dependencies, allDeployments); reason {
		case lib_constants.AutoUpdateReasonDependencyNotDeployed:
			errs[id] = fmt.Sprintf("deployment not updated: dependency '%s' not deployed", dep)
			continue
		case lib_constants.AutoUpdateReasonDeploymentError:
			errs[id] = fmt.Sprintf("deployment not updated: deployment of dependency '%s' could not be read", dep)
			continue
		}
		input, missing, err := carryOverInstalled(installedMods[id], deployment, refs)
		switch {
		case err != nil:
			errs[id] = "deployment not updated: " + err.Error()
		case len(missing) > 0:
			errs[id] = "deployment not updated, needs input: " + formatCarryOverMissing(missing)
		default:
			userInputs = append(userInputs, input)
		}
	}
	if job.Context().Err() != nil {
		jobResult.ErrorResult = lib_models.NewErrorResult(autoUpdateDeploymentCancelled)
		return
	}
	s.updateDeployments(ctx, job.Context(), userInputs, &jobResult)
	return
}

func deploymentUpdateErr(results map[string]lib_models.DeploymentUpdateResult, id string, jobErr lib_models.ErrorResult) (string, bool) {
	result, ok := results[id]
	switch {
	case !ok && jobErr.HasError:
		return "deployment not updated: " + jobErr.ErrorMsg, true
	case !ok:
		return "deployment not updated", true
	case result.HasError:
		return "deployment not updated: " + result.ErrorMsg, true
	case result.AuxiliaryDeployments.HasError:
		return "auxiliary deployments not recreated: " + result.AuxiliaryDeployments.ErrorMsg, true
	case result.AuxiliaryDeployments.ResultsErrNum > 0:
		return fmt.Sprintf("auxiliary deployments not recreated: %d failed", result.AuxiliaryDeployments.ResultsErrNum), true
	}
	return "", false
}

func formatCarryOverMissing(missing []lib_models.CarryOverMissing) string {
	items := make([]string, 0, len(missing))
	for _, item := range missing {
		items = append(items, fmt.Sprintf("%s '%s' %s", item.Kind, item.Ref, item.Reason))
	}
	return strings.Join(items, ", ")
}
