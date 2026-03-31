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
	"slices"

	helper_maps "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/maps"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_handler_deployment "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployment"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_repo "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/repository"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
	models_service "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
)

// TODO implement tags filter
func (s *Service) Modules(ctx context.Context, filter models_service.ModulesFilter) ([]models_service.ModuleReduced, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	modules, err := s.modulesHandler.Modules(ctx, models_handler_module.ModuleFilter{
		Ids:  filter.Ids,
		Name: filter.Name,
	})
	if err != nil {
		return nil, err
	}
	deployments, err := s.deploymentsHandler.GetDeploymentsReduced(ctx, models_handler_deployment.DeploymentsFilter{
		DeploymentsFilter: models_handler_storage.DeploymentsFilter{
			ModuleIds: slices.Collect(maps.Keys(modules)),
		},
	})
	if err != nil {
		return nil, err
	}
	deployments = helper_maps.CollectFunc(maps.Values(deployments), func(value models_handler_deployment.DeploymentReduced) string {
		return value.ModuleId
	}) // TODO sollten deployments generell per module ID zurückgegeben werden
	var result []models_service.ModuleReduced
	for moduleId, module := range modules {
		deployment, ok := deployments[moduleId]
		if ok {
			if filter.DeploymentEnabled < 0 && deployment.Enabled {
				continue
			}
			if filter.DeploymentEnabled > 0 && !deployment.Enabled {
				continue
			}
		}
		if filter.IsDeployed < 0 && ok {
			continue
		}
		if filter.IsDeployed > 0 && !ok {
			continue
		}
		if filter.DeploymentState > 0 && deployment.State != filter.DeploymentState {
			continue
		}
		if filter.Author != "" && module.Author != filter.Author {
			continue
		}
		result = append(result, models_service.ModuleReduced{
			Id:          moduleId,
			Source:      module.Source,
			Channel:     module.Channel,
			Version:     module.Version,
			Name:        module.Name,
			Description: module.Description,
			Tags:        slices.Collect(maps.Keys(module.Tags)),
			License:     module.License,
			Author:      module.Author,
			Deployment: models_service.DeploymentReduced{
				Id:            deployment.Id,
				ModuleSource:  deployment.ModuleSource,
				ModuleChannel: deployment.ModuleChannel,
				ModuleVersion: deployment.ModuleVersion,
				Enabled:       deployment.Enabled,
				Created:       deployment.Created,
				Updated:       deployment.Updated,
				State:         deployment.State,
			},
		})
	}
	return result, nil
}

// TODO include deployment
func (s *Service) Module(ctx context.Context, id string) (models_handler_module.Module, error) {
	//s.mu.RLock()
	//defer s.mu.RUnlock()
	//installedMod, err := s.modsHdl.Module(ctx, id)
	//if err != nil {
	//	return models_module.Module{}, err
	//}
	panic("implement me")
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
	installedMods, err := s.modulesHandler.Modules(ctx, models_handler_module.ModuleFilter{})
	if err != nil {
		return models_service.ModulesChangeRequest{}, err
	}
	selectedRepoMods, err := s.selectRepoModules(ctx, reqItems, installedMods)
	if err != nil {
		return models_service.ModulesChangeRequest{}, err
	}
	var toRemoveMods []string
	for _, item := range reqItems {
		if item.Remove {
			toRemoveMods = append(toRemoveMods, item.Id)
		}
	}
	changeRequest := newModulesChangeRequest(selectedRepoMods, installedMods, toRemoveMods)
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
			Id:     id,
			Action: models_service.ChangeActionRemove,
		}
		err := s.modulesHandler.Remove(ctx, id)
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
			Id:     repoMod.Mod.ID,
			Action: models_service.ChangeActionInstall,
		}
		err := s.modulesHandler.Add(ctx, repoMod.Mod.ID, repoMod.Source, repoMod.Channel, repoMod.FS)
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
			Id:     item.Next.Mod.ID,
			Action: models_service.ChangeActionChange,
		}
		err := s.modulesHandler.Update(ctx, item.Next.Mod.ID, item.Next.Source, item.Next.Channel, item.Next.FS)
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
	installedMods, err := s.modulesHandler.Modules(ctx, models_handler_module.ModuleFilter{})
	if err != nil {
		return modulesChangeRequest{}, err
	}
	if len(installedMods) == 0 {
		return modulesChangeRequest{}, nil
	}
	repoMods, err := s.repositoriesHandler.Modules(ctx, models_handler_repo.ModulesFilter{Ids: slices.Collect(maps.Keys(installedMods))})
	if err != nil {
		return modulesChangeRequest{}, err
	}
	if len(repoMods) == 0 {
		return modulesChangeRequest{}, nil
	}
	var reqItems []models_service.ChangeRequestItem
	for _, repoMod := range repoMods {
		installedMod, ok := installedMods[repoMod.Id]
		if !ok {
			continue
		}
		if installedMod.Source == repoMod.Source && installedMod.Channel == repoMod.Channel && installedMod.Version != repoMod.Version {
			reqItems = append(reqItems, models_service.ChangeRequestItem{
				Id:      installedMod.ID,
				Source:  installedMod.Source,
				Channel: installedMod.Channel,
			})
		}
	}
	selectedRepoMods, err := s.selectRepoModules(ctx, reqItems, installedMods)
	if err != nil {
		return modulesChangeRequest{}, err
	}
	return newModulesChangeRequest(selectedRepoMods, installedMods, nil), nil
}

func validateReqItems(reqItems []models_service.ChangeRequestItem) ([]models_service.ChangeRequestItem, error) {
	var validatedItems []models_service.ChangeRequestItem
	tmpMap := make(map[string]models_service.ChangeRequestItem)
	for _, item := range reqItems {
		if (item.Update && item.Remove) || (!(item.Update || item.Remove) && item.Source+item.Channel == "") {
			return nil, fmt.Errorf("ivalid change request for '%s'", item.Id)
		}
		if tmp, ok := tmpMap[item.Id]; ok {
			if !reflect.DeepEqual(tmp, item) {
				return nil, fmt.Errorf("duplicate entry for '%s'", item.Id)
			}
			continue
		}
		tmpMap[item.Id] = item
		validatedItems = append(validatedItems, item)
	}
	return validatedItems, nil
}

func newModulesChangeRequest(selectedRepoMods map[string]modWrapper, installedModsMap map[string]models_handler_module.Module, toRemoveMods []string) modulesChangeRequest {
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
					Id:   installedMod.ID,
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
		Id:   w.Mod.ID,
		Name: w.Mod.Name,
		Desc: w.Mod.Description,
		ModuleVariant: models_service.ModuleVariant{
			Source:  w.Source,
			Channel: w.Channel,
			Version: w.Mod.Version,
		},
	}
}
