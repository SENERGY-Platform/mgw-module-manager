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
	"fmt"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	hm_model "github.com/SENERGY-Platform/mgw-host-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"github.com/SENERGY-Platform/mgw-module-manager/util/naming_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/parser"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
	"github.com/google/uuid"
	"os"
	"path"
	"time"
)

func (h *Handler) Create(ctx context.Context, mod *module.Module, depInput lib_model.DepInput, incl dir_fs.DirFS, indirect bool) (string, error) {
	modDependencyDeps, err := h.getModDependencyDeployments(ctx, mod.Dependencies)
	if err != nil {
		return "", err
	}
	inclDir, err := h.mkInclDir(incl)
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			if e := os.RemoveAll(path.Join(h.wrkSpcPath, inclDir)); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
	dep := lib_model.Deployment{DepBase: newDepBase(mod, depInput, inclDir, indirect)}
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return "", err
	}
	defer func() {
		if e := tx.Rollback(); e != nil {
			util.Logger.Error(e)
		}
	}()
	ch := context_hdl.New()
	defer ch.CancelAll()
	if dep.ID, err = h.storageHandler.CreateDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dep.DepBase); err != nil {
		return "", err
	}
	dep.RequiredDep = newDepDependencies(modDependencyDeps)
	if err = h.storageHandler.CreateDepDependencies(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dep.ID, dep.RequiredDep); err != nil {
		return "", err
	}
	hostResources, secrets, userConfigs, err := h.getDepAssets(ctx, mod, dep.ID, depInput)
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			if e := h.unloadSecrets(context.Background(), dep.ID); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
	dep.DepAssets = h.newDepAssets(hostResources, secrets, userConfigs)
	if err = h.storageHandler.CreateDepAssets(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dep.ID, dep.DepAssets); err != nil {
		return "", err
	}
	var volumes []string
	for ref := range mod.Volumes {
		volumes = append(volumes, ref)
	}
	if err = h.createVolumes(ctx, volumes, dep.ID); err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			if e := h.removeVolumes(context.Background(), volumes, true); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
	dep.Containers, err = h.createContainers(ctx, mod, dep.DepBase, userConfigs, hostResources, secrets, modDependencyDeps)
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			if e := h.removeContainers(context.Background(), dep.Containers, true); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
	if err = h.storageHandler.CreateDepContainers(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dep.ID, dep.Containers); err != nil {
		return "", nil
	}
	err = tx.Commit()
	if err != nil {
		return "", lib_model.NewInternalError(err)
	}
	return dep.ID, nil
}

