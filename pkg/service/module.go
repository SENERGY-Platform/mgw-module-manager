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
	"fmt"
	"maps"
	"reflect"
	"slices"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/models/constants"
	helper_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/configs"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

func (s *Service) Modules(ctx context.Context, filter lib_models.ModulesFilter) ([]lib_models.ModuleReduced, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	currentJob, ok := s.jobsHandler.CurrentSlotJob(moduleJobSlotNum)
	if ok {
		return nil, lib_errors.New[lib_errors.ErrActiveJob](activeJobErrMsg(currentJob))
	}
	modules, err := s.modulesHandler.GetModules(
		ctx,
		pkg_models.ModulesFilterWithName{
			ModulesFilter: pkg_models.ModulesFilter{
				Ids: filter.Ids,
			},
			Name: filter.Name,
		},
		false,
	)
	if err != nil {
		return nil, err
	}
	deployments, err := s.deploymentsHandler.GetReducedDeploymentsByModuleIds(ctx, pkg_models.DeploymentsFilterWithState{
		DeploymentsFilter: pkg_models.DeploymentsFilter{
			ModuleIds: slices.Collect(maps.Keys(modules)),
		},
	})
	if err != nil {
		return nil, err
	}
	return getModulesReduced(modules, deployments, filter), nil
}

func (s *Service) Module(ctx context.Context, id string) (lib_models.Module, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	currentJob, ok := s.jobsHandler.CurrentSlotJob(moduleJobSlotNum)
	if ok {
		return lib_models.Module{}, lib_errors.New[lib_errors.ErrActiveJob](activeJobErrMsg(currentJob))
	}
	handlerModule, err := s.modulesHandler.GetModule(ctx, id)
	if err != nil {
		return lib_models.Module{}, err
	}
	ok = true
	handlerDeployment, err := s.deploymentsHandler.GetDeploymentByModuleId(ctx, id)
	if err != nil {
		if !lib_errors.IsOf[lib_errors.ErrNotFound](err) {
			return lib_models.Module{}, err
		}
		ok = false
	}
	module := getModule(handlerModule, handlerDeployment)
	module.IsDeployed = ok
	return module, nil
}

func (s *Service) ModulesChangeRequest(_ context.Context) (lib_models.ModulesChangeRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.changeRequest == nil {
		return lib_models.ModulesChangeRequest{}, lib_errors.New[lib_errors.ErrNotFound]("no module change request available")
	}
	return transformModulesChangeRequest(*s.changeRequest), nil
}

