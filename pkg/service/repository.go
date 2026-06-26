package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_modfile "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/modfile"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (s *Service) RefreshRepositories(ctx context.Context, filter lib_models.RepositoriesRefreshFilter) (lib_models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJob, ok := s.jobsHandler.CurrentSlotJob(repositoryJobSlotNum)
	if ok {
		return lib_models.Job{}, lib_errors.New[lib_errors.ErrActiveJob](activeJobErrMsg(currentJob))
	}
	s.changeRequest = nil
	job, err := s.jobsHandler.CreateSlotJob(repositoryJobSlotNum, "refresh repositories")
	if err != nil {
		return lib_models.Job{}, err
	}
	go func() {
		jobResult := lib_models.RepositoryJobResult{
			JobResult: lib_models.JobResult{
				JobId: job.Id,
			},
		}
		defer func() {
			if st := recover(); st != nil {
				jobResult.ErrorResult = lib_models.NewErrorResult(fmt.Sprintf("%v", st))
				logger.ErrorContext(
					ctx,
					"refresh repositories",
					slog_keys.JobId, job.Id,
					slog_keys.Error, "panic",
					slog_keys.StackTrace, st,
				)
			}
			s.setRefreshRepositoriesJobResult(job.Id, jobResult)
			job.Done()
			logJobDone(ctx, job)
		}()
		logJobStart(ctx, job)
		jobResult.Results, err = s.repositoriesHandler.RefreshRepositories(job.Context(), filter)
		if err != nil {
			jobResult.ErrorResult = lib_models.NewErrorResult(err.Error())
		}
		for _, res := range jobResult.Results {
			if res.HasError {
				jobResult.ResultsErrNum++
			}
		}
	}()
	return lib_models.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) GetRepositories(ctx context.Context) ([]lib_models.Repository, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	currentJob, ok := s.jobsHandler.CurrentSlotJob(repositoryJobSlotNum)
	if ok {
		return nil, lib_errors.New[lib_errors.ErrActiveJob](activeJobErrMsg(currentJob))
	}
	return s.repositoriesHandler.GetRepositories(ctx)
}

func (s *Service) CreateRepository(ctx context.Context, repositoryType string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJob, ok := s.jobsHandler.CurrentSlotJob(repositoryJobSlotNum)
	if ok {
		return lib_errors.New[lib_errors.ErrActiveJob](activeJobErrMsg(currentJob))
	}
	return s.repositoriesHandler.CreateRepository(ctx, repositoryType, data)
}

func (s *Service) DeleteRepository(ctx context.Context, source string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJob, ok := s.jobsHandler.CurrentSlotJob(repositoryJobSlotNum)
	if ok {
		return lib_errors.New[lib_errors.ErrActiveJob](activeJobErrMsg(currentJob))
	}
	return s.repositoriesHandler.DeleteRepository(ctx, source)
}

func (s *Service) GetRepositoryModules(ctx context.Context, filter lib_models.RepoModulesFilter) ([]lib_models.RepoModule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	currentJob, ok := s.jobsHandler.CurrentSlotJob(repositoryJobSlotNum)
	if ok {
		return nil, lib_errors.New[lib_errors.ErrActiveJob](activeJobErrMsg(currentJob))
	}
	repos, err := s.repositoriesHandler.GetRepositories(ctx)
	if err != nil {
		return nil, err
	}
	repoModules, err := s.repositoriesHandler.GetModules(ctx, pkg_models.RepositoryModulesFilter{
		Ids:     filter.Ids,
		Name:    filter.Name,
		Sources: newSourceFilters(filter.Repositories),
	})
	if err != nil {
		return nil, err
	}
	mergedRepoModules, err := s.mergeRepoModules(
		ctx,
		repos,
		repoModules,
	)
	if err != nil {
		return nil, err
	}
	installedMods, err := s.modulesHandler.GetModules(ctx, pkg_models.ModulesFilterWithName{}, false)
	if err != nil {
		return nil, err
	}
	return handleInstalledMods(mergedRepoModules, installedMods, filter.Installed, filter.UpdateAvailable), nil
}

