package service

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	helper_modfile "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/modfile"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/module"
	models_repo "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/repository"
	models_service "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/slog_attr"
)

func (s *Service) RefreshRepositories(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.changeReq = nil
	return s.reposHdl.RefreshRepositories(ctx)
}

func (s *Service) RepoModules(ctx context.Context, filter models_service.RepoModulesFilter) ([]models_service.RepoModule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	repos, err := s.reposHdl.Repositories(ctx)
	if err != nil {
		return nil, err
	}
	repoMods, err := s.reposHdl.Modules(ctx, models_repo.ModulesFilter{
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
	installedMods, err := s.modsHdl.Modules(ctx, models_module.ModuleFilter{})
	if err != nil {
		return nil, err
	}
	return handleInstalledMods(mods, installedMods, filter.Installed, filter.UpdateAvailable), nil
}

func (s *Service) repoModules(repos []models_repo.Repository, repoMods []models_repo.Module) ([]models_service.RepoModule, error) {
	reposTree := buildReposTree(repos)
	var repoModules []models_service.RepoModule
	for id, sources := range buildRepoModsTree(repoMods) {
		repoModule := models_service.RepoModule{Id: id}
		var fErr error
		for source, channels := range sources {
			repo, ok := reposTree[source]
			if !ok {
				fErr = fmt.Errorf("repository '%s' not found", source)
				break
			}
			repository := models_service.Repository{
				Source:   source,
				Priority: repo.Priority,
			}
			for channel, repoMod := range channels {
				channelPrio, ok := repo.Channels[channel]
				if !ok {
					fErr = fmt.Errorf("channel '%s' not found", channel)
					break
				}
				repository.Channels = append(repository.Channels, models_service.Channel{
					Name:     channel,
					Priority: channelPrio,
					Version:  repoMod.Version,
				})
			}
			slices.SortStableFunc(repository.Channels, func(a, b models_service.Channel) int {
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
		slices.SortStableFunc(repoModule.Repositories, func(a, b models_service.Repository) int {
			return b.Priority - a.Priority
		})
		repoModules = append(repoModules, repoModule)
	}
	slices.SortStableFunc(repoModules, func(a, b models_service.RepoModule) int {
		return strings.Compare(a.Name, b.Name)
	})
	return repoModules, nil
}

func (s *Service) selectRepoModules(ctx context.Context, reqItems []models_service.ChangeRequestItem, installedModsMap map[string]models_module.Module) (map[string]modWrapper, error) {
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
		modFS, err := s.reposHdl.ModuleFS(ctx, item.Id, item.Source, item.Channel)
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
	modRepos, err := s.reposHdl.Repositories(ctx)
	if err != nil {
		return nil, err
	}
	highestPrioRepo := selectByPriority(modRepos, func(item models_repo.Repository, lastPrio int) (int, bool) {
		return item.Priority, item.Priority >= lastPrio
	})
	highestPrioChannel := selectByPriority(highestPrioRepo.Channels, func(item models_repo.Channel, lastPrio int) (int, bool) {
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

func (s *Service) addRepoModDepsToMap(ctx context.Context, mod module_lib.Module, source, channel string, deps map[string]modWrapper, skipNotFound bool) error {
	for depId := range mod.Dependencies {
		if _, ok := deps[depId]; !ok {
			depFS, err := s.reposHdl.ModuleFS(ctx, depId, source, channel)
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

func newSourceFilters(repoFilters []models_service.RepositoryFilter) []models_repo.SourceFilter {
	var sourcesFilter []models_repo.SourceFilter
	for _, repoFilter := range repoFilters {
		sourcesFilter = append(sourcesFilter, models_repo.SourceFilter{
			Name:     repoFilter.Source,
			Channels: repoFilter.Channels,
		})
	}
	return sourcesFilter
}

func buildRepoModsTree(repoMods []models_repo.Module) map[string]map[string]map[string]repoModAbbreviated {
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

func buildReposTree(repos []models_repo.Repository) map[string]repoAbbreviated {
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

func handleInstalledMods(mods []models_service.RepoModule, installedMods []models_module.Module, filterInstalled, filterUpdateAvailable bool) []models_service.RepoModule {
	if len(installedMods) == 0 {
		return mods
	}
	installedModsMap := maps.Collect(helper_slices.AllFunc(installedMods, func(item models_module.Module) string {
		return item.ID
	}))
	var tmp []models_service.RepoModule
	for _, mod := range mods {
		variant, ok := installedModsMap[mod.Id]
		if ok {
			nextVersion := getNextVersion(variant, mod.Repositories)
			if filterUpdateAvailable && nextVersion == "" {
				continue
			}
			mod.Installed = &models_service.InstalledModuleVariant{
				ModuleVariant: models_service.ModuleVariant{
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

func getNextVersion(installed models_module.Module, repos []models_service.Repository) string {
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
