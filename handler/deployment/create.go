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

package deployment

import (
	"context"
	"errors"
	"fmt"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	ml_util "github.com/SENERGY-Platform/mgw-module-lib/util"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/ctx_handler"
	"path"
	"time"
)

func (h *Handler) Create(ctx context.Context, dr model.DepRequest) (string, error) {
	m, dms, err := h.moduleHandler.GetWithDep(ctx, dr.ModuleID)
	if err != nil {
		return "", err
	}
	ch := ctx_handler.New()
	defer ch.CancelAll()
	if m.DeploymentType == module.SingleDeployment {
		if l, err := h.storageHandler.ListDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepFilter{ModuleID: m.ID}); err != nil {
			return "", err
		} else if len(l) > 0 {
			return "", model.NewInvalidInputError(errors.New("already deployed"))
		}
	}
	depMap := make(map[string]string)
	if len(dms) > 0 {
		for dmID := range dms {
			if l, err := h.storageHandler.ListDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepFilter{ModuleID: dmID}); err != nil {
				return "", err
			} else if len(l) > 0 {
				depMap[dmID] = l[0].ID
			}
		}
		order, err := getModOrder(dms)
		if err != nil {
			return "", model.NewInternalError(err)
		}
		var depNew []string
		for _, dmID := range order {
			if _, ok := depMap[dmID]; !ok {
				dID, err := h.create(ctx, dms[dmID], dr.Dependencies[dmID], depMap, true)
				if err != nil {
					//for _, id := range depNew {
					//	h.Delete(ctx, id)
					//}
					return "", err
				}
				depMap[dmID] = dID
				depNew = append(depNew, dID)
			}
		}
	}
	return h.create(ctx, m, dr.DepRequestBase, depMap, false)
}

func (h *Handler) validateConfigs(dCs map[string]any, mCs module.Configs) error {
	for ref, val := range dCs {
		mC := mCs[ref]
		if err := h.cfgVltHandler.ValidateValue(mC.Type, mC.TypeOpt, val, mC.IsSlice, mC.DataType); err != nil {
			return model.NewInvalidInputError(err)
		}
		if mC.Options != nil && !mC.OptExt {
			if err := h.cfgVltHandler.ValidateValInOpt(mC.Options, val, mC.IsSlice, mC.DataType); err != nil {
				return model.NewInvalidInputError(err)
			}
		}
	}
	return nil
}

func (h *Handler) getConfigs(mConfigs module.Configs, userInput map[string]any) (map[string]string, map[string]any, error) {
	userConfigs, err := getUserConfigs(userInput, mConfigs)
	if err != nil {
		return nil, nil, model.NewInvalidInputError(err)
	}
	if err = h.validateConfigs(userConfigs, mConfigs); err != nil {
		return nil, nil, err
	}
	configs, err := getConfigsWithDefaults(mConfigs, userConfigs)
	if err != nil {
		return nil, nil, model.NewInvalidInputError(err)
	}
	return configs, userConfigs, nil
}

func (h *Handler) getHostRes(mHostRes map[string]module.HostResource, userInput map[string]string) (map[string]string, error) {
	hostRes, missing, err := getUserHostRes(userInput, mHostRes)
	if err != nil {
		return nil, model.NewInvalidInputError(err)
	}
	if len(missing) > 0 {
		return nil, model.NewInternalError(errors.New("host resource discovery not implemented"))
	}
	return hostRes, nil
}

func (h *Handler) getSecrets(mSecrets map[string]module.Secret, userInput map[string]string) (map[string]string, error) {
	secrets, missing, err := getUserSecrets(userInput, mSecrets)
	if err != nil {
		return nil, model.NewInvalidInputError(err)
	}
	if len(missing) > 0 {
		return nil, model.NewInternalError(errors.New("secret discovery not implemented"))
	}
	return secrets, nil
}

func (h *Handler) createVolume(ctx context.Context, dID, iID, v string) (string, error) {
	httpCtx, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	vName, err := h.cewClient.CreateVolume(httpCtx, cew_model.Volume{
		Name:   getVolumeName(iID, v),
		Labels: map[string]string{"d_id": dID, "i_id": iID},
	})
	if err != nil {
		return "", model.NewInternalError(err)
	}
	return vName, nil
}

func (h *Handler) getVolumes(ctx context.Context, mVolumes ml_util.Set[string], dID, iID string) (map[string]string, error) {
	volumes := make(map[string]string)
	for ref := range mVolumes {
		name, err := h.createVolume(ctx, dID, iID, ref)
		if err != nil {
			return nil, err
		}
		volumes[ref] = name
	}
	return volumes, nil
}

