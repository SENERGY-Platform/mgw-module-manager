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
	job_hdl_lib "github.com/SENERGY-Platform/go-service-base/job-hdl/lib"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	hm_model "github.com/SENERGY-Platform/mgw-host-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/parser"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
	"net/http"
	"path"
	"time"
)

func (h *Handler) getDepInstance(ctx context.Context, id string) (model.DepInstance, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	instances, err := h.storageHandler.ListInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepInstFilter{DepID: id})
	if err != nil {
		return model.DepInstance{}, err
	}
	if len(instances) != 1 {
		return model.DepInstance{}, model.NewInternalError(fmt.Errorf("invalid number of instances: %d", len(instances)))
	}
	inst, err := h.storageHandler.ReadInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), instances[0].ID)
	if err != nil {
		return model.DepInstance{}, err
	}
	ctrs, err := h.storageHandler.ListInstCtr(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), inst.ID, model.CtrFilter{SortOrder: model.Ascending})
	if err != nil {
		return model.DepInstance{}, err
	}
	return model.DepInstance{
		ID:         inst.ID,
		Created:    inst.Created,
		Containers: ctrs,
	}, nil
}

func (h *Handler) createInstance(ctx context.Context, tx driver.Tx, mod *module.Module, dID, inclDir string, userConfigs map[string]model.DepConfig, hostRes map[string]hm_model.HostResource, secrets map[string]secret, reqModDepMap map[string]string) (model.DepInstance, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	iID, err := h.storageHandler.CreateInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dID, time.Now().UTC())
	if err != nil {
		return model.DepInstance{}, err
	}
	stringValues, err := userConfigsToStringValues(mod.Configs, userConfigs)
	if err != nil {
		return model.DepInstance{}, model.NewInternalError(err)
	}
	order, err := sorting.GetSrvOrder(mod.Services)
	if err != nil {
		return model.DepInstance{}, model.NewInternalError(err)
	}
	depInstance := model.DepInstance{
		ID:      iID,
		Created: time.Now().UTC(),
	}
	defer func() {
		if err != nil {
			h.removeContainers(context.Background(), depInstance.Containers)
		}
	}()
	for i, ref := range order {
		srv := mod.Services[ref]
		var envVars map[string]string
		envVars, err = getEnvVars(srv, stringValues, reqModDepMap, secrets, dID, iID)
		if err != nil {
			return model.DepInstance{}, model.NewInternalError(err)
		}
		mounts, devices := h.getMounts(srv, hostRes, secrets, dID, inclDir)
		container := getContainer(srv, ref, getSrvName(iID, ref), dID, iID, h.managerID, h.moduleNet, h.coreID, envVars, mounts, devices, getPorts(srv.Ports))
		var cID string
		cID, err = h.cewClient.CreateContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), container)
		if err != nil {
			return model.DepInstance{}, model.NewInternalError(err)
		}
		ctr := model.Container{
			ID:    cID,
			Ref:   ref,
			Order: uint(i),
		}
		depInstance.Containers = append(depInstance.Containers, ctr)
		err = h.storageHandler.CreateInstCtr(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, iID, ctr)
		if err != nil {
			return model.DepInstance{}, err
		}
	}

	return depInstance, nil
}

func (h *Handler) removeInstance(ctx context.Context, dep model.Deployment) error {
	if dep.Instance.ID == "" {
		instance, err := h.getDepInstance(ctx, dep.ID)
		if err != nil {
			return err
		}
		dep.Instance = instance
	}
	if err := h.removeContainers(ctx, dep.Instance.Containers); err != nil {
		return err
	}
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	return h.storageHandler.DeleteInst(ctxWt, dep.Instance.ID)
}

func (h *Handler) removeContainers(ctx context.Context, containers []model.Container) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, ctr := range containers {
		err := h.cewClient.RemoveContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), ctr.ID)
		if err != nil {
			var nfe *cew_model.NotFoundError
			if !errors.As(err, &nfe) {
				return model.NewInternalError(err)
			}
		}
	}
	return nil
}

func (h *Handler) startInstance(ctx context.Context, dep model.Deployment) error {
	if dep.Instance.ID == "" {
		instance, err := h.getDepInstance(ctx, dep.ID)
		if err != nil {
			return err
		}
		dep.Instance = instance
	}
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, ctr := range dep.Instance.Containers {
		err := h.cewClient.StartContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), ctr.ID)
		if err != nil {
			return model.NewInternalError(err)
		}
	}
	return nil
}

