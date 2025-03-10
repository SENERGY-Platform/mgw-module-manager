/*
 * Copyright 2024 InfAI (CC SES)
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

package aux_dep_hdl

import (
	"context"
	"errors"
	"fmt"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	context_hdl "github.com/SENERGY-Platform/mgw-go-service-base/context-hdl"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/naming_hdl"
	"time"
)

func (h *Handler) Update(ctx context.Context, aID string, mod *module.Module, dep lib_model.Deployment, requiredDep map[string]lib_model.Deployment, auxReq lib_model.AuxDepReq, forcePullImg, incremental bool) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	oldAuxDep, err := h.storageHandler.ReadAuxDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), aID, true)
	if err != nil {
		return err
	}
	if oldAuxDep.DepID != dep.ID {
		return lib_model.NewForbiddenError(errors.New("deployment ID mismatch"))
	}
	auxSrv, ok := mod.AuxServices[oldAuxDep.Ref]
	if !ok {
		return lib_model.NewInvalidInputError(fmt.Errorf("aux service ref '%s' not defined", auxReq.Ref))
	}
	newAuxDep := oldAuxDep
	if auxReq.Image != "" && auxReq.Image != oldAuxDep.Image {
		if ok, err := validImage(mod.AuxImgSrc, auxReq.Image); err != nil {
			return lib_model.NewInternalError(err)
		} else if !ok {
			return lib_model.NewInvalidInputError(errors.New("invalid image"))
		}
		newAuxDep.Image = auxReq.Image
	}
	if incremental {
		for key, val := range auxReq.Labels {
			newAuxDep.Labels[key] = val
		}
		for key, val := range auxReq.Configs {
			newAuxDep.Configs[key] = val
		}
		for key, val := range auxReq.Volumes {
			newAuxDep.Volumes[key] = val
		}
	} else {
		newAuxDep.Labels = auxReq.Labels
		newAuxDep.Configs = auxReq.Configs
		newAuxDep.Volumes = auxReq.Volumes
	}
	if auxReq.Name != "" {
		newAuxDep.Name = auxReq.Name
	}
	if auxReq.RunConfig != nil {
		newAuxDep.RunConfig = *auxReq.RunConfig
	}
	newAuxDep.Updated = time.Now().UTC()
	modVolumes := make(map[string]string)
	for ref := range mod.Volumes {
		modVolumes[ref] = naming_hdl.Global.NewVolumeName(dep.ID, ref)
	}
	defer func() {
		if err != nil {
			if oldAuxDep.Enabled {
				if e := h.cewClient.StartContainer(context.Background(), oldAuxDep.Container.ID); err != nil {
					util.Logger.Error(e)
				}
			}
		}
	}()
	if err = h.pullImage(ctx, newAuxDep.Image, forcePullImg); err != nil {
		return lib_model.NewInternalError(err)
	}
	if err = h.stopContainer(ctx, oldAuxDep.Container.ID); err != nil {
		return err
	}
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = h.storageHandler.DeleteAuxDepContainer(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, aID); err != nil {
		return err
	}
	if err = h.storageHandler.UpdateAuxDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, newAuxDep.AuxDepBase); err != nil {
		return err
	}
	auxVolumes, newAuxVolumes, orphanAuxVolumes, err := h.diffVolumes(ctx, aID, newAuxDep.Volumes)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	if err = h.createVolumes(ctx, newAuxVolumes, dep.ID, aID); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			var nv []string
			for _, v := range newAuxVolumes {
				nv = append(nv, v)
			}
			if e := h.removeVolumes(context.Background(), nv, true); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
	container, err := h.createContainer(ctx, auxSrv, newAuxDep, mod, dep, requiredDep, modVolumes, auxVolumes)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := h.removeContainer(context.Background(), container.ID, true); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
	if err = h.storageHandler.CreateAuxDepContainer(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, aID, container); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if oldAuxDep.Enabled {
		if e := h.cewClient.StartContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), container.ID); err != nil {
			util.Logger.Error(e)
		}
	}
	if e := h.removeContainer(ctx, oldAuxDep.Container.ID, true); e != nil {
		util.Logger.Error(e)
	}
	if e := h.removeVolumes(ctx, orphanAuxVolumes, true); e != nil {
		util.Logger.Error(e)
	}
	return nil
}

func (h *Handler) UpdateAll(ctx context.Context, mod *module.Module, dep lib_model.Deployment, requiredDep map[string]lib_model.Deployment) ([]string, error) {
	auxServices := make(map[string]*module.AuxService)
	for ref, auxSrv := range mod.AuxServices {
		if len(auxSrv.BindMounts)+len(auxSrv.Tmpfs)+len(auxSrv.Volumes)+len(auxSrv.Configs)+len(auxSrv.SrvReferences)+len(auxSrv.ExtDependencies) > 0 {
			auxServices[ref] = auxSrv
		}
	}
	var updated []string
	if len(auxServices) > 0 {
		modVolumes := make(map[string]string)
		for ref := range mod.Volumes {
			modVolumes[ref] = naming_hdl.Global.NewVolumeName(dep.ID, ref)
		}
		ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
		defer cf()
		auxDeployments, err := h.storageHandler.ListAuxDep(ctxWt, dep.ID, lib_model.AuxDepFilter{}, true)
		if err != nil {
			return nil, err
		}
		for _, auxDep := range auxDeployments {
			if auxSrv, ok := auxServices[auxDep.Ref]; ok {
				if err = h.updateBase(ctx, mod, modVolumes, dep, requiredDep, auxSrv, auxDep); err != nil {
					util.Logger.Error(err)
					continue
				}
				updated = append(updated, auxDep.ID)
			}
		}
	}
	return updated, nil
}

func (h *Handler) updateBase(ctx context.Context, mod *module.Module, modVolumes map[string]string, dep lib_model.Deployment, requiredDep map[string]lib_model.Deployment, auxSrv *module.AuxService, auxDep lib_model.AuxDeployment) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	if err := h.stopContainer(ctx, auxDep.Container.ID); err != nil {
		return err
	}
	auxVolumes := make(map[string]string)
	for ref := range auxDep.Volumes {
		auxVolumes[ref] = naming_hdl.Global.NewVolumeName(auxDep.ID, ref)
	}
	auxDep.Updated = time.Now().UTC()
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = h.storageHandler.DeleteAuxDepContainer(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, auxDep.ID); err != nil {
		return err
	}
	if err = h.storageHandler.UpdateAuxDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, auxDep.AuxDepBase); err != nil {
		return err
	}
	container, err := h.createContainer(ctx, auxSrv, auxDep, mod, dep, requiredDep, modVolumes, auxVolumes)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := h.removeContainer(context.Background(), container.ID, true); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
	if err = h.storageHandler.CreateAuxDepContainer(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, auxDep.ID, container); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if auxDep.Enabled {
		if e := h.cewClient.StartContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), container.ID); err != nil {
			util.Logger.Error(e)
		}
	}
	if e := h.removeContainer(ctx, auxDep.Container.ID, true); e != nil {
		util.Logger.Error(e)
	}
	return nil
}

func (h *Handler) diffVolumes(ctx context.Context, aID string, auxVolumes map[string]string) (map[string]string, map[string]string, []string, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	cewVolumes, err := h.cewClient.GetVolumes(ctxWt, cew_model.VolumeFilter{Labels: map[string]string{naming_hdl.ManagerIDLabel: h.managerID, naming_hdl.AuxDeploymentID: aID}})
	if err != nil {
		return nil, nil, nil, err
	}
	hashAuxVolMap := make(map[string]string)
	for ref := range auxVolumes {
		hashAuxVolMap[naming_hdl.Global.NewVolumeName(aID, ref)] = ref
	}
	volumes := make(map[string]string)
	var orphanVolumes []string
	for _, v := range cewVolumes {
		ref, ok := hashAuxVolMap[v.Name]
		if !ok {
			orphanVolumes = append(orphanVolumes, v.Name)
			continue
		}
		volumes[ref] = v.Name
	}
	newVolumes := make(map[string]string)
	for hsh, ref := range hashAuxVolMap {
		if _, ok := volumes[ref]; !ok {
			volumes[ref] = hsh
			newVolumes[ref] = hsh
		}
	}
	return volumes, newVolumes, orphanVolumes, nil
}
