package service

import (
	"context"
	"errors"
	"fmt"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/modfile_util"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"reflect"
)

type Service struct {
	modReposHandler ModuleReposHandler
}

func New(modReposHandler ModuleReposHandler) *Service {
	return &Service{modReposHandler: modReposHandler}
}

func (s *Service) selectRepoModules(ctx context.Context, reqItems []models.RepoModuleVariantBase) (map[string]modWrapper, error) {
	reqItemMap := make(map[string]models.RepoModuleVariantBase)
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
		modFS, err := s.modReposHandler.ModuleFS(ctx, id, reqItem.Source, reqItem.Channel)
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
	modRepos, err := s.modReposHandler.Repositories(ctx)
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
	for _, wrapper := range mods {
		repo, ok := modRepos[wrapper.Source]
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

func (s *Service) addRepoModDepsToMap(ctx context.Context, mod *module_lib.Module, source, channel string, deps map[string]modWrapper) error {
	for depID := range mod.Dependencies {
		if _, ok := deps[depID]; !ok {
			depFS, err := s.modReposHandler.ModuleFS(ctx, depID, source, channel)
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

func selectChannel(repo models.Repository) string {
	channel := repo.DefaultChannel
	if channel == "" && len(repo.Channels) > 0 {
		channel = repo.Channels[0]
	}
	return channel
}

/*

liste von modules zum installieren
	ziel
		autmatisieren was nicht explizit ausgewählt wird
		nutzer soll auch alles auswhählen können um die automatisierung zu umgehen
	erst die auswahl durchgehen und dann automatiseren
	mehr als ein repo
		zuerst module der default source nehmen
		abhängigkeiten werden aus der jeweiligen source genommen
			wie mit dublicaten umgehen?
				sources ohne default sortieren und dann einfach der reihe nach
	mehr als ein channel
		default channel hat vorrang
		gibt es keinen default wird erster in der liste genommen
		abhängigkeiten werden aus default oder erster in der liste genommen



*/