func (s *Service) mergeRepoModules(ctx context.Context, repos []lib_models.Repository, repoMods []pkg_models.RepositoryModule) ([]lib_models.RepoModule, error) {
	reposTree := buildReposTree(repos)
	var repoModules []lib_models.RepoModule
	for id, sources := range buildRepoModsTree(repoMods) {
		repoModule := lib_models.RepoModule{Id: id}
		var fErr error
		for source, channels := range sources {
			repo, ok := reposTree[source]
			if !ok {
				fErr = fmt.Errorf("repository '%s' not found", source)
				break
			}
			repoModuleVariant := lib_models.RepoModuleVariant{
				Source:   source,
				Priority: repo.Priority,
			}
			for channel, repoMod := range channels {
				channelPrio, ok := repo.Channels[channel]
				if !ok {
					fErr = fmt.Errorf("channel '%s' not found", channel)
					break
				}
				repoModuleVariant.Channels = append(repoModuleVariant.Channels, lib_models.RepoModuleVariantChannel{
					Name:     channel,
					Priority: channelPrio,
					Version:  repoMod.Version,
				})
			}
			slices.SortStableFunc(repoModuleVariant.Channels, func(a, b lib_models.RepoModuleVariantChannel) int {
				return b.Priority - a.Priority
			})
			if len(repoModuleVariant.Channels) == 0 {
				fErr = fmt.Errorf("no channels for '%s'", source)
				break
			}
			repoMod := channels[repoModuleVariant.Channels[0].Name]
			repoModule.Name = repoMod.Name
			repoModule.Desc = repoMod.Desc
			repoModule.Version = repoMod.Version
			repoModule.RepositoryVariants = append(repoModule.RepositoryVariants, repoModuleVariant)
		}
		if fErr != nil {
			logger.ErrorContext(ctx, "invalid repository module", slog_keys.ModuleId, id, slog_keys.Error, fErr)
			continue
		}
		slices.SortStableFunc(repoModule.RepositoryVariants, func(a, b lib_models.RepoModuleVariant) int {
			return b.Priority - a.Priority
		})
		repoModules = append(repoModules, repoModule)
	}
	slices.SortStableFunc(repoModules, func(a, b lib_models.RepoModule) int {
		return strings.Compare(a.Name, b.Name)
	})
	return repoModules, nil
}

func (s *Service) selectRepoModules(ctx context.Context, reqItems []lib_models.ChangeRequestItem, installedModsMap map[string]pkg_models.Module) (map[string]modWrapper, error) {
	// get module filesystem and modfile
	mods := make(map[string]modWrapper)
	for _, item := range reqItems {
		if item.Remove {
			continue
		}
		var installedVer string
		if item.Update {
			installedMod, ok := installedModsMap[item.Id]
			if !ok {
				continue
			}
			item.Source = installedMod.Source
			item.Channel = installedMod.Channel
			installedVer = installedMod.Version
		}
		modFS, err := s.repositoriesHandler.GetModuleFS(ctx, item.Id, item.Source, item.Channel)
		if err != nil {
			return nil, err
		}
		mod, err := helper_modfile.GetModule(modFS)
		if err != nil {
			return nil, err
		}
		if item.Update && (mod.Version == installedVer) {
			continue
		}
		if _, ok := mods[mod.ID]; !ok {
			mods[mod.ID] = modWrapper{
				Mod:     mod,
				FS:      modFS,
				Source:  item.Source,
				Channel: item.Channel,
			}
		}
	}
	// get repo with the highest priority
	modRepos, err := s.repositoriesHandler.GetRepositories(ctx)
	if err != nil {
		return nil, err
	}
	highestPrioRepo := selectByPriority(modRepos, func(item lib_models.Repository, lastPrio int) (int, bool) {
		return item.Priority, item.Priority >= lastPrio
	})
	highestPrioChannel := selectByPriority(highestPrioRepo.Channels, func(item lib_models.RepositoryChannel, lastPrio int) (int, bool) {
		return item.Priority, item.Priority >= lastPrio
	})
	deps := make(map[string]modWrapper)
	// select dependencies from main source and channel
	for _, wrapper := range mods {
		if err := s.addRepoModDepsToMap(ctx, wrapper.Mod, highestPrioRepo.Source, highestPrioChannel.Name, deps, true); err != nil {
			return nil, err
		}
	}
	// select dependencies only available in origin repo and channel
	for _, wrapper := range mods {
		if err := s.addRepoModDepsToMap(ctx, wrapper.Mod, wrapper.Source, wrapper.Channel, deps, false); err != nil {
			return nil, err
		}
	}
	// add dependencies to module selection
	for id, wrapper := range deps {
		if _, ok := mods[id]; !ok {
			mods[id] = wrapper
		}
	}
	return mods, nil
}

