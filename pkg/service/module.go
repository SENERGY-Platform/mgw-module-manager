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
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"

	helper_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/configs"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	models_config "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/config"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_handler_database "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
	models_handler_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployments"
	models_handler_modules "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/modules"
	models_handler_repositories "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/repositories"
	models_service "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
)

func (s *Service) Modules(ctx context.Context, filter models_service.ModulesFilter) ([]models_service.ModuleReduced, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.jobsHandler.CurrentSlotJob(moduleJobSlotNum)
	if ok {
		return nil, errors.New("active job") // TODO
	}
	modules, err := s.modulesHandler.Modules(ctx, models_handler_modules.ModuleFilter{
		Ids:  filter.Ids,
		Name: filter.Name,
	})
	if err != nil {
		return nil, err
	}
	deployments, err := s.deploymentsHandler.GetReducedDeploymentsByModuleIds(ctx, models_handler_deployments.DeploymentsFilter{
		DeploymentsFilter: models_handler_database.DeploymentsFilter{
			ModuleIds: slices.Collect(maps.Keys(modules)),
		},
	})
	if err != nil {
		return nil, err
	}
	return getModulesReduced(modules, deployments, filter), nil
}

func (s *Service) Module(ctx context.Context, id string) (models_service.Module, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.jobsHandler.CurrentSlotJob(moduleJobSlotNum)
	if ok {
		return models_service.Module{}, errors.New("active job") // TODO
	}
	handlerModule, err := s.modulesHandler.Module(ctx, id)
	if err != nil {
		return models_service.Module{}, err
	}
	ok = true
	handlerDeployment, err := s.deploymentsHandler.GetDeploymentByModuleId(ctx, id)
	if err != nil {
		if !errors.Is(err, models_error.NotFoundErr) {
			return models_service.Module{}, err
		}
		ok = false
	}
	module := getModule(handlerModule, handlerDeployment)
	module.IsDeployed = ok
	return module, nil
}

func (s *Service) ModulesChangeRequest(_ context.Context) (models_service.ModulesChangeRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.changeRequest == nil {
		return models_service.ModulesChangeRequest{}, models_error.NotFoundErr
	}
	return transformModulesChangeRequest(*s.changeRequest), nil
}

func (s *Service) NewModulesChangeRequest(
	ctx context.Context,
	reqItems []models_service.ChangeRequestItem,
) (models_service.ModulesChangeRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{moduleJobSlotNum, repositoryJobSlotNum})
	if len(currentJobs) > 0 {
		return models_service.ModulesChangeRequest{}, errors.New("active jobs") // TODO
	}
	reqItems, err := validateReqItems(reqItems)
	if err != nil {
		return models_service.ModulesChangeRequest{}, err
	}
	installedMods, err := s.modulesHandler.Modules(ctx, models_handler_modules.ModuleFilter{})
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
	s.changeRequest = &changeRequest
	return transformModulesChangeRequest(changeRequest), nil
}

func (s *Service) ExecModulesChangeRequest(_ context.Context) (models_service.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.changeRequest == nil {
		return models_service.Job{}, models_error.NotFoundErr
	}
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{moduleJobSlotNum, repositoryJobSlotNum, deploymentJobSlotNum})
	if len(currentJobs) > 0 {
		return models_service.Job{}, errors.New("active jobs") // TODO
	}
	job, err := s.jobsHandler.CreateSlotJob(moduleJobSlotNum, "execute modules change")
	if err != nil {
		return models_service.Job{}, err
	}
	go func() {
		defer job.Done()
		jobResult := models_service.JobResultModulesChange{
			JobResult: models_service.JobResult{JobId: job.Id},
		}
		defer func() {
			if err := recover(); err != nil {
				jobResult.ErrorResult = models_error.NewErrorResult(fmt.Sprintf("panic: %v", err))
				s.setModuleChangeJobResult(job.Id, jobResult)
			}
		}()
		jobResult.ModulesChangeReport = s.execModulesChangeRequest(job.Context())
		s.setModuleChangeJobResult(job.Id, jobResult)
	}()
	return models_service.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) CancelModulesChangeRequest(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.changeRequest == nil {
		return models_error.NotFoundErr
	}
	s.changeRequest = nil
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
	s.changeRequest = &changeRequest
	return transformModulesChangeRequest(changeRequest), nil
}

