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
	"fmt"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	ml_util "github.com/SENERGY-Platform/mgw-module-lib/util"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"github.com/SENERGY-Platform/mgw-module-manager/util/parser"
	"os"
	"path"
	"time"
)

func (h *Handler) Create(ctx context.Context, mod *module.Module, depReq model.DepRequestBase, inclDir dir_fs.DirFS, indirect bool) (string, error) {
	reqModDepMap, err := h.getReqModDepMap(ctx, mod.Dependencies)
	if err != nil {
		return "", err
	}
	name, userConfigs, hostRes, secrets, err := h.prepareDep(mod, depReq)
	if err != nil {
		return "", err
	}
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	ch := context_hdl.New()
	defer ch.CancelAll()
	timestamp := time.Now().UTC()
	dID, err := h.storageHandler.CreateDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, mod.ID, name, indirect, timestamp)
	if err != nil {
		return "", err
	}
	if err = h.storeDep(ctx, tx, dID, hostRes, secrets, mod.Configs, userConfigs); err != nil {
		return "", err
	}
	if len(mod.Dependencies) > 0 {
		var dIDs []string
		for mID := range mod.Dependencies {
			dIDs = append(dIDs, reqModDepMap[mID])
		}
		if err = h.storageHandler.CreateDepReq(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dIDs, dID); err != nil {
			return "", err
		}
	}
	stringValues, err := parser.ConfigsToStringValues(mod.Configs, userConfigs)
	if err != nil {
		return "", err
	}
	depDirPth, err := h.mkDepDir(dID, inclDir)
	if err != nil {
		return "", err
	}
	volumes, err := h.createVolumes(ctx, mod.Volumes, dID)
	if err != nil {
		return "", err
	}
	_, err = h.createInstance(ctx, tx, mod, dID, depDirPth, stringValues, hostRes, secrets, volumes, reqModDepMap)
	if err != nil {
		return "", err
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

func (h *Handler) createVolumes(ctx context.Context, mVolumes ml_util.Set[string], dID string) (map[string]string, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	volumes := make(map[string]string)
	for ref := range mVolumes {
		name, err := h.cewClient.CreateVolume(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cew_model.Volume{
			Name:   getVolumeName(dID, ref),
			Labels: map[string]string{"d_id": dID},
		})
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		volumes[ref] = name
	}
	return volumes, nil
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

func getMounts(srv *module.Service, volumes, hostRes, secrets map[string]string, depDirPth string) []cew_model.Mount {
	var mounts []cew_model.Mount
	for mntPoint, vName := range srv.Volumes {
		mounts = append(mounts, cew_model.Mount{
			Type:   cew_model.VolumeMount,
			Source: volumes[vName],
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