func (s *Service) addRepoModDepsToMap(
	ctx context.Context,
	mod external_models.ModuleLibModule,
	source string,
	channel string,
	deps map[string]modWrapper,
	skipNotFound bool,
) error {
	for depId := range mod.Dependencies {
		if _, ok := deps[depId]; !ok {
			depFS, err := s.repositoriesHandler.GetModuleFS(ctx, depId, source, channel)
			if err != nil {
				if lib_errors.IsOf[lib_errors.ErrNotFound](err) && skipNotFound {
					continue
				}
				return err
			}
			dep, err := helper_modfile.GetModule(depFS)
			if err != nil {
				return err
			}
			deps[depId] = modWrapper{
				Mod:     dep,
				FS:      depFS,
				Source:  source,
				Channel: channel,
			}
			err = s.addRepoModDepsToMap(ctx, dep, source, channel, deps, skipNotFound)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func newSourceFilters(repoFilters []lib_models.RepoModuleRepositoriesFilter) []pkg_models.RepositorySourceFilter {
	var sourcesFilter []pkg_models.RepositorySourceFilter
	for _, repoFilter := range repoFilters {
		sourcesFilter = append(sourcesFilter, pkg_models.RepositorySourceFilter{
			Name:     repoFilter.Source,
			Channels: repoFilter.Channels,
		})
	}
	return sourcesFilter
}

func buildRepoModsTree(repoMods []pkg_models.RepositoryModule) map[string]map[string]map[string]repoModAbbreviated {
	repoModsTree := make(map[string]map[string]map[string]repoModAbbreviated) // {id:{source:{channel:repoModAbbreviated}}}
	for _, repoMod := range repoMods {
		sources, ok := repoModsTree[repoMod.Id]
		if !ok {
			sources = make(map[string]map[string]repoModAbbreviated)
			repoModsTree[repoMod.Id] = sources
		}
		channels, ok := sources[repoMod.Source]
		if !ok {
			channels = make(map[string]repoModAbbreviated)
			sources[repoMod.Source] = channels
		}
		channels[repoMod.Channel] = repoModAbbreviated{
			Name:    repoMod.Name,
			Desc:    repoMod.Desc,
			Version: repoMod.Version,
		}
	}
	return repoModsTree
}

func buildReposTree(repos []lib_models.Repository) map[string]repoAbbreviated {
	reposTree := make(map[string]repoAbbreviated) // {source:repoAbbreviated}
	for _, repo := range repos {
		channels := make(map[string]int)
		for _, channel := range repo.Channels {
			channels[channel.Name] = channel.Priority
		}
		reposTree[repo.Source] = repoAbbreviated{
			Priority: repo.Priority,
			Channels: channels,
		}
	}
	return reposTree
}

func handleInstalledMods(mods []lib_models.RepoModule, installedMods map[string]pkg_models.Module, filterInstalled, filterUpdateAvailable bool) []lib_models.RepoModule {
	if len(installedMods) == 0 {
		if filterInstalled || filterUpdateAvailable {
			return nil
		}
		return mods
	}
	var tmp []lib_models.RepoModule
	for _, mod := range mods {
		variant, ok := installedMods[mod.Id]
		if ok {
			nextVersion := getNextVersion(variant, mod.RepositoryVariants)
			if filterUpdateAvailable && nextVersion == "" {
				continue
			}
			mod.IsInstalled = true
			mod.InstalledVariant = lib_models.InstalledModuleVariant{
				ModuleVariant: lib_models.ModuleVariant{
					Source:  variant.Source,
					Channel: variant.Channel,
					Version: variant.Version,
				},
				NextVersion: nextVersion,
			}
		} else {
			if filterInstalled {
				continue
			}
			if filterUpdateAvailable {
				continue
			}
		}
		tmp = append(tmp, mod)
	}
	return tmp
}

func getNextVersion(installed pkg_models.Module, repos []lib_models.RepoModuleVariant) string {
	for _, repo := range repos {
		if repo.Source == installed.Source {
			for _, channel := range repo.Channels {
				if channel.Name == installed.Channel {
					if channel.Version != installed.Version {
						return channel.Version
					}
				}
			}
		}
	}
	return ""
}

func selectByPriority[S ~[]E, E any](sl S, comp func(item E, lastPrio int) (int, bool)) E {
	var lastPrio int
	var candidate E
	for i := 0; i < len(sl); i++ {
		prio, ok := comp(sl[i], lastPrio)
		if i == 0 || ok {
			lastPrio = prio
			candidate = sl[i]
		}
	}
	return candidate
}
