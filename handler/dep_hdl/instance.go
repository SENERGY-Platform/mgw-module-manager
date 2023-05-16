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

package dep_hdl

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
	"path"
	"time"
)

func (h *Handler) getCurrentInst(ctx context.Context, dID string) (model.Instance, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	instances, err := h.storageHandler.ListInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepInstFilter{DepID: dID})
	if err != nil {
		return model.Instance{}, err
	}
	if len(instances) != 1 {
		return model.Instance{}, model.NewInternalError(fmt.Errorf("invalid number of instances: %d", len(instances)))
	}
	return h.storageHandler.ReadInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), instances[0].ID)
}

func (h *Handler) createInstance(ctx context.Context, tx driver.Tx, mod *module.Module, dID, depDirPth string, stringValues, hostRes, secrets, reqModDepMap map[string]string) (string, []string, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	iID, err := h.storageHandler.CreateInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dID, time.Now().UTC())
	if err != nil {
		return "", nil, err
	}
	order, err := sorting.GetSrvOrder(mod.Services)
	if err != nil {
		return "", nil, model.NewInternalError(err)
	}
	var cIDs []string
	for i, ref := range order {
		srv := mod.Services[ref]
		envVars, err := getEnvVars(srv, stringValues, reqModDepMap, dID, iID)
		if err != nil {
			return "", nil, model.NewInternalError(err)
		}
		container := getContainer(srv, ref, getSrvName(iID, ref), dID, iID, envVars, getMounts(srv, hostRes, secrets, dID, depDirPth), getPorts(srv.Ports))
		cID, err := h.cewClient.CreateContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), container)
		if err != nil {
			return "", cIDs, model.NewInternalError(err)
		}
		cIDs = append(cIDs, cID)
		err = h.storageHandler.CreateInstCtr(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, iID, cID, ref, uint(i))
		if err != nil {
			return "", cIDs, err
		}
	}
	return iID, cIDs, nil
}

func (h *Handler) removeInstance(ctx context.Context, iID string) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	containers, err := h.storageHandler.ListInstCtr(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), iID, model.CtrFilter{})
	if err != nil {
		return err
	}
	for _, ctr := range containers {
		err = h.cewClient.RemoveContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), ctr.ID)
		if err != nil {
			var nfe *cew_model.NotFoundError
			if !errors.As(err, &nfe) {
				return model.NewInternalError(err)
			}
		}
	}
	return h.storageHandler.DeleteInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), iID)
}

func (h *Handler) startInstance(ctx context.Context, iID string) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	containers, err := h.storageHandler.ListInstCtr(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), iID, model.CtrFilter{SortOrder: model.Ascending})
	if err != nil {
		return err
	}
	for _, ctr := range containers {
		err = h.cewClient.StartContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), ctr.ID)
		if err != nil {
			return model.NewInternalError(err)
		}
	}
	return nil
}

func (h *Handler) stopInstance(ctx context.Context, iID string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	containers, err := h.storageHandler.ListInstCtr(ctxWt, iID, model.CtrFilter{SortOrder: model.Descending})
	if err != nil {
		return err
	}
	for _, ctr := range containers {
		if err = h.stopContainer(ctx, ctr.ID); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) stopContainer(ctx context.Context, cID string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	jID, err := h.cewClient.StopContainer(ctxWt, cID)
	if err != nil {
		return model.NewInternalError(err)
	}
	job, err := h.cewJobHandler.AwaitJob(ctx, jID)
	if err != nil {
		return err
	}
	if job.Error != nil {
		return model.NewInternalError(fmt.Errorf("%v", job.Error))
	}
	return nil
}

func getEnvVars(srv *module.Service, configs, depMap map[string]string, dID, iID string) (map[string]string, error) {
	envVars := make(map[string]string)
	for eVar, cRef := range srv.Configs {
		if val, ok := configs[cRef]; ok {
			envVars[eVar] = val
		}
	}
	for eVar, sRef := range srv.SrvReferences {
		envVars[eVar] = getSrvName(dID, sRef)
	}
	for eVar, target := range srv.ExtDependencies {
		val, ok := depMap[target.ID]
		if !ok {
			return nil, fmt.Errorf("service '%s' of '%s' not deployed but required", target.Service, target.ID)
		}
		envVars[eVar] = getSrvName(val, target.Service)
	}
	envVars["MGW_DID"] = dID
	envVars["MGW_IID"] = iID
	return envVars, nil
}

func getMounts(srv *module.Service, hostRes, secrets map[string]string, dID, depDirPth string) []cew_model.Mount {
	var mounts []cew_model.Mount
	for mntPoint, name := range srv.Volumes {
		mounts = append(mounts, cew_model.Mount{
			Type:   cew_model.VolumeMount,
			Source: getVolumeName(dID, name),
			Target: mntPoint,
		})
	}
	for mntPoint, mount := range srv.BindMounts {
		mounts = append(mounts, cew_model.Mount{
			Type:     cew_model.BindMount,
			Source:   path.Join(depDirPth, mount.Source),
			Target:   mntPoint,
			ReadOnly: mount.ReadOnly,
		})
	}
	for mntPoint, mount := range srv.Tmpfs {
		mounts = append(mounts, cew_model.Mount{
			Type:   cew_model.TmpfsMount,
			Target: mntPoint,
			Size:   int64(mount.Size),
			Mode:   mount.Mode,
		})
	}
	//for mntPoint, target := range srv.HostResources {
	//	src, ok := hostRes[target.Ref]
	//	if ok {
	//		mounts = append(mounts, cew_model.Mount{
	//			Type:     cew_model.BindMount,
	//			Source:   "",
	//			Target:   mntPoint,
	//			ReadOnly: target.ReadOnly,
	//		})
	//	}
	//}
	//for mntPoint, sRef := range srv.Secrets {
	//	src, ok := secrets[sRef]
	//	if ok {
	//		mounts = append(mounts, cew_model.Mount{
	//			Type:     cew_model.BindMount,
	//			Source:   "",
	//			Target:   mntPoint,
	//			ReadOnly: true,
	//		})
	//	}
	//}
	return mounts
}

func getPorts(sPorts []module.Port) (ports []cew_model.Port) {
	for _, port := range sPorts {
		p := cew_model.Port{
			Number:   int(port.Number),
			Protocol: port.Protocol,
		}
		if len(port.Bindings) > 0 {
			var bindings []cew_model.PortBinding
			for _, n := range port.Bindings {
				bindings = append(bindings, cew_model.PortBinding{Number: int(n)})
			}
			p.Bindings = bindings
		}
		ports = append(ports, p)
	}
	return ports
}

func getContainer(srv *module.Service, ref, name, dID, iID string, envVars map[string]string, mounts []cew_model.Mount, ports []cew_model.Port) cew_model.Container {
	retries := int(srv.RunConfig.MaxRetries)
	stopTimeout := srv.RunConfig.StopTimeout
	return cew_model.Container{
		Name:    name,
		Image:   srv.Image,
		EnvVars: envVars,
		Labels:  map[string]string{"mgw_did": dID, "mgw_iid": iID, "mgw_sref": ref},
		Mounts:  mounts,
		Ports:   ports,
		Networks: []cew_model.ContainerNet{
			{
				Name:        "module-net",
				DomainNames: []string{getSrvName(dID, ref), name},
			},
		},
		RunConfig: cew_model.RunConfig{
			RestartStrategy: cew_model.RestartOnFail,
			Retries:         &retries,
			StopTimeout:     &stopTimeout,
			StopSignal:      srv.RunConfig.StopSignal,
			PseudoTTY:       srv.RunConfig.PseudoTTY,
		},
	}
}