func (s *Service) NewModulesChangeRequest(
	ctx context.Context,
	reqItems []lib_models.ChangeRequestItem,
) (lib_models.ModulesChangeRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{moduleJobSlotNum, repositoryJobSlotNum})
	if len(currentJobs) > 0 {
		return lib_models.ModulesChangeRequest{}, lib_errors.New[lib_errors.ErrActiveJob](activeJobsErrMsg(currentJobs))
	}
	reqItems, err := validateReqItems(reqItems)
	if err != nil {
		return lib_models.ModulesChangeRequest{}, err
	}
	installedMods, err := s.modulesHandler.GetModules(ctx, pkg_models.ModulesFilterWithName{}, false)
	if err != nil {
		return lib_models.ModulesChangeRequest{}, err
	}
	selectedRepoMods, err := s.selectRepoModules(ctx, reqItems, installedMods)
	if err != nil {
		return lib_models.ModulesChangeRequest{}, err
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

func (s *Service) ExecModulesChangeRequest(_ context.Context) (lib_models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.changeRequest == nil {
		return lib_models.Job{}, lib_errors.New[lib_errors.ErrNotFound]("no module change request available")
	}
	currentJobs := s.jobsHandler.CurrentSlotJobs([]int{moduleJobSlotNum, repositoryJobSlotNum, deploymentJobSlotNum})
	if len(currentJobs) > 0 {
		return lib_models.Job{}, lib_errors.New[lib_errors.ErrActiveJob]("active jobs")
	}
	job, err := s.jobsHandler.CreateSlotJob(moduleJobSlotNum, "execute modules change")
	if err != nil {
		return lib_models.Job{}, err
	}
	go func() {
		defer job.Done()
		jobResult := lib_models.ModulesChangeJobResult{
			JobResult: lib_models.JobResult{JobId: job.Id},
		}
		defer func() {
			if err := recover(); err != nil {
				jobResult.ErrorResult = lib_models.NewErrorResult(fmt.Sprintf("panic\n%v", err))
				s.setModuleChangeJobResult(job.Id, jobResult)
			}
		}()
		jobResult.ModulesChangeReport = s.execModulesChangeRequest(job.Context())
		s.setModuleChangeJobResult(job.Id, jobResult)
	}()
	return lib_models.Job{
		Id:          job.Id,
		Description: job.Description,
		Start:       job.Start,
	}, nil
}

func (s *Service) CancelModulesChangeRequest(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.changeRequest == nil {
		return lib_errors.New[lib_errors.ErrNotFound]("no module change request available")
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

func (s *Service) NewModulesUpdateAllChangeRequest(ctx context.Context) (lib_models.ModulesChangeRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	changeRequest, err := s.newModulesUpdateAllChangeRequest(ctx)
	if err != nil {
		return lib_models.ModulesChangeRequest{}, err
	}
	s.changeRequest = &changeRequest
	return transformModulesChangeRequest(changeRequest), nil
}

func (s *Service) newModulesUpdateAllChangeRequest(ctx context.Context) (modulesChangeRequest, error) {
	installedMods, err := s.modulesHandler.GetModules(ctx, pkg_models.ModulesFilterWithName{}, false)
	if err != nil {
		return modulesChangeRequest{}, err
	}
	if len(installedMods) == 0 {
		return modulesChangeRequest{}, nil
	}
	repoMods := s.repositoriesHandler.GetModules(ctx, pkg_models.RepositoryModulesFilter{Ids: slices.Collect(maps.Keys(installedMods))})
	if len(repoMods) == 0 {
		return modulesChangeRequest{}, nil
	}
	var reqItems []lib_models.ChangeRequestItem
	for _, repoMod := range repoMods {
		installedMod, ok := installedMods[repoMod.Id]
		if !ok {
			continue
		}
		if installedMod.Source == repoMod.Source && installedMod.Channel == repoMod.Channel && installedMod.Version != repoMod.Version {
			reqItems = append(reqItems, lib_models.ChangeRequestItem{
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

func (s *Service) execModulesChangeRequest(ctx context.Context) lib_models.ModulesChangeReport {
	defer func() {
		s.changeRequest = nil
	}()
	var success []lib_models.ChangeReportItem
	var failed []lib_models.ChangeReportErrItem
	for _, id := range s.changeRequest.Remove {
		cri := lib_models.ChangeReportItem{
			Id:     id,
			Action: lib_constants.ModuleChangeActionRemove,
		}
		err := s.modulesHandler.DeleteModule(ctx, id)
		if err != nil {
			failed = append(failed, lib_models.ChangeReportErrItem{
				ChangeReportItem: cri,
				Error:            err.Error(),
			})
			continue
		}
		success = append(success, cri)
	}
	for _, repoMod := range s.changeRequest.Install {
		cri := lib_models.ChangeReportItem{
			Id:     repoMod.Mod.ID,
			Action: lib_constants.ModuleChangeActionInstall,
		}
		err := s.modulesHandler.AddModule(ctx, repoMod.Mod.ID, repoMod.Source, repoMod.Channel, repoMod.FS)
		if err != nil {
			failed = append(failed, lib_models.ChangeReportErrItem{
				ChangeReportItem: cri,
				Error:            err.Error(),
			})
			continue
		}
		success = append(success, cri)
	}
	for _, item := range s.changeRequest.Change {
		cri := lib_models.ChangeReportItem{
			Id:     item.Next.Mod.ID,
			Action: lib_constants.ModuleChangeActionChange,
		}
		err := s.modulesHandler.UpdateModule(ctx, item.Next.Mod.ID, item.Next.Source, item.Next.Channel, item.Next.FS)
		if err != nil {
			failed = append(failed, lib_models.ChangeReportErrItem{
				ChangeReportItem: cri,
				Error:            err.Error(),
			})
			continue
		}
		success = append(success, cri)
	}
	return lib_models.ModulesChangeReport{
		Success: success,
		Failed:  failed,
	}
}

func validateReqItems(reqItems []lib_models.ChangeRequestItem) ([]lib_models.ChangeRequestItem, error) {
	var validatedItems []lib_models.ChangeRequestItem
	tmpMap := make(map[string]lib_models.ChangeRequestItem)
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
	installedModsMap map[string]pkg_models.Module,
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
				Previous: lib_models.ModuleAbbreviated{
					Id:   installedMod.ID,
					Name: installedMod.Name,
					Desc: installedMod.Description,
					ModuleVariant: lib_models.ModuleVariant{
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

func equalMods(repoMod modWrapper, installedMod pkg_models.Module) bool {
	return repoMod.Mod.ID == installedMod.ID &&
		repoMod.Source == installedMod.Source &&
		repoMod.Channel == installedMod.Channel &&
		repoMod.Mod.Version == installedMod.Version
}

func transformModulesChangeRequest(req modulesChangeRequest) lib_models.ModulesChangeRequest {
	mcr := lib_models.ModulesChangeRequest{
		Created: req.Created,
	}
	for _, mod := range req.Install {
		mcr.Install = append(mcr.Install, modWrapperToServiceModuleAbbreviated(mod))
	}
	for _, item := range req.Change {
		mcr.Change = append(mcr.Change, [2]lib_models.ModuleAbbreviated{
			item.Previous,
			modWrapperToServiceModuleAbbreviated(item.Next),
		})
	}
	if req.Remove != nil {
		mcr.Remove = req.Remove
	}
	return mcr
}

func modWrapperToServiceModuleAbbreviated(w modWrapper) lib_models.ModuleAbbreviated {
	return lib_models.ModuleAbbreviated{
		Id:   w.Mod.ID,
		Name: w.Mod.Name,
		Desc: w.Mod.Description,
		ModuleVariant: lib_models.ModuleVariant{
			Source:  w.Source,
			Channel: w.Channel,
			Version: w.Mod.Version,
		},
	}
}

func getModulesReduced(
	handlerModules map[string]pkg_models.Module,
	handlerDeployments map[string]pkg_models.DeploymentReduced,
	filter lib_models.ModulesFilter) []lib_models.ModuleReduced {
	var modules []lib_models.ModuleReduced
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
		modules = append(modules, lib_models.ModuleReduced{
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
			Deployment: lib_models.DeploymentReduced{
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

func getModule(module pkg_models.Module, deployment pkg_models.Deployment) lib_models.Module {
	containers := make(map[string]lib_models.Container)
	for reference, container := range deployment.Containers {
		containers[reference] = lib_models.Container{
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
	secrets := make(map[string]lib_models.DeploymentSecret)
	for reference, secret := range deployment.Secrets {
		secrets[reference] = lib_models.DeploymentSecret{
			Id:    secret.Id,
			Items: secret.Items,
		}
	}
	configs := make(map[string]lib_models.InterfaceValue)
	for reference, config := range deployment.Configs {
		configs[reference] = lib_models.InterfaceValue{
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
	fileGroups := make(map[string]lib_models.DeploymentFileGroup)
	for reference, fileGroup := range deployment.FileGroups {
		var fileGroupFiles []lib_models.DeploymentFileGroupFile
		for _, file := range fileGroup.Files {
			fileGroupFiles = append(fileGroupFiles, lib_models.DeploymentFileGroupFile{
				Path:   file.Path,
				Format: file.Format,
				Data:   base64.StdEncoding.EncodeToString(file.Data),
			})
		}
		fileGroups[reference] = lib_models.DeploymentFileGroup{
			Id:    fileGroup.Id,
			Files: fileGroupFiles,
		}
	}
	return lib_models.Module{
		ModuleLibModule: module.ModuleLibModule,
		Source:          module.Source,
		Channel:         module.Channel,
		Added:           module.Added,
		Updated:         module.Updated,
		Deployment: lib_models.Deployment{
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