func (s *Service) newModulesUpdateAllChangeRequest(ctx context.Context) (modulesChangeRequest, error) {
	installedMods, err := s.modulesHandler.Modules(ctx, models_handler_modules.ModuleFilter{})
	if err != nil {
		return modulesChangeRequest{}, err
	}
	if len(installedMods) == 0 {
		return modulesChangeRequest{}, nil
	}
	repoMods, err := s.repositoriesHandler.Modules(ctx, models_handler_repositories.ModulesFilter{Ids: slices.Collect(maps.Keys(installedMods))})
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

func (s *Service) execModulesChangeRequest(ctx context.Context) models_service.ModulesChangeReport {
	defer func() {
		s.changeRequest = nil
	}()
	var success []models_service.ChangeReportItem
	var failed []models_service.ChangeReportErrItem
	for _, id := range s.changeRequest.Remove {
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
	for _, repoMod := range s.changeRequest.Install {
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
	for _, item := range s.changeRequest.Change {
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
	}
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

func newModulesChangeRequest(
	selectedRepoMods map[string]modWrapper,
	installedModsMap map[string]models_handler_modules.Module,
	toRemoveMods []string,
) modulesChangeRequest {
	var install []modWrapper
	var change []changeItem
	var remove []string
	for id, repoMod := range selectedRepoMods {
		installedMod, ok := installedModsMap[id]
		if ok {
			if equalMods(repoMod, installedMod) {
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

func equalMods(repoMod modWrapper, installedMod models_handler_modules.Module) bool {
	return repoMod.Mod.ID == installedMod.ID &&
		repoMod.Source == installedMod.Source &&
		repoMod.Channel == installedMod.Channel &&
		repoMod.Mod.Version == installedMod.Version
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

func getModulesReduced(
	handlerModules map[string]models_handler_modules.Module,
	handlerDeployments map[string]models_handler_deployments.DeploymentReduced,
	filter models_service.ModulesFilter) []models_service.ModuleReduced {
	var modules []models_service.ModuleReduced
	for moduleId, module := range handlerModules {
		deployment, ok := handlerDeployments[moduleId]
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
		// TODO implement tags filter
		modules = append(modules, models_service.ModuleReduced{
			Id:          moduleId,
			Source:      module.Source,
			Channel:     module.Channel,
			Version:     module.Version,
			Name:        module.Name,
			Description: module.Description,
			Tags:        slices.Collect(maps.Keys(module.Tags)),
			License:     module.License,
			Author:      module.Author,
			IsDeployed:  ok,
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
	return modules
}

func getModule(module models_handler_modules.Module, deployment models_handler_deployments.Deployment) models_service.Module {
	containers := make(map[string]models_service.Container)
	for reference, container := range deployment.Containers {
		containers[reference] = models_service.Container{
			Name:    container.Name,
			Alias:   container.Alias,
			ImageId: container.ImageId,
			State:   container.State,
			Health:  container.Health,
		}
	}
	volumes := make(map[string]string)
	for reference, volume := range deployment.Volumes {
		volumes[reference] = volume.Name
	}
	hostResources := make(map[string]string)
	for reference, resource := range deployment.HostResources {
		hostResources[reference] = resource.Id
	}
	secrets := make(map[string]models_service.Secret)
	for reference, secret := range deployment.Secrets {
		secrets[reference] = models_service.Secret{
			Id:    secret.Id,
			Items: secret.Items,
		}
	}
	configs := make(map[string]models_config.InterfaceValue)
	for reference, config := range deployment.Configs {
		configs[reference] = models_config.InterfaceValue{
			DataType: config.DataType,
			IsSlice:  config.IsSlice,
			Value:    helper_configs.ValueToInterface(config.Value),
		}
	}
	globalConfigs := make(map[string]string)
	for reference, globalConfig := range deployment.GlobalConfigs {
		globalConfigs[reference] = globalConfig.Id
	}
	files := make(map[string]string)
	for reference, file := range deployment.Files {
		files[reference] = base64.StdEncoding.EncodeToString(file.Data)
	}
	fileGroups := make(map[string]models_service.FileGroup)
	for reference, fileGroup := range deployment.FileGroups {
		var fileGroupFiles []models_service.FileGroupFile
		for _, file := range fileGroup.Files {
			fileGroupFiles = append(fileGroupFiles, models_service.FileGroupFile{
				Path:   file.Path,
				Format: file.Format,
				Data:   base64.StdEncoding.EncodeToString(file.Data),
			})
		}
		fileGroups[reference] = models_service.FileGroup{
			Id:    fileGroup.Id,
			Files: fileGroupFiles,
		}
	}
	return models_service.Module{
		ModuleLibModule: module.ModuleLibModule,
		Source:          module.Source,
		Channel:         module.Channel,
		Added:           module.Added,
		Updated:         module.Updated,
		Deployment: models_service.Deployment{
			Id:            deployment.Id,
			ModuleSource:  deployment.ModuleSource,
			ModuleChannel: deployment.ModuleChannel,
			ModuleVersion: deployment.ModuleVersion,
			Enabled:       deployment.Enabled,
			Created:       deployment.Created,
			Updated:       deployment.Updated,
			Containers:    containers,
			Volumes:       volumes,
			HostResources: hostResources,
			Secrets:       secrets,
			Configs:       configs,
			GlobalConfigs: globalConfigs,
			Files:         files,
			FileGroups:    fileGroups,
			State:         deployment.State,
		},
	}
}
