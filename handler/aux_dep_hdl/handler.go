/*
 * Copyright 2023 InfAI (CC SES)
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
	cew_lib "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"time"
)

type Handler struct {
	storageHandler handler.AuxDepStorageHandler
	cewClient      cew_lib.Api
	dbTimeout      time.Duration
	httpTimeout    time.Duration
	managerID      string
	coreID         string
	moduleNet      string
	depHostPath    string
}

func New(storageHandler handler.AuxDepStorageHandler, cewClient cew_lib.Api, dbTimeout, httpTimeout time.Duration, managerID, moduleNet, coreID, depHostPath string) *Handler {
	return &Handler{
		storageHandler: storageHandler,
		cewClient:      cewClient,
		dbTimeout:      dbTimeout,
		httpTimeout:    httpTimeout,
		managerID:      managerID,
		coreID:         coreID,
		moduleNet:      moduleNet,
		depHostPath:    depHostPath,
	}
}

//func (h *Handler) List(ctx context.Context, dID string, filter model.AuxDepFilter, ctrInfo bool) ([]model.AuxDeployment, error) {
//	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
//	defer cf()
//	auxDeployments, err := h.storageHandler.ListAuxDep(ctxWt, dID, filter)
//	if err != nil {
//		return nil, err
//	}
//	if ctrInfo && len(auxDeployments) > 0 {
//		ctrMap, err := h.getContainersMap(ctx, dID)
//		if err != nil {
//			util.Logger.Error(err)
//		} else {
//			var auxDeps []model.AuxDeployment
//			for _, auxDep := range auxDeployments {
//				ctr, ok := ctrMap[auxDep.Container.ID]
//				if !ok {
//					return nil, model.NewInternalError(fmt.Errorf("container '%s' not in map", auxDep.Container.ID))
//				}
//				auxDep.Container.Info = &model.AuxDepCtrInfo{
//					ImageID: ctr.ImageID,
//					State:   ctr.State,
//				}
//				auxDeps = append(auxDeps, auxDep)
//			}
//			return auxDeps, nil
//		}
//	}
//	return auxDeployments, nil
//}
//
//func (h *Handler) Get(ctx context.Context, aID string, ctrInfo bool) (model.AuxDeployment, error) {
//	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
//	defer cf()
//	auxDep, err := h.storageHandler.ReadAuxDep(ctxWt, aID)
//	if err != nil {
//		return model.AuxDeployment{}, err
//	}
//	if ctrInfo {
//		ctxWt2, cf2 := context.WithTimeout(ctx, h.httpTimeout)
//		defer cf2()
//		ctr, err := h.cewClient.GetContainer(ctxWt2, auxDep.Container.ID)
//		if err != nil {
//			util.Logger.Error(err)
//		} else {
//			auxDep.Container.Info = &model.AuxDepCtrInfo{
//				ImageID: ctr.ImageID,
//				State:   ctr.State,
//			}
//		}
//	}
//	return auxDep, nil
//}
//
//func (h *Handler) Create(ctx context.Context, mod *module.Module, dep model.Deployment, auxReq model.AuxDepReq) (string, error) {
//	auxSrv, ok := mod.AuxServices[auxReq.Ref]
//	if !ok {
//		return "", model.NewInvalidInputError(fmt.Errorf("aux service ref '%s' no defined", auxReq.Ref))
//	}
//	if err := setModConfigs(mod.Configs, auxSrv.Configs, auxReq.Configs); err != nil {
//		return "", err
//	}
//	name := auxSrv.Name
//	if auxReq.Name != nil && *auxReq.Name != "" {
//		name = *auxReq.Name
//	}
//	timestamp := time.Now().UTC()
//	tx, err := h.storageHandler.BeginTransaction(ctx)
//	if err != nil {
//		return "", err
//	}
//	defer tx.Rollback()
//	ch := context_hdl.New()
//	defer ch.CancelAll()
//	aID, err := h.storageHandler.CreateAuxDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, model.AuxDepBase{
//		DepID:   auxReq.DepID,
//		Image:   auxReq.Image,
//		Labels:  auxReq.Labels,
//		Configs: auxReq.Configs,
//		Ref:     auxReq.Ref,
//		Name:    name,
//		Created: timestamp,
//		Updated: timestamp,
//	})
//	if err != nil {
//		return "", err
//	}
//	ctrName, err := getCtrName(aID)
//	if err != nil {
//		return "", err
//	}
//	alias := getAuxSrvName(auxReq.DepID, aID)
//	mounts := getMounts(auxSrv, auxReq.DepID, dep.Dir, h.depHostPath)
//	ctr := getContainer(auxSrv, ctrName, alias, auxReq.Image, h.coreID, h.managerID, auxReq.DepID, aID, h.moduleNet, envVars, mounts)
//	cID, err := h.cewClient.CreateContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), ctr)
//	if err != nil {
//		return "", err
//	}
//	if err = h.storageHandler.CreateAuxDepCtr(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, aID, model.AuxDepContainer{ID: cID, Alias: alias}); err != nil {
//		return "", err
//	}
//	if err = tx.Commit(); err != nil {
//		return "", err
//	}
//	return aID, nil
//}
//
//func (h *Handler) Update(ctx context.Context, aID string, mod *module.Module, auxReq model.AuxDepReq) error {
//	panic("not implemented")
//}
//
//func (h *Handler) Delete(ctx context.Context, aID string) error {
//	panic("not implemented")
//}
//
//func (h *Handler) DeleteAll(ctx context.Context, dID string, filter model.AuxDepFilter) error {
//	panic("not implemented")
//}
//
//func (h *Handler) Start(ctx context.Context, aID string) error {
//	panic("not implemented")
//}
//
//func (h *Handler) StartAll(ctx context.Context, dID string, filter model.AuxDepFilter) error {
//	panic("not implemented")
//}
//
//func (h *Handler) Stop(ctx context.Context, aID string) error {
//	panic("not implemented")
//}
//
//func (h *Handler) StopAll(ctx context.Context, dID string, filter model.AuxDepFilter) error {
//	panic("not implemented")
//}
//
//func (h *Handler) getContainersMap(ctx context.Context, dID string) (map[string]cew_model.Container, error) {
//	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
//	defer cf()
//	containers, err := h.cewClient.GetContainers(ctxWt, cew_model.ContainerFilter{Labels: map[string]string{handler.ManagerIDLabel: h.managerID, handler.DeploymentIDLabel: dID}})
//	if err != nil {
//		return nil, model.NewInternalError(err)
//	}
//	ctrMap := make(map[string]cew_model.Container)
//	for _, container := range containers {
//		ctrMap[container.ID] = container
//	}
//	return ctrMap, nil
//}
//
//func getCtrName(s string) (string, error) {
//	id, err := uuid.NewRandom()
//	if err != nil {
//		return "", err
//	}
//	return getAuxSrvName(s, id.String()), nil
//}
//
//func getContainer(srv *module.AuxService, name, alias, image, cID, mID, dID, aID, moduleNet string, envVars map[string]string, mounts []cew_model.Mount) cew_model.Container {
//	retries := int(srv.RunConfig.MaxRetries)
//	stopTimeout := srv.RunConfig.StopTimeout
//	return cew_model.Container{
//		Name:    name,
//		Image:   image,
//		EnvVars: envVars,
//		Labels:  map[string]string{handler.CoreIDLabel: cID, handler.ManagerIDLabel: mID, handler.DeploymentIDLabel: dID, handler.AuxDeploymentID: aID},
//		Mounts:  mounts,
//		Networks: []cew_model.ContainerNet{
//			{
//				Name:        moduleNet,
//				DomainNames: []string{alias, name},
//			},
//		},
//		RunConfig: cew_model.RunConfig{
//			RestartStrategy: cew_model.RestartOnFail,
//			Retries:         &retries,
//			StopTimeout:     &stopTimeout,
//			StopSignal:      srv.RunConfig.StopSignal,
//			PseudoTTY:       srv.RunConfig.PseudoTTY,
//		},
//	}
//}
//
//func setModConfigs(modConfigs module.Configs, configMap, configs map[string]string) error {
//	for refVar, ref := range configMap {
//		if _, ok := configs[refVar]; !ok {
//			mConfig, ok := modConfigs[ref]
//			if !ok {
//				return fmt.Errorf("config '%s' not defined", ref)
//			}
//			if mConfig.Required && mConfig.Default == nil {
//				return fmt.Errorf("config '%s' required", ref)
//			}
//			if mConfig.Default != nil {
//				var val string
//				var err error
//				if mConfig.IsSlice {
//					val, err = parser.DataTypeToStringList(mConfig.Default, mConfig.Delimiter, mConfig.DataType)
//				} else {
//					val, err = parser.DataTypeToString(mConfig.Default, mConfig.DataType)
//				}
//				if err != nil {
//					return err
//				}
//				configs[refVar] = val
//			}
//		}
//	}
//	return nil
//}
//
//func getEnvVars(srv *module.AuxService, configs, depMap map[string]string, dID, aID string) (map[string]string, error) {
//	envVars := make(map[string]string)
//	for eVar, cRef := range srv.Configs {
//		if val, ok := configs[cRef]; ok {
//			envVars[eVar] = val
//		}
//	}
//	for eVar, target := range srv.SrvReferences {
//		envVars[eVar] = target.FillTemplate(getSrvName(dID, target.Ref))
//	}
//	for eVar, target := range srv.ExtDependencies {
//		val, ok := depMap[target.ID]
//		if !ok {
//			return nil, fmt.Errorf("service '%s' of '%s' not deployed but required", target.Service, target.ID)
//		}
//		envVars[eVar] = target.FillTemplate(getSrvName(val, target.Service))
//	}
//	envVars["MGW_DID"] = dID
//	envVars["MGW_AID"] = aID
//	return envVars, nil
//}
//
//func getAuxSrvName(s, r string) string {
//	return "mgw-aux-" + util.GenHash(s, r)
//}
//
//func getVolumeName(dID, name string) string {
//	return "mgw_" + util.GenHash(dID, name)
//}
//
//func getMounts(srv *module.AuxService, dID, inclDir, depHostPath string) []cew_model.Mount {
//	var mounts []cew_model.Mount
//	for mntPoint, name := range srv.Volumes {
//		mounts = append(mounts, cew_model.Mount{
//			Type:   cew_model.VolumeMount,
//			Source: getVolumeName(dID, name),
//			Target: mntPoint,
//		})
//	}
//	for mntPoint, mount := range srv.BindMounts {
//		mounts = append(mounts, cew_model.Mount{
//			Type:     cew_model.BindMount,
//			Source:   path.Join(depHostPath, inclDir, mount.Source),
//			Target:   mntPoint,
//			ReadOnly: mount.ReadOnly,
//		})
//	}
//	for mntPoint, mount := range srv.Tmpfs {
//		mounts = append(mounts, cew_model.Mount{
//			Type:   cew_model.TmpfsMount,
//			Target: mntPoint,
//			Size:   int64(mount.Size),
//			Mode:   mount.Mode,
//		})
//	}
//	return mounts
//}

/*

create:
	insert auxDep return ID
		if err -> return
	create Container return CtrID
		if err -> rollback & return
	insert container ID
		if err -> remove container & rollback & return
	commit

update:
	create new container return CtrID
		if err -> remove new container & return
	update auxDep
		if err -> remove new container & return
	stop old container
		if err -> rollback & remove new container & return
	start new container
		if err -> rollback & remove new container & return
	commit
	remove old container

*/
