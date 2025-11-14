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
	"fmt"
	"maps"
	"reflect"

	helper_slices "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/module"
	models_repo "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/repository"
	models_service "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
)

func (s *Service) Modules(ctx context.Context, filter models_module.ModuleFilter) ([]models_module.Module, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.modsHdl.Modules(ctx, filter)
}

func (s *Service) Module(ctx context.Context, id string) (models_module.Module, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.modsHdl.Module(ctx, id)
}

func (s *Service) ModulesChangeRequest(_ context.Context) (models_service.ModulesChangeRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.changeReq == nil {
		return models_service.ModulesChangeRequest{}, models_error.NotFoundErr
	}
	return transformModulesChangeRequest(*s.changeReq), nil
}

func (s *Service) NewModulesChangeRequest(ctx context.Context, reqItems []models_service.ChangeRequestItem) (models_service.ModulesChangeRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	reqItems, err := validateReqItems(reqItems)
	if err != nil {
		return models_service.ModulesChangeRequest{}, err
	}
	installedMods, err := s.modsHdl.Modules(ctx, models_module.ModuleFilter{})
	if err != nil {
		return models_service.ModulesChangeRequest{}, err
	}
	installedModsMap := maps.Collect(helper_slices.AllFunc(installedMods, func(item models_module.Module) string {
		return item.ID
	}))
	selectedRepoMods, err := s.selectRepoModules(ctx, reqItems, installedModsMap)
	if err != nil {
		return models_service.ModulesChangeRequest{}, err
	}
	var toRemoveMods []string
	for _, item := range reqItems {
		if item.Remove {
			toRemoveMods = append(toRemoveMods, item.ID)
		}
	}
	changeRequest := newModulesChangeRequest(selectedRepoMods, installedModsMap, toRemoveMods)
	s.changeReq = &changeRequest
	return transformModulesChangeRequest(changeRequest), nil
}

func (s *Service) ExecModulesChangeRequest(ctx context.Context) (models_service.ModulesChangeReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.changeReq == nil {
		return models_service.ModulesChangeReport{}, models_error.NotFoundErr
	}
	defer func() {
		s.changeReq = nil
	}()
	var success []models_service.ChangeReportItem
	var failed []models_service.ChangeReportErrItem
	for _, id := range s.changeReq.Remove {
		cri := models_service.ChangeReportItem{
			ID:     id,
			Action: models_service.ChangeActionRemove,
		}
		err := s.modsHdl.Remove(ctx, id)
		if err != nil {
			failed = append(failed, models_service.ChangeReportErrItem{
				ChangeReportItem: cri,
				Error:            err.Error(),
			})
			continue
		}
		success = append(success, cri)
	}
	for _, repoMod := range s.changeReq.Install {
		cri := models_service.ChangeReportItem{
			ID:     repoMod.Mod.ID,
			Action: models_service.ChangeActionInstall,
		}
		err := s.modsHdl.Add(ctx, repoMod.Mod.ID, repoMod.Source, repoMod.Channel, repoMod.FS)
		if err != nil {
			failed = append(failed, models_service.ChangeReportErrItem{
				ChangeReportItem: cri,
				Error:            err.Error(),
			})
			continue
		}
		success = append(success, cri)
	}
	for _, item := range s.changeReq.Change {
		cri := models_service.ChangeReportItem{
			ID:     item.Next.Mod.ID,
			Action: models_service.ChangeActionChange,
		}
		err := s.modsHdl.Update(ctx, item.Next.Mod.ID, item.Next.Source, item.Next.Channel, item.Next.FS)
		if err != nil {
			failed = append(failed, models_service.ChangeReportErrItem{
				ChangeReportItem: cri,
				Error:            err.Error(),
			})
			continue
		}
		success = append(success, cri)
	}
	return models_service.ModulesChangeReport{
		Success: success,
		Failed:  failed,
		Created: helper_time.Now(),
	}, nil
}

func (s *Service) CancelModulesChangeRequest(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.changeReq == nil {
		return models_error.NotFoundErr
	}
	s.changeReq = nil
	return nil
}

func (s *Service) ModulesAvailableUpdates(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	changeRequest, err := s.newModulesUpdateAllChangeRequest(ctx)
	if err != nil {
		return 0, err
	}
	return len(changeRequest.Change), nil
}

func (s *Service) NewModulesUpdateAllChangeRequest(ctx context.Context) (models_service.ModulesChangeRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	changeRequest, err := s.newModulesUpdateAllChangeRequest(ctx)
	if err != nil {
		return models_service.ModulesChangeRequest{}, err
	}
	s.changeReq = &changeRequest
	return transformModulesChangeRequest(changeRequest), nil
}