func (h *Handler) stopInstance(ctx context.Context, dep model.Deployment) error {
	if dep.Instance.ID == "" {
		instance, err := h.getDepInstance(ctx, dep.ID)
		if err != nil {
			return err
		}
		dep.Instance = instance
	}
	for i := len(dep.Instance.Containers) - 1; i >= 0; i-- {
		if err := h.stopContainer(ctx, dep.Instance.Containers[i].ID); err != nil {
			var nfe *model.NotFoundError
			if !errors.As(err, &nfe) {
				return err
			}
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
	job, err := job_hdl_lib.Await(ctx, h.cewClient, jID, time.Second, h.httpTimeout, util.Logger)
	if err != nil {
		return err
	}
	if job.Error != nil {
		if job.Error.Code != nil && *job.Error.Code == http.StatusNotFound {
			return model.NewNotFoundError(errors.New(job.Error.Message))
		}
		return model.NewInternalError(errors.New(job.Error.Message))
	}
	return nil
}

func (h *Handler) getMounts(srv *module.Service, hostRes map[string]hm_model.HostResource, secrets map[string]secret, dID, inclDir string) ([]cew_model.Mount, []cew_model.Device) {
	var mounts []cew_model.Mount
	var devices []cew_model.Device
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
			Source:   path.Join(h.depHostPath, inclDir, mount.Source),
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
	for mntPoint, target := range srv.HostResources {
		if res, ok := hostRes[target.Ref]; ok {
			// [REMINDER] throw error if type unknown?
			switch res.Type {
			case hm_model.Application:
				mounts = append(mounts, cew_model.Mount{
					Type:     cew_model.BindMount,
					Source:   res.Path,
					Target:   mntPoint,
					ReadOnly: target.ReadOnly,
				})
			case hm_model.SerialDevice:
				devices = append(devices, cew_model.Device{
					Source:   res.Path,
					Target:   mntPoint,
					ReadOnly: target.ReadOnly,
				})
			}
		}
	}
	for mntPoint, target := range srv.SecretMounts {
		if sec, ok := secrets[target.Ref]; ok {
			if variant, ok := sec.Variants[genSecretVariantID(sec.ID, target.Item)]; ok {
				mounts = append(mounts, cew_model.Mount{
					Type:     cew_model.BindMount,
					Source:   path.Join(h.secHostPath, variant.Path),
					Target:   mntPoint,
					ReadOnly: true,
				})
			}
		}
	}
	return mounts, devices
}

func getEnvVars(srv *module.Service, configs, depMap map[string]string, secrets map[string]secret, dID, iID string) (map[string]string, error) {
	envVars := make(map[string]string)
	for eVar, cRef := range srv.Configs {
		if val, ok := configs[cRef]; ok {
			envVars[eVar] = val
		}
	}
	for eVar, target := range srv.SrvReferences {
		envVars[eVar] = target.FillTemplate(getSrvName(dID, target.Ref))
	}
	for eVar, target := range srv.ExtDependencies {
		val, ok := depMap[target.ID]
		if !ok {
			return nil, fmt.Errorf("service '%s' of '%s' not deployed but required", target.Service, target.ID)
		}
		envVars[eVar] = target.FillTemplate(getSrvName(val, target.Service))
	}
	for eVar, target := range srv.SecretVars {
		if sec, ok := secrets[target.Ref]; ok {
			if variant, ok := sec.Variants[genSecretVariantID(sec.ID, target.Item)]; ok {
				envVars[eVar] = variant.Value
			}
		}
	}
	envVars["MGW_DID"] = dID
	envVars["MGW_IID"] = iID
	return envVars, nil
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

func getContainer(srv *module.Service, ref, name, dID, iID, mID, moduleNet, cID string, envVars map[string]string, mounts []cew_model.Mount, devices []cew_model.Device, ports []cew_model.Port) cew_model.Container {
	retries := int(srv.RunConfig.MaxRetries)
	stopTimeout := srv.RunConfig.StopTimeout
	return cew_model.Container{
		Name:    name,
		Image:   srv.Image,
		EnvVars: envVars,
		Labels:  map[string]string{handler.CoreIDLabel: cID, handler.ManagerIDLabel: mID, handler.DeploymentIDLabel: dID, handler.InstanceIDLabel: iID, handler.ServiceRefLabel: ref},
		Mounts:  mounts,
		Devices: devices,
		Ports:   ports,
		Networks: []cew_model.ContainerNet{
			{
				Name:        moduleNet,
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

func getSrvName(s, r string) string {
	return "mgw-inst-" + util.GenHash(s, r)
}

func userConfigsToStringValues(modConfigs module.Configs, userConfigs map[string]model.DepConfig) (map[string]string, error) {
	values := make(map[string]string)
	for ref, mConfig := range modConfigs {
		depConfig, ok := userConfigs[ref]
		val := depConfig.Value
		if !ok {
			if mConfig.Required {
				if mConfig.Default != nil {
					val = mConfig.Default
				} else {
					return nil, fmt.Errorf("config '%s' required", ref)
				}
			} else {
				if mConfig.Default != nil {
					val = mConfig.Default
				} else {
					continue
				}
			}
		}
		var s string
		var err error
		if mConfig.IsSlice {
			s, err = parser.DataTypeToStringList(val, mConfig.Delimiter, mConfig.DataType)
		} else {
			s, err = parser.DataTypeToString(val, mConfig.DataType)
		}
		if err != nil {
			return nil, err
		}
		values[ref] = s
	}
	return values, nil
}