func (h *Handler) createContainers(ctx context.Context, mod *module.Module, depBase lib_model.DepBase, userConfigs map[string]lib_model.DepConfig, hostRes map[string]hm_model.HostResource, secrets map[string]secret, modDependencyDeps map[string]lib_model.Deployment) (map[string]lib_model.DepContainer, error) {
	stringValues, err := userConfigsToStringValues(mod.Configs, userConfigs)
	if err != nil {
		return nil, lib_model.NewInternalError(err)
	}
	cewContainers := make(map[string]cew_model.Container)
	depContainers := make(map[string]lib_model.DepContainer)
	for ref, srv := range mod.Services {
		var envVars map[string]string
		envVars, err = getEnvVars(srv, stringValues, modDependencyDeps, secrets, depBase.ID)
		if err != nil {
			return nil, lib_model.NewInternalError(err)
		}
		mounts, devices := newMounts(srv, depBase, hostRes, secrets, h.depHostPath, h.secHostPath)
		name, err := naming_hdl.Global.NewContainerName("dep")
		if err != nil {
			return nil, err
		}
		alias := naming_hdl.Global.NewContainerAlias(depBase.ID, ref)
		labels := map[string]string{
			naming_hdl.CoreIDLabel:       h.coreID,
			naming_hdl.ManagerIDLabel:    h.managerID,
			naming_hdl.DeploymentIDLabel: depBase.ID,
			naming_hdl.ServiceRefLabel:   ref,
		}
		cewContainers[ref] = newCewContainer(srv, name, alias, h.moduleNet, labels, envVars, mounts, devices, newPorts(srv.Ports))
		depContainers[ref] = lib_model.DepContainer{Alias: alias}
	}
	order, err := sorting.GetSrvOrder(mod.Services)
	if err != nil {
		return nil, lib_model.NewInternalError(err)
	}
	ch := context_hdl.New()
	defer ch.CancelAll()
	defer func() {
		if err != nil {
			if e := h.removeContainers(ctx, depContainers, true); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
	for i, ref := range order {
		cewContainer := cewContainers[ref]
		depContainer := depContainers[ref]
		if depContainer.ID, err = h.cewClient.CreateContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cewContainer); err != nil {
			return nil, lib_model.NewInternalError(err)
		}
		depContainer.Order = uint(i)
		depContainers[ref] = depContainer
	}
	return depContainers, nil
}

func (h *Handler) mkInclDir(inclDir dir_fs.DirFS) (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	strID := id.String()
	p := path.Join(h.wrkSpcPath, strID)
	if err = dir_fs.Copy(inclDir, p); err != nil {
		return "", lib_model.NewInternalError(err)
	}
	return strID, nil
}

func (h *Handler) createVolumes(ctx context.Context, mVolumes []string, dID string) error {
	var err error
	var createdVols []string
	defer func() {
		if err != nil {
			if e := h.removeVolumes(context.Background(), createdVols, true); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, ref := range mVolumes {
		name := naming_hdl.Global.NewVolumeName(dID, ref)
		var n string
		n, err = h.cewClient.CreateVolume(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cew_model.Volume{
			Name:   name,
			Labels: map[string]string{naming_hdl.CoreIDLabel: h.coreID, naming_hdl.ManagerIDLabel: h.managerID, naming_hdl.DeploymentIDLabel: dID},
		})
		if err != nil {
			return lib_model.NewInternalError(err)
		}
		if n != name {
			err = fmt.Errorf("volume name missmatch: %s != %s", n, name)
			return lib_model.NewInternalError(err)
		}
		createdVols = append(createdVols, n)
	}
	return nil
}

func newDepBase(mod *module.Module, depInput lib_model.DepInput, inclDir string, indirect bool) lib_model.DepBase {
	timestamp := time.Now().UTC()
	return lib_model.DepBase{
		Module: lib_model.DepModule{
			ID:      mod.ID,
			Version: mod.Version,
		},
		Name:     newDepName(mod.Name, depInput.Name),
		Dir:      inclDir,
		Indirect: indirect,
		Created:  timestamp,
		Updated:  timestamp,
	}
}

func newDepDependencies(modDependencyDeps map[string]lib_model.Deployment) (dependencies []string) {
	for _, dep := range modDependencyDeps {
		dependencies = append(dependencies, dep.ID)
	}
	return
}

func newMounts(srv *module.Service, depBase lib_model.DepBase, hostRes map[string]hm_model.HostResource, secrets map[string]secret, depHostPath, secHostPath string) ([]cew_model.Mount, []cew_model.Device) {
	var mounts []cew_model.Mount
	var devices []cew_model.Device
	for mntPoint, name := range srv.Volumes {
		mounts = append(mounts, cew_model.Mount{
			Type:   cew_model.VolumeMount,
			Source: naming_hdl.Global.NewVolumeName(depBase.ID, name),
			Target: mntPoint,
		})
	}
	for mntPoint, mount := range srv.BindMounts {
		mounts = append(mounts, cew_model.Mount{
			Type:     cew_model.BindMount,
			Source:   path.Join(depHostPath, depBase.Dir, mount.Source),
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
			if variant, ok := sec.Variants[newSecretVariantID(sec.ID, target.Item)]; ok {
				mounts = append(mounts, cew_model.Mount{
					Type:     cew_model.BindMount,
					Source:   path.Join(secHostPath, variant.Path),
					Target:   mntPoint,
					ReadOnly: true,
				})
			}
		}
	}
	return mounts, devices
}

func getEnvVars(srv *module.Service, configs map[string]string, modDependencyDeps map[string]lib_model.Deployment, secrets map[string]secret, dID string) (map[string]string, error) {
	envVars := make(map[string]string)
	for eVar, cRef := range srv.Configs {
		if val, ok := configs[cRef]; ok {
			envVars[eVar] = val
		}
	}
	for eVar, target := range srv.SrvReferences {
		envVars[eVar] = target.FillTemplate(naming_hdl.Global.NewContainerAlias(dID, target.Ref))
	}
	for eVar, target := range srv.ExtDependencies {
		dep, ok := modDependencyDeps[target.ID]
		if !ok {
			return nil, fmt.Errorf("service '%s' of '%s' not deployed but required", target.Service, target.ID)
		}
		depContainer, ok := dep.Containers[target.Service]
		if !ok {
			return nil, fmt.Errorf("module '%s' service '%s' not defined", target.ID, target.Service)
		}
		envVars[eVar] = target.FillTemplate(depContainer.Alias)
	}
	for eVar, target := range srv.SecretVars {
		if sec, ok := secrets[target.Ref]; ok {
			if variant, ok := sec.Variants[newSecretVariantID(sec.ID, target.Item)]; ok {
				envVars[eVar] = variant.Value
			}
		}
	}
	envVars[naming_hdl.DeploymentIDEnvVar] = dID
	return envVars, nil
}

func newPorts(sPorts []module.Port) (ports []cew_model.Port) {
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

func newCewContainer(srv *module.Service, name, alias, moduleNet string, labels, envVars map[string]string, mounts []cew_model.Mount, devices []cew_model.Device, ports []cew_model.Port) cew_model.Container {
	retries := int(srv.RunConfig.MaxRetries)
	stopTimeout := srv.RunConfig.StopTimeout
	return cew_model.Container{
		Name:    name,
		Image:   srv.Image,
		EnvVars: envVars,
		Labels:  labels,
		Mounts:  mounts,
		Devices: devices,
		Ports:   ports,
		Networks: []cew_model.ContainerNet{
			{
				Name:        moduleNet,
				DomainNames: []string{alias, name},
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

func userConfigsToStringValues(modConfigs module.Configs, userConfigs map[string]lib_model.DepConfig) (map[string]string, error) {
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

func newDepName(mName string, userInput *string) string {
	if userInput != nil && *userInput != "" {
		return *userInput
	}
	return mName
}
