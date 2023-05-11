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
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	ml_util "github.com/SENERGY-Platform/mgw-module-lib/util"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"github.com/SENERGY-Platform/mgw-module-manager/util/parser"
	"github.com/SENERGY-Platform/mgw-module-manager/util/sorting"
	"os"
	"path"
	"time"
)

func (h *Handler) Create(ctx context.Context, mod *module.Module, depReq model.DepRequestBase, inclDir dir_fs.DirFS, indirect bool) (string, error) {
	depMap, err := h.getDepMap(ctx, mod.Dependencies)
	if err != nil {
		return "", err
	}
	configs, userConfigs, err := h.getConfigs(mod.Configs, depReq.Configs)
	if err != nil {
		return "", err
	}
	hostRes, err := h.getHostRes(mod.HostResources, depReq.HostResources)
	if err != nil {
		return "", err
	}
	secrets, err := h.getSecrets(mod.Secrets, depReq.Secrets)
	if err != nil {
		return "", err
	}
	name := getName(mod.Name, depReq.Name)
	timestamp := time.Now().UTC()
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	ch := context_hdl.New()
	defer ch.CancelAll()
	dID, err := h.storageHandler.CreateDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, mod.ID, name, indirect, timestamp)
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
		if err = h.storageHandler.CreateDepConfigs(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, mod.Configs, userConfigs, dID); err != nil {
			return "", err
		}
	}
	if len(mod.Dependencies) > 0 {
		var dr []string
		for rmID := range mod.Dependencies {
			dr = append(dr, depMap[rmID])
		}
		if err = h.storageHandler.CreateDepReq(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dr, dID); err != nil {
			return "", err
		}
	}
	iID, err := h.storageHandler.CreateInst(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dID, timestamp)
	if err != nil {
		return "", err
	}
	depDirPth, err := h.mkDepDir(dID, inclDir)
	if err != nil {
		return "", err
	}
	volumes, err := h.createVolumes(ctx, mod.Volumes, dID, iID)
	order, err := sorting.GetSrvOrder(mod.Services)
	if err != nil {
		return "", model.NewInternalError(err)
	}
	for i := 0; i < len(order); i++ {
		cID, err := h.createContainer(ctx, mod.Services[order[i]], order[i], dID, iID, depDirPth, configs, volumes, depMap, hostRes, secrets)
		if err != nil {
			return "", err
		}
		err = h.storageHandler.CreateInstCtr(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, iID, cID, order[i], uint(i))
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

func (h *Handler) mkDepDir(dID string, inclDir dir_fs.DirFS) (string, error) {
	p := path.Join(h.wrkSpcPath, dID)
	if err := util.CopyDir(inclDir.Path(), p); err != nil {
		_ = os.RemoveAll(p)
		return "", model.NewInternalError(err)
	}
	return p, nil
}

func (h *Handler) getDepMap(ctx context.Context, mDependencies map[string]string) (map[string]string, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	depMap := make(map[string]string)
	for rmID := range mDependencies {
		depList, err := h.storageHandler.ListDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepFilter{ModuleID: rmID})
		if err != nil {
			return nil, err
		}
		if len(depList) == 0 {
			return nil, model.NewInternalError(fmt.Errorf("dependency '%s' not deployed", rmID))
		}
		depMap[rmID] = depList[0].ID
	}
	return depMap, nil
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

func (h *Handler) createVolumes(ctx context.Context, mVolumes ml_util.Set[string], dID, iID string) (map[string]string, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	volumes := make(map[string]string)
	for ref := range mVolumes {
		name, err := h.cewClient.CreateVolume(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cew_model.Volume{
			Name:   getVolumeName(iID, ref),
			Labels: map[string]string{"d_id": dID, "i_id": iID},
		})
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		volumes[ref] = name
	}
	return volumes, nil
}

func (h *Handler) getDeployments(ctx context.Context, modules map[string]*module.Module, deployments map[string]string) error {
	ch := context_hdl.New()
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

func getVolumeName(s, v string) string {
	return "MGW_" + genHash(s, v)
}

func getSrvName(s, r string) string {
	return "MGW_" + genHash(s, r)
}

func genHash(str ...string) string {
	hash := sha1.New()
	for _, s := range str {
		hash.Write([]byte(s))
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash.Sum(nil))
}

func getConfigsWithDefaults(mConfigs module.Configs, dConfigs map[string]any) (map[string]string, error) {
	envVals := make(map[string]string)
	for ref, mConfig := range mConfigs {
		val, ok := dConfigs[ref]
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
			s, err = parser.ToStringList(val, mConfig.Delimiter, mConfig.DataType)
		} else {
			s, err = parser.ToString(val, mConfig.DataType)
		}
		if err != nil {
			return nil, err
		}
		envVals[ref] = s
	}
	return envVals, nil
}

func getUserHostRes(hrs map[string]string, mHRs map[string]module.HostResource) (map[string]string, []string, error) {
	dRs := make(map[string]string)
	var ad []string
	for ref, mRH := range mHRs {
		id, ok := hrs[ref]
		if ok {
			dRs[ref] = id
		} else {
			if mRH.Required {
				if len(mRH.Tags) > 0 {
					ad = append(ad, ref)
				} else {
					return nil, nil, fmt.Errorf("host resource '%s' required", ref)
				}
			}
		}
	}
	return dRs, ad, nil
}

func getUserSecrets(s map[string]string, mSs map[string]module.Secret) (map[string]string, []string, error) {
	dSs := make(map[string]string)
	var ad []string
	for ref, mS := range mSs {
		id, ok := s[ref]
		if ok {
			dSs[ref] = id
		} else {
			if mS.Required {
				if len(mS.Tags) > 0 {
					ad = append(ad, ref)
				} else {
					return nil, nil, fmt.Errorf("secret '%s' required", ref)
				}
			}
		}
	}
	return dSs, ad, nil
}

func getUserConfigs(cfgs map[string]any, mCs module.Configs) (map[string]any, error) {
	dCs := make(map[string]any)
	for ref, mC := range mCs {
		val, ok := cfgs[ref]
		if !ok {
			if mC.Default == nil && mC.Required {
				return nil, fmt.Errorf("config '%s' requried", ref)
			}
		} else {
			var v any
			var err error
			if mC.IsSlice {
				v, err = parser.ToDataTypeSlice(val, mC.DataType)
			} else {
				v, err = parser.ToDataType(val, mC.DataType)
			}
			if err != nil {
				return nil, fmt.Errorf("parsing config '%s' failed: %s", ref, err)
			}
			dCs[ref] = v
		}
	}
	return dCs, nil
}
