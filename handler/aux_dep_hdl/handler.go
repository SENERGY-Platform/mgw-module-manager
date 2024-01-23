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
	"context"
	"errors"
	job_hdl_lib "github.com/SENERGY-Platform/go-service-base/job-hdl/lib"
	cew_lib "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/naming_hdl"
	"net/http"
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

func (h *Handler) List(ctx context.Context, dID string, filter lib_model.AuxDepFilter, assets, containerInfo bool) (map[string]lib_model.AuxDeployment, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	auxDeployments, err := h.storageHandler.ListAuxDep(ctxWt, dID, filter, assets)
	if err != nil {
		return nil, err
	}
	if containerInfo && len(auxDeployments) > 0 {
		ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
		defer cf2()
		ctrList, err := h.cewClient.GetContainers(ctxWt2, cew_model.ContainerFilter{Labels: map[string]string{naming_hdl.ManagerIDLabel: h.managerID, naming_hdl.DeploymentIDLabel: dID}})
		if err != nil {
			util.Logger.Errorf("could not retrieve containers: %s", err.Error())
			return auxDeployments, nil
		}
		ctrMap := make(map[string]cew_model.Container)
		for _, ctr := range ctrList {
			ctrMap[ctr.ID] = ctr
		}
		withCtrInfo := make(map[string]lib_model.AuxDeployment)
		for aID, auxDeployment := range auxDeployments {
			ctr, ok := ctrMap[auxDeployment.Container.ID]
			if ok {
				auxDeployment.Container.Info = &lib_model.ContainerInfo{
					ImageID: ctr.ImageID,
					State:   ctr.State,
				}
			} else {
				util.Logger.Warningf("aux deployment '%s' missing container '%s'", aID, auxDeployment.Container.ID)
			}
			withCtrInfo[aID] = auxDeployment
		}
		return withCtrInfo, nil
	}
	return auxDeployments, nil
}

func (h *Handler) Get(ctx context.Context, aID string, assets, containerInfo bool) (lib_model.AuxDeployment, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	auxDeployment, err := h.storageHandler.ReadAuxDep(ctxWt, aID, assets)
	if err != nil {
		return lib_model.AuxDeployment{}, err
	}
	if containerInfo {
		ctxWt2, cf2 := context.WithTimeout(ctx, h.httpTimeout)
		defer cf2()
		ctr, err := h.cewClient.GetContainer(ctxWt2, auxDeployment.Container.ID)
		if err != nil {
			util.Logger.Error(err)
		} else {
			auxDeployment.Container.Info = &lib_model.ContainerInfo{
				ImageID: ctr.ImageID,
				State:   ctr.State,
			}
		}
	}
	return auxDeployment, nil
}

func (h *Handler) Delete(ctx context.Context, aID string, force bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	auxDeployment, err := h.storageHandler.ReadAuxDep(ctxWt, aID, false)
	if err != nil {
		return err
	}
	if err = h.removeContainer(ctx, auxDeployment.Container.ID, force); err != nil {
		return err
	}
	ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
	defer cf2()
	return h.storageHandler.DeleteAuxDep(ctxWt2, nil, aID)
}

func (h *Handler) DeleteAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter, force bool) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	auxDeployments, err := h.storageHandler.ListAuxDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID, filter, false)
	if err != nil {
		return err
	}
	for aID, auxDeployment := range auxDeployments {
		if err = h.removeContainer(ctx, auxDeployment.Container.ID, force); err != nil {
			return err
		}
		if err = h.storageHandler.DeleteAuxDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), nil, aID); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) Start(ctx context.Context, aID string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	auxDeployment, err := h.storageHandler.ReadAuxDep(ctxWt, aID, false)
	if err != nil {
		return err
	}
	ctxWt2, cf2 := context.WithTimeout(ctx, h.httpTimeout)
	defer cf2()
	if err = h.cewClient.StartContainer(ctxWt2, auxDeployment.Container.ID); err != nil {
		return lib_model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) StartAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	auxDeployments, err := h.storageHandler.ListAuxDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID, filter, false)
	if err != nil {
		return err
	}
	for _, auxDeployment := range auxDeployments {
		if err = h.cewClient.StartContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), auxDeployment.Container.ID); err != nil {
			return lib_model.NewInternalError(err)
		}
	}
	return nil
}

func (h *Handler) Stop(ctx context.Context, aID string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	auxDeployment, err := h.storageHandler.ReadAuxDep(ctxWt, aID, false)
	if err != nil {
		return err
	}
	if err = h.stopContainer(ctx, auxDeployment.Container.ID); err != nil {
		return lib_model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) StopAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	auxDeployments, err := h.storageHandler.ListAuxDep(ctxWt, dID, filter, false)
	if err != nil {
		return err
	}
	for _, auxDeployment := range auxDeployments {
		if err = h.stopContainer(ctx, auxDeployment.Container.ID); err != nil {
			return lib_model.NewInternalError(err)
		}
	}
	return nil
}

func (h *Handler) Restart(ctx context.Context, aID string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	auxDeployment, err := h.storageHandler.ReadAuxDep(ctxWt, aID, false)
	if err != nil {
		return err
	}
	return h.restart(ctx, auxDeployment.Container.ID)
}

func (h *Handler) RestartAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	auxDeployments, err := h.storageHandler.ListAuxDep(ctxWt, dID, filter, false)
	if err != nil {
		return err
	}
	for _, auxDeployment := range auxDeployments {
		if err = h.restart(ctx, auxDeployment.Container.ID); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) removeContainer(ctx context.Context, id string, force bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	if err := h.cewClient.RemoveContainer(ctxWt, id, force); err != nil {
		var nfe *cew_model.NotFoundError
		if !errors.As(err, &nfe) {
			return lib_model.NewInternalError(err)
		}
	}
	return nil
}

func (h *Handler) stopContainer(ctx context.Context, cID string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	jID, err := h.cewClient.StopContainer(ctxWt, cID)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	job, err := job_hdl_lib.Await(ctx, h.cewClient, jID, time.Second, h.httpTimeout, util.Logger)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	if job.Error != nil {
		if job.Error.Code != nil && *job.Error.Code == http.StatusNotFound {
			return lib_model.NewNotFoundError(errors.New(job.Error.Message))
		}
		return lib_model.NewInternalError(errors.New(job.Error.Message))
	}
	return nil
}

func (h *Handler) restart(ctx context.Context, cID string) error {
	if err := h.stopContainer(ctx, cID); err != nil {
		return err
	}
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	if err := h.cewClient.StartContainer(ctxWt, cID); err != nil {
		return lib_model.NewInternalError(err)
	}
	return nil
}

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
