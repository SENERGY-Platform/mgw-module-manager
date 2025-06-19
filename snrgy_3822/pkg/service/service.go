package service

import (
	"context"
	"errors"
	"fmt"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/modfile_util"
	models_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/module"
	models_repo "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/repository"
	models_service "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/util"
	"reflect"
	"slices"
	"strings"
)

type Service struct {
	modReposHdl ModuleReposHandler
	modHdl      ModuleHandler
	logger      Logger
}

func New(modReposHdl ModuleReposHandler, modHdl ModuleHandler, logger Logger) *Service {
	return &Service{
		modReposHdl: modReposHdl,
		modHdl:      modHdl,
		logger:      logger,
	}
}

func (s *Service) RepoModules(ctx context.Context) ([]models_service.RepoModule, error) {
	repos, err := s.modReposHdl.Repositories(ctx)
	if err != nil {
		return nil, err
	}
	repoMods, err := s.modReposHdl.Modules(ctx)
	if err != nil {
		return nil, err
	}
	installedMods, err := s.modHdl.Modules(ctx, models_module.ModuleFilter{})
	if err != nil {
		return nil, err
	}
	return s.repoModules(repos, repoMods, installedMods)
}

func (s *Service) repoModules(repos []models_repo.Repository, repoMods []models_repo.Module, installedMods []models_module.ModuleAbbreviated) ([]models_service.RepoModule, error) {
	reposMap := util.SliceToMap(repos, func(item models_repo.Repository) string {
		return item.Source
	})
	installedModsMap := util.SliceToMap(installedMods, func(item models_module.ModuleAbbreviated) string {
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
		var defaultRepo bool
		var fErr error
		for source, channels := range sources {
			repo, ok := reposMap[source]
			if !ok {
				fErr = fmt.Errorf("repo '%s' not found", source)
				break
			}
			if repo.Default {
				defaultRepo = true
			}
			repository := models_service.Repository{
				Source:  source,
				Default: repo.Default,
			}
			var defaultChannel bool
			for channel, repoMod := range channels {
				if channel == repo.DefaultChannel {
					defaultChannel = true
					repoModule.Name = repoMod.Name
					repoModule.Desc = repoMod.Desc
					repoModule.Version = repoMod.Version
				}
				repository.Channels = append(repository.Channels, models_service.Channel{
					Name:    channel,
					Default: channel == repo.DefaultChannel,
					Version: repoMod.Version,
				})
			}
			slices.SortStableFunc(repository.Channels, func(a, b models_service.Channel) int {
				return strings.Compare(a.Name, b.Name)
			})
			if !defaultChannel {
				var tmpChannels []models_service.Channel
				for i, channel := range repository.Channels {
					if i == 0 {
						channel.Default = true
						repoMod := channels[channel.Name]
						repoModule.Name = repoMod.Name
						repoModule.Desc = repoMod.Desc
						repoModule.Version = repoMod.Version
					}
					tmpChannels = append(tmpChannels, channel)
				}
				repository.Channels = tmpChannels
			}
			repoModule.Repositories = append(repoModule.Repositories, repository)
		}
		if fErr != nil {
			s.logger.Error(fErr)
			continue
		}
		slices.SortStableFunc(repoModule.Repositories, func(a, b models_service.Repository) int {
			return strings.Compare(a.Source, b.Source)
		})
		if !defaultRepo {
			var tmpRepos []models_service.Repository
			for i, repository := range repoModule.Repositories {
				if i == 0 {
					repository.Default = true
				}
				tmpRepos = append(tmpRepos, repository)
			}
			repoModule.Repositories = tmpRepos
		}
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
			if !reflect.DeepEqual(tmp, item) {
				return nil, fmt.Errorf("duplicate entry for %s", item.ID)
			}
			continue
		}
		reqItemMap[item.ID] = item
	}
	mods := make(map[string]modWrapper)
	for id, reqItem := range reqItemMap {
		modFS, err := s.modReposHdl.ModuleFS(ctx, id, reqItem.Source, reqItem.Channel)
		if err != nil {
			return nil, err
		}
		mod, err := modfile_util.GetModule(modFS)
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
	modRepos, err := s.modReposHdl.Repositories(ctx)
	if err != nil {
		return nil, err
	}
	deps := make(map[string]modWrapper)
	for _, repo := range modRepos {
		if repo.Default {
			channel := selectChannel(repo)
			for _, wrapper := range mods {
				if wrapper.Source == repo.Source {
					if err = s.addRepoModDepsToMap(ctx, wrapper.Mod, wrapper.Source, channel, deps); err != nil {
						return nil, err
					}
				}
			}
			break
		}
	}
	modReposMap := util.SliceToMap(modRepos, func(item models_repo.Repository) string {
		return item.Source
	})
	for _, wrapper := range mods {
		repo, ok := modReposMap[wrapper.Source]
		if !ok {
			return nil, errors.New("source not found")
		}
		if err = s.addRepoModDepsToMap(ctx, wrapper.Mod, wrapper.Source, selectChannel(repo), deps); err != nil {
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
			depFS, err := s.modReposHdl.ModuleFS(ctx, depID, source, channel)
			if err != nil {
				return err
			}
			dep, err := modfile_util.GetModule(depFS)
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

func selectChannel(repo models_repo.Repository) string {
	channel := repo.DefaultChannel
	if channel == "" && len(repo.Channels) > 0 {
		channel = repo.Channels[0]
	}
	return channel
}

func buildRepoModsTree(repoMods []models_repo.Module) map[string]map[string]map[string]repoModAbbreviated {
	repoModsTree := make(map[string]map[string]map[string]repoModAbbreviated) // {id:{source:{channel:models_repo.Module}}}
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
