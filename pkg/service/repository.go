package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	lib_models_results "github.com/SENERGY-Platform/mgw-module-manager/lib/models/results"
	lib_models_service "github.com/SENERGY-Platform/mgw-module-manager/lib/models/service"
	helper_modfile "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/modfile"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_modules "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/modules"
	models_handler_repositories "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/repositories"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/slog_attr"
)

func (s *Service) RefreshRepositories(_ context.Context) (lib_models_service.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.jobsHandler.CurrentSlotJob(repositoryJobSlotNum)
	if ok {
		return lib_models_service.Job{}, errors.New("active job") // TODO
	}
	s.changeRequest = nil
	job, err := s.jobsHandler.CreateSlotJob(repositoryJobSlotNum, "refresh repositories")
	if err != nil {
		return lib_models_service.Job{}, err
	}
	go func() {
		defer job.Done()
		jobResult := lib_models_service.JobResult{JobId: job.Id}
		defer func() {
			if err := recover(); err != nil {
				jobResult.ErrorResult = lib_models_results.NewErrorResult(fmt.Sprintf("panic: %v", err))
				s.setRefreshRepositoriesJobResult(job.Id, jobResult)
			}
		}()
		err = s.repositoriesHandler.RefreshRepositories(job.Context())
		if err != nil {
			jobResult.ErrorResult = lib_models_results.NewErrorResult(err.Error())
		}
		s.setRefreshRepositoriesJobResult(job.Id, jobResult)
	}()
	return lib_models_service.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) RepoModules(ctx context.Context, filter lib_models_service.RepoModulesFilter) ([]lib_models_service.RepoModule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.jobsHandler.CurrentSlotJob(repositoryJobSlotNum)
	if ok {
		return nil, errors.New("active job") // TODO
	}
	repos, err := s.repositoriesHandler.Repositories(ctx)
	if err != nil {
		return nil, err
	}
	repoMods, err := s.repositoriesHandler.Modules(ctx, models_handler_repositories.ModulesFilter{
		Ids:     filter.Ids,
		Name:    filter.Name,
		Sources: newSourceFilters(filter.Repositories),
	})
	if err != nil {
		return nil, err
	}
	mods, err := s.repoModules(repos, repoMods)
	if err != nil {
		return nil, err
	}
	installedMods, err := s.modulesHandler.Modules(ctx, models_handler_modules.ModuleFilter{})
	if err != nil {
		return nil, err
	}
	return handleInstalledMods(mods, installedMods, filter.Installed, filter.UpdateAvailable), nil
}

func (s *Service) repoModules(repos []models_handler_repositories.Repository, repoMods []models_handler_repositories.Module) ([]lib_models_service.RepoModule, error) {
	reposTree := buildReposTree(repos)
	var repoModules []lib_models_service.RepoModule
	for id, sources := range buildRepoModsTree(repoMods) {
		repoModule := lib_models_service.RepoModule{Id: id}
		var fErr error
		for source, channels := range sources {
			repo, ok := reposTree[source]
			if !ok {
				fErr = fmt.Errorf("repository '%s' not found", source)
				break
			}
			repository := lib_models_service.Repository{
				Source:   source,
				Priority: repo.Priority,
			}
			for channel, repoMod := range channels {
				channelPrio, ok := repo.Channels[channel]
				if !ok {
					fErr = fmt.Errorf("channel '%s' not found", channel)
					break
				}
				repository.Channels = append(repository.Channels, lib_models_service.Channel{
					Name:     channel,
					Priority: channelPrio,
					Version:  repoMod.Version,
				})
			}
			slices.SortStableFunc(repository.Channels, func(a, b lib_models_service.Channel) int {
				return b.Priority - a.Priority
			})
			if len(repository.Channels) == 0 {
				fErr = fmt.Errorf("no channels for '%s'", source)
				break
			}
			repoMod := channels[repository.Channels[0].Name]
			repoModule.Name = repoMod.Name
			repoModule.Desc = repoMod.Desc
			repoModule.Version = repoMod.Version
			repoModule.Repositories = append(repoModule.Repositories, repository)
		}
		if fErr != nil {
			logger.Error("invalid repository module", slog_attr.IdKey, id, slog_attr.ErrorKey, fErr)
			continue
		}
		slices.SortStableFunc(repoModule.Repositories, func(a, b lib_models_service.Repository) int {
			return b.Priority - a.Priority
		})
		repoModules = append(repoModules, repoModule)
	}
	slices.SortStableFunc(repoModules, func(a, b lib_models_service.RepoModule) int {
		return strings.Compare(a.Name, b.Name)
	})
	return repoModules, nil
}

