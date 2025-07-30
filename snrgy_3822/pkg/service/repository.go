package service

import (
	"context"
	"errors"
	"fmt"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	helper_modfile "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/modfile"
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	models_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/module"
	models_repo "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/repository"
	models_service "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/slog_attr"
	"reflect"
	"slices"
	"strings"
)

func (s *Service) RepoModules(ctx context.Context) ([]models_service.RepoModule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	repos, err := s.reposHdl.Repositories(ctx)
	if err != nil {
		return nil, err
	}
	repoMods, err := s.reposHdl.Modules(ctx)
	if err != nil {
		return nil, err
	}
	installedMods, err := s.modsHdl.Modules(ctx, models_module.ModuleFilter{})
	if err != nil {
		return nil, err
	}
	return s.repoModules(repos, repoMods, installedMods)
}

func (s *Service) repoModules(repos []models_repo.Repository, repoMods []models_repo.Module, installedMods []models_module.ModuleAbbreviated) ([]models_service.RepoModule, error) {
	reposTree := buildReposTree(repos)
	installedModsMap := helper_slices.SliceToMap(installedMods, func(item models_module.ModuleAbbreviated) string {
		return item.ID
	})
	var repoModules []models_service.RepoModule
	for id, sources := range buildRepoModsTree(repoMods) {
		repoModule := models_service.RepoModule{ID: id}
		if variant, ok := installedModsMap[id]; ok {
			repoModule.Installed = &models_service.ModuleVariant{
				Source:  variant.Source,
				Channel: variant.Channel,
				Version: variant.Version,
			}
		}
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
			logger.Error("invalid repository module", slog_attr.IDKey, id, slog_attr.ErrorKey, fErr)
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

func (s *Service) selectRepoModules(ctx context.Context, reqItems []models_repo.ModuleBase) (map[string]modWrapper, error) {
	reqItemMap := make(map[string]models_repo.ModuleBase)
	for _, item := range reqItems {
		tmp, ok := reqItemMap[item.ID]
		if ok {
			if !equalMods(tmp.ID, tmp.Source, tmp.Channel, "", item.ID, item.Source, item.Channel, "") {
				return nil, fmt.Errorf("duplicate entry for %s", item.ID)
			}
			continue
		}
		reqItemMap[item.ID] = item
	}
	mods := make(map[string]modWrapper)
	for id, reqItem := range reqItemMap {
		modFS, err := s.reposHdl.ModuleFS(ctx, id, reqItem.Source, reqItem.Channel)
		if err != nil {
			return nil, err
		}
		mod, err := helper_modfile.GetModule(modFS)
		if err != nil {
			return nil, err
		}
		if _, ok := mods[mod.ID]; !ok {
			mods[mod.ID] = modWrapper{
				Mod:     mod,
				FS:      modFS,
				Source:  reqItem.Source,
				Channel: reqItem.Channel,
			}
		}
	}
	modRepos, err := s.reposHdl.Repositories(ctx)
	if err != nil {
		return nil, err
	}
	deps := make(map[string]modWrapper)
	highestPrioRepo := helper_slices.SelectByPriority(modRepos, func(item models_repo.Repository, lastPrio int) (int, bool) {
		return item.Priority, item.Priority >= lastPrio
	})
	highestPrioChannel := helper_slices.SelectByPriority(highestPrioRepo.Channels, func(item models_repo.Channel, lastPrio int) (int, bool) {
		return item.Priority, item.Priority >= lastPrio
	})
	for _, wrapper := range mods {
		if wrapper.Source == highestPrioRepo.Source {
			if err = s.addRepoModDepsToMap(ctx, wrapper.Mod, wrapper.Source, highestPrioChannel.Name, deps); err != nil {
				return nil, err
			}
		}
	}
	modReposMap := helper_slices.SliceToMap(modRepos, func(item models_repo.Repository) string {
		return item.Source
	})
	for _, wrapper := range mods {
		repo, ok := modReposMap[wrapper.Source]
		if !ok {
			return nil, errors.New("source not found")
		}
		highestPrioChannel = helper_slices.SelectByPriority(repo.Channels, func(item models_repo.Channel, lastPrio int) (int, bool) {
			return item.Priority, item.Priority >= lastPrio
		})
		if err = s.addRepoModDepsToMap(ctx, wrapper.Mod, wrapper.Source, highestPrioChannel.Name, deps); err != nil {
			return nil, err
		}
	}
	for id, wrapper := range deps {
		if _, ok := mods[id]; !ok {
			mods[id] = wrapper
		}
	}
	return mods, nil
}

func (s *Service) addRepoModDepsToMap(ctx context.Context, mod module_lib.Module, source, channel string, deps map[string]modWrapper) error {
	for depID := range mod.Dependencies {
		if _, ok := deps[depID]; !ok {
			depFS, err := s.reposHdl.ModuleFS(ctx, depID, source, channel)
			if err != nil {
				return err
			}
			dep, err := helper_modfile.GetModule(depFS)
			if err != nil {
				return err
			}
			deps[depID] = modWrapper{
				Mod:     dep,
				FS:      depFS,
				Source:  source,
				Channel: channel,
			}
			err = s.addRepoModDepsToMap(ctx, dep, source, channel, deps)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func buildRepoModsTree(repoMods []models_repo.Module) map[string]map[string]map[string]repoModAbbreviated {
	repoModsTree := make(map[string]map[string]map[string]repoModAbbreviated) // {id:{source:{channel:repoModAbbreviated}}}
	for _, repoMod := range repoMods {
		sources, ok := repoModsTree[repoMod.ID]
		if !ok {
			sources = make(map[string]map[string]repoModAbbreviated)
			repoModsTree[repoMod.ID] = sources
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