func (s *Service) newModulesUpdateAllChangeRequest(ctx context.Context) (modulesChangeRequest, error) {
	installedMods, err := s.modsHdl.Modules(ctx, models_module.ModuleFilter{})
	if err != nil {
		return modulesChangeRequest{}, err
	}
	if len(installedMods) == 0 {
		return modulesChangeRequest{}, nil
	}
	installedModIDs := make([]string, len(installedMods))
	for i, mod := range installedMods {
		installedModIDs[i] = mod.ID
	}
	repoMods, err := s.reposHdl.Modules(ctx, models_repo.ModulesFilter{IDs: installedModIDs})
	if err != nil {
		return modulesChangeRequest{}, err
	}
	if len(repoMods) == 0 {
		return modulesChangeRequest{}, nil
	}
	installedModsMap := maps.Collect(helper_slices.AllFunc(installedMods, func(item models_module.Module) string {
		return item.ID
	}))
	var reqItems []models_service.ChangeRequestItem
	for _, repoMod := range repoMods {
		installedMod, ok := installedModsMap[repoMod.ID]
		if !ok {
			continue
		}
		if installedMod.Source == repoMod.Source && installedMod.Channel == repoMod.Channel && installedMod.Version != repoMod.Version {
			reqItems = append(reqItems, models_service.ChangeRequestItem{
				ID:      installedMod.ID,
				Source:  installedMod.Source,
				Channel: installedMod.Channel,
			})
		}
	}
	selectedRepoMods, err := s.selectRepoModules(ctx, reqItems, installedModsMap)
	if err != nil {
		return modulesChangeRequest{}, err
	}
	return newModulesChangeRequest(selectedRepoMods, installedModsMap, nil), nil
}

func validateReqItems(reqItems []models_service.ChangeRequestItem) ([]models_service.ChangeRequestItem, error) {
	var validatedItems []models_service.ChangeRequestItem
	tmpMap := make(map[string]models_service.ChangeRequestItem)
	for _, item := range reqItems {
		if (item.Update && item.Remove) || (!(item.Update || item.Remove) && item.Source+item.Channel == "") {
			return nil, fmt.Errorf("ivalid change request for '%s'", item.ID)
		}
		if tmp, ok := tmpMap[item.ID]; ok {
			if !reflect.DeepEqual(tmp, item) {
				return nil, fmt.Errorf("duplicate entry for '%s'", item.ID)
			}
			continue
		}
		tmpMap[item.ID] = item
		validatedItems = append(validatedItems, item)
	}
	return validatedItems, nil
}

func newModulesChangeRequest(selectedRepoMods map[string]modWrapper, installedModsMap map[string]models_module.Module, toRemoveMods []string) modulesChangeRequest {
	var install []modWrapper
	var change []changeItem
	var remove []string
	for id, repoMod := range selectedRepoMods {
		installedMod, ok := installedModsMap[id]
		if ok {
			if equalMods(repoMod.Mod.ID, repoMod.Source, repoMod.Channel, repoMod.Mod.Version, installedMod.ID, installedMod.Source, installedMod.Channel, installedMod.Version) {
				continue
			}
			change = append(change, changeItem{
				Previous: models_service.ModuleAbbreviated{
					ID:   installedMod.ID,
					Name: installedMod.Name,
					Desc: installedMod.Description,
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
		install = append(install, repoMod)
	}
	for _, id := range toRemoveMods {
		if _, ok := installedModsMap[id]; !ok {
			continue
		}
		if _, ok := selectedRepoMods[id]; ok {
			continue
		}
		remove = append(remove, id)
	}
	return modulesChangeRequest{
		Install: install,
		Change:  change,
		Remove:  remove,
		Created: helper_time.Now(),
	}
}

func transformModulesChangeRequest(req modulesChangeRequest) models_service.ModulesChangeRequest {
	mcr := models_service.ModulesChangeRequest{
		Created: req.Created,
	}
	for _, mod := range req.Install {
		mcr.Install = append(mcr.Install, modWrapperToServiceModuleAbbreviated(mod))
	}
	for _, item := range req.Change {
		mcr.Change = append(mcr.Change, [2]models_service.ModuleAbbreviated{
			item.Previous,
			modWrapperToServiceModuleAbbreviated(item.Next),
		})
	}
	if req.Remove != nil {
		mcr.Remove = req.Remove
	}
	return mcr
}

func modWrapperToServiceModuleAbbreviated(w modWrapper) models_service.ModuleAbbreviated {
	return models_service.ModuleAbbreviated{
		ID:   w.Mod.ID,
		Name: w.Mod.Name,
		Desc: w.Mod.Description,
		ModuleVariant: models_service.ModuleVariant{
			Source:  w.Source,
			Channel: w.Channel,
			Version: w.Mod.Version,
		},
	}
}
