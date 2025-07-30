/*
 * Copyright 2025 InfAI (CC SES)
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
	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/module"
	models_repo "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/repository"
	models_service "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
	"time"
)

func (s *Service) Modules(ctx context.Context, filter models_module.ModuleFilter) ([]models_module.ModuleAbbreviated, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.modsHdl.Modules(ctx, filter)
}

func (s *Service) Module(ctx context.Context, id string) (models_module.Module, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.modsHdl.Module(ctx, id)
}

func (s *Service) ModulesInstallRequest(_ context.Context) (models_service.ModulesInstallRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.installReq == nil {
		return models_service.ModulesInstallRequest{}, models_error.NotFoundErr
	}
	return transformModulesInstallRequest(*s.installReq), nil
}

func (s *Service) NewModulesInstallRequest(ctx context.Context, reqItems []models_repo.ModuleBase) (models_service.ModulesInstallRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	selectedRepoMods, err := s.selectRepoModules(ctx, reqItems)
	if err != nil {
		return models_service.ModulesInstallRequest{}, nil
	}
	installedMods, err := s.modsHdl.Modules(ctx, models_module.ModuleFilter{})
	if err != nil {
		return models_service.ModulesInstallRequest{}, nil
	}
	installRequest := newModulesInstallRequest(selectedRepoMods, installedMods)
	s.installReq = &installRequest
	return transformModulesInstallRequest(installRequest), nil
}

func (s *Service) ExecModulesInstallRequest(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.installReq == nil {
		return models_error.NotFoundErr
	}
	defer func() {
		s.installReq = nil
	}()
	for _, repoMod := range s.installReq.New {

	}
}

func (s *Service) CancelModulesInstallRequest(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.installReq == nil {
		return models_error.NotFoundErr
	}
	s.installReq = nil
	return nil
}

func newModulesInstallRequest(selectedRepoMods map[string]modWrapper, installedMods []models_module.ModuleAbbreviated) modulesInstallRequest {
	var newMods []modWrapper
	var stcMods []moduleSTC
	for id, repoMod := range selectedRepoMods {
		installedMod, ok := helper_slices.SelectByValue(installedMods, id, func(item models_module.ModuleAbbreviated) string {
			return item.ID
		})
		if ok {
			if equalMods(repoMod.Mod.ID, repoMod.Source, repoMod.Channel, repoMod.Mod.Version, installedMod.ID, installedMod.Source, installedMod.Channel, installedMod.Version) {
				continue
			}
			stcMods = append(stcMods, moduleSTC{
				Previous: models_service.ModuleAbbreviated{
					ID:   installedMod.ID,
					Name: installedMod.Name,
					Desc: installedMod.Desc,
					ModuleVariant: models_service.ModuleVariant{
						Source:  installedMod.Source,
						Channel: installedMod.Channel,
						Version: installedMod.Version,
					},
				},
				Next: repoMod,
			})
			continue
		}
		newMods = append(newMods, repoMod)
	}
	return modulesInstallRequest{
		New:     newMods,
		STC:     stcMods,
		Created: time.Now().UTC(),
	}
}

func transformModulesInstallRequest(req modulesInstallRequest) models_service.ModulesInstallRequest {
	newMods := make([]models_service.ModuleAbbreviated, len(req.New))
	for _, mod := range req.New {
		newMods = append(newMods, models_service.ModuleAbbreviated{
			ID:   mod.Mod.ID,
			Name: mod.Mod.Name,
			Desc: mod.Mod.Description,
			ModuleVariant: models_service.ModuleVariant{
				Source:  mod.Source,
				Channel: mod.Channel,
				Version: mod.Mod.Version,
			},
		})
	}
	stcMods := make([][2]models_service.ModuleAbbreviated, len(req.STC))
	for _, item := range req.STC {
		stcMods = append(stcMods, [2]models_service.ModuleAbbreviated{
			{
				ID:   item.Previous.ID,
				Name: item.Previous.Name,
				Desc: item.Previous.Desc,
				ModuleVariant: models_service.ModuleVariant{
					Source:  item.Previous.Source,
					Channel: item.Previous.Channel,
					Version: item.Previous.Version,
				},
			},
			{
				ID:   item.Next.Mod.ID,
				Name: item.Next.Mod.Name,
				Desc: item.Next.Mod.Description,
				ModuleVariant: models_service.ModuleVariant{
					Source:  item.Next.Source,
					Channel: item.Next.Channel,
					Version: item.Next.Mod.Version,
				},
			},
		})
	}
	return models_service.ModulesInstallRequest{
		New:     newMods,
		STC:     stcMods,
		Created: req.Created,
	}
}

//func newAndSTCMods(selectedRepoMods map[string]modWrapper, installedMods []models_module.ModuleAbbreviated) ([]models_service.ModuleAbbreviated, [][2]models_service.ModuleAbbreviated) {
//	var newMods []models_service.ModuleAbbreviated
//	var stcMods [][2]models_service.ModuleAbbreviated
//	for id, repoMod := range selectedRepoMods {
//		installedMod, ok := helper_slices.SelectByValue(installedMods, id, func(item models_module.ModuleAbbreviated) string {
//			return item.ID
//		})
//		if ok {
//			if equalMods(repoMod.Mod.ID, repoMod.Source, repoMod.Channel, repoMod.Mod.Version, installedMod.ID, installedMod.Source, installedMod.Channel, installedMod.Version) {
//				continue
//			}
//			stcMods = append(stcMods, [2]models_service.ModuleAbbreviated{
//				{
//					ID:   installedMod.ID,
//					Name: installedMod.Name,
//					Desc: installedMod.Desc,
//					ModuleVariant: models_service.ModuleVariant{
//						Source:  installedMod.Source,
//						Channel: installedMod.Channel,
//						Version: installedMod.Version,
//					},
//				},
//				{
//					ID:   repoMod.Mod.ID,
//					Name: repoMod.Mod.Name,
//					Desc: repoMod.Mod.Description,
//					ModuleVariant: models_service.ModuleVariant{
//						Source:  repoMod.Source,
//						Channel: repoMod.Channel,
//						Version: repoMod.Mod.Version,
//					},
//				},
//			})
//			continue
//		}
//		newMods = append(newMods, models_service.ModuleAbbreviated{
//			ID:   repoMod.Mod.ID,
//			Name: repoMod.Mod.Name,
//			Desc: repoMod.Mod.Description,
//			ModuleVariant: models_service.ModuleVariant{
//				Source:  repoMod.Source,
//				Channel: repoMod.Channel,
//				Version: repoMod.Mod.Version,
//			},
//		})
//	}
//	return newMods, stcMods
//}