func (s *Service) selectRepoModules(ctx context.Context, reqItems []lib_models_service.ChangeRequestItem, installedModsMap map[string]models_handler_modules.Module) (map[string]modWrapper, error) {
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
		modFS, err := s.repositoriesHandler.ModuleFS(ctx, item.Id, item.Source, item.Channel)
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
	modRepos, err := s.repositoriesHandler.Repositories(ctx)
	if err != nil {
		return nil, err
	}
	highestPrioRepo := selectByPriority(modRepos, func(item models_handler_repositories.Repository, lastPrio int) (int, bool) {
		return item.Priority, item.Priority >= lastPrio
	})
	highestPrioChannel := selectByPriority(highestPrioRepo.Channels, func(item models_handler_repositories.Channel, lastPrio int) (int, bool) {
		return item.Priority, item.Priority >= lastPrio
	})
	deps := make(map[string]modWrapper)
	// select dependencies from main source and channel
	for _, wrapper := range mods {
		if err = s.addRepoModDepsToMap(ctx, wrapper.Mod, highestPrioRepo.Source, highestPrioChannel.Name, deps, true); err != nil {
			return nil, err
		}
	}
	// select dependencies only available in origin repo and channel
	for _, wrapper := range mods {
		if err = s.addRepoModDepsToMap(ctx, wrapper.Mod, wrapper.Source, wrapper.Channel, deps, false); err != nil {
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

func (s *Service) addRepoModDepsToMap(ctx context.Context, mod models_external.ModuleLibModule, source, channel string, deps map[string]modWrapper, skipNotFound bool) error {
	for depId := range mod.Dependencies {
		if _, ok := deps[depId]; !ok {
			depFS, err := s.repositoriesHandler.ModuleFS(ctx, depId, source, channel)
			if err != nil {
				if errors.Is(err, models_error.NotFoundErr) && skipNotFound {
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

func newSourceFilters(repoFilters []lib_models_service.RepositoryFilter) []models_handler_repositories.SourceFilter {
	var sourcesFilter []models_handler_repositories.SourceFilter
	for _, repoFilter := range repoFilters {
		sourcesFilter = append(sourcesFilter, models_handler_repositories.SourceFilter{
			Name:     repoFilter.Source,
			Channels: repoFilter.Channels,
		})
	}
	return sourcesFilter
}

func buildRepoModsTree(repoMods []models_handler_repositories.Module) map[string]map[string]map[string]repoModAbbreviated {
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

func buildReposTree(repos []models_handler_repositories.Repository) map[string]repoAbbreviated {
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

func handleInstalledMods(mods []lib_models_service.RepoModule, installedMods map[string]models_handler_modules.Module, filterInstalled, filterUpdateAvailable bool) []lib_models_service.RepoModule {
	if len(installedMods) == 0 {
		return mods
	}
	var tmp []lib_models_service.RepoModule
	for _, mod := range mods {
		variant, ok := installedMods[mod.Id]
		if ok {
			nextVersion := getNextVersion(variant, mod.Repositories)
			if filterUpdateAvailable && nextVersion == "" {
				continue
			}
			mod.Installed = &lib_models_service.InstalledModuleVariant{
				ModuleVariant: lib_models_service.ModuleVariant{
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

func getNextVersion(installed models_handler_modules.Module, repos []lib_models_service.Repository) string {
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