func (h *Handler) getDeployments(ctx context.Context, modules map[string]*module.Module, deployments map[string]string) error {
	ch := ctx_handler.New()
	defer ch.CancelAll()
	for mID := range modules {
		ds, err := h.storageHandler.ListDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepFilter{ModuleID: mID})
		if err != nil {
			return err
		}
		if len(ds) > 0 {
			deployments[mID] = ds[0].ID
		}
	}
	return nil
}

func (h *Handler) create(ctx context.Context, m *module.Module, drb model.DepRequestBase, depMap map[string]string, indirect bool) (string, error) {
	configs, userConfigs, err := h.getConfigs(m.Configs, drb.Configs)
	if err != nil {
		return "", err
	}
	hostRes, err := h.getHostRes(m.HostResources, drb.HostResources)
	if err != nil {
		return "", err
	}
	secrets, err := h.getSecrets(m.Secrets, drb.Secrets)
	if err != nil {
		return "", err
	}
	name := getName(m.Name, drb.Name)
	timestamp := time.Now().UTC()
	ch := ctx_handler.New()
	defer ch.CancelAll()
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	dID, err := h.storageHandler.CreateDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, m.ID, name, indirect, timestamp)
	if err != nil {
		return "", err
	}
	if len(hostRes) > 0 {
		if err = h.storageHandler.CreateDepHostRes(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, hostRes, dID); err != nil {
			return "", err
		}
	}
	if len(secrets) > 0 {
		if err = h.storageHandler.CreateDepSecrets(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, secrets, dID); err != nil {
			return "", err
		}
	}
	if len(userConfigs) > 0 {
		if err = h.storageHandler.CreateDepConfigs(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, m.Configs, userConfigs, dID); err != nil {
			return "", err
		}
	}
	if len(m.Dependencies) > 0 {
		var depReq []string
		for rmID := range m.Dependencies {
			depReq = append(depReq, depMap[rmID])
		}
		if err = h.storageHandler.CreateDepReq(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, depReq, dID); err != nil {
			return "", err
		}
	}
	iID, err := h.storageHandler.CreateInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dID, timestamp)
	if err != nil {
		return "", err
	}
	inclDirPath, err := h.moduleHandler.CreateInclDir(ctx, m.ID, dID)
	if err != nil {
		return "", err
	}
	volumes, err := h.getVolumes(ctx, m.Volumes, dID, iID)
	order, err := getSrvOrder(m.Services)
	if err != nil {
		return "", model.NewInternalError(err)
	}
	for _, ref := range order {
		cID, err := h.createContainer(ctx, m.Services[ref], ref, dID, iID, inclDirPath, configs, volumes, depMap, hostRes, secrets)
		if err != nil {
			return "", err
		}
		err = h.storageHandler.CreateInstCtr(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, iID, cID, ref)
		if err != nil {
			return "", err
		}
	}
	err = tx.Commit()
	if err != nil {
		return "", model.NewInternalError(err)
	}
	return dID, nil
}

func (h *Handler) createContainer(ctx context.Context, srv *module.Service, ref, dID, iID, inclDirPath string, configs, volumes, depMap, hostRes, secrets map[string]string) (string, error) {
	envVars, err := getEnvVars(srv, configs, depMap, dID, iID)
	if err != nil {
		return "", model.NewInternalError(err)
	}
	mounts := getMounts(srv, volumes, inclDirPath, dID, iID)
	ports := getPorts(srv.Ports)
	name := getSrvName(iID, ref)
	retries := int(srv.RunConfig.MaxRetries)
	stopTimeout := srv.RunConfig.StopTimeout
	c := cew_model.Container{
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
	httpCtx, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	cID, err := h.cewClient.CreateContainer(httpCtx, c)
	if err != nil {
		return "", model.NewInternalError(err)
	}
	return cID, nil
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

func getMounts(srv *module.Service, volumes map[string]string, inclDirPath, dID, iID string) []cew_model.Mount {
	var mounts []cew_model.Mount
	vLabels := map[string]string{"mgw_did": dID, "mgw_iid": iID}
	for mntPoint, vName := range srv.Volumes {
		mounts = append(mounts, cew_model.Mount{
			Type:   cew_model.VolumeMount,
			Source: volumes[vName],
			Target: mntPoint,
			Labels: vLabels,
		})
	}
	for mntPoint, mount := range srv.BindMounts {
		mounts = append(mounts, cew_model.Mount{
			Type:     cew_model.BindMount,
			Source:   path.Join(inclDirPath, mount.Source),
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
	//	src, ok := hostRes[sRef]
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

func getName(mName string, userInput *string) string {
	if userInput != nil {
		return *userInput
	}
	return mName
}