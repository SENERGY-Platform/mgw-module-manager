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
	"github.com/SENERGY-Platform/go-service-base/context-hdl"
	job_hdl_lib "github.com/SENERGY-Platform/go-service-base/job-hdl/lib"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/naming_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/parser"
	"path"
	"regexp"
	"strings"
	"time"
)

func (h *Handler) Create(ctx context.Context, mod model.Module, dep lib_model.Deployment, requiredDep map[string]lib_model.Deployment, auxReq lib_model.AuxDepReq, forcePullImg bool) (string, error) {
	auxSrv, ok := mod.AuxServices[auxReq.Ref]
	if !ok {
		return "", lib_model.NewInvalidInputError(fmt.Errorf("aux service ref '%s' not defined", auxReq.Ref))
	}
	if ok, err := validImage(mod.AuxImgSrc, auxReq.Image); err != nil {
		return "", lib_model.NewInternalError(err)
	} else if !ok {
		return "", lib_model.NewInvalidInputError(errors.New("invalid image"))
	}
	if err := h.pullImage(ctx, auxReq.Image, forcePullImg); err != nil {
		return "", lib_model.NewInternalError(err)
	}
	timestamp := time.Now().UTC()
	auxDep := lib_model.AuxDeployment{
		AuxDepBase: lib_model.AuxDepBase{
			DepID:   dep.ID,
			Image:   auxReq.Image,
			Labels:  auxReq.Labels,
			Configs: auxReq.Configs,
			Volumes: auxReq.Volumes,
			Ref:     auxReq.Ref,
			Name:    auxSrv.Name,
			Created: timestamp,
			Updated: timestamp,
		},
	}
	if auxReq.Name != nil && *auxReq.Name != "" {
		auxDep.Name = *auxReq.Name
	}
	if auxReq.RunConfig != nil {
		auxDep.RunConfig = *auxReq.RunConfig
	}
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	ch := context_hdl.New()
	defer ch.CancelAll()
	auxDep.ID, err = h.storageHandler.CreateAuxDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, auxDep.AuxDepBase)
	if err != nil {
		return "", err
	}
	modVolumes := make(map[string]string)
	for ref := range mod.Volumes {
		modVolumes[ref] = naming_hdl.Global.NewVolumeName(dep.ID, ref)
	}
	auxVolumes := make(map[string]string)
	for ref := range auxDep.Volumes {
		auxVolumes[ref] = naming_hdl.Global.NewVolumeName(auxDep.ID, ref)
	}
	if err = h.createVolumes(ctx, auxVolumes, dep.ID, auxDep.ID); err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			var nv []string
			for _, v := range auxVolumes {
				nv = append(nv, v)
			}
			if e := h.removeVolumes(context.Background(), nv, true); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
	auxDep.Container, err = h.createContainer(ctx, auxSrv, auxDep, mod.Module.Module, dep, requiredDep, modVolumes, auxVolumes)
	if err != nil {
		return "", err
	}
	if err = h.storageHandler.CreateAuxDepContainer(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, auxDep.ID, auxDep.Container); err != nil {
		return "", err
	}
	err = tx.Commit()
	if err != nil {
		return "", lib_model.NewInternalError(err)
	}
	return auxDep.ID, nil
}

func (h *Handler) createContainer(ctx context.Context, auxSrv *module.AuxService, auxDep lib_model.AuxDeployment, mod *module.Module, dep lib_model.Deployment, requiredDep map[string]lib_model.Deployment, modVolumes, auxVolumes map[string]string) (lib_model.AuxDepContainer, error) {
	globalConfigs, err := getGlobalConfigs(mod.Configs, dep.Configs, auxDep.Configs, auxSrv.Configs)
	if err != nil {
		return lib_model.AuxDepContainer{}, lib_model.NewInternalError(err)
	}
	requiredDepModMap := make(map[string]lib_model.Deployment)
	for _, d := range requiredDep {
		requiredDepModMap[d.Module.ID] = d
	}
	envVars, err := getEnvVars(auxSrv, auxDep, globalConfigs, dep.Containers, requiredDepModMap)
	if err != nil {
		return lib_model.AuxDepContainer{}, lib_model.NewInternalError(err)
	}
	mounts := newMounts(auxSrv, dep.Dir, h.depHostPath, auxDep.Volumes, modVolumes, auxVolumes)
	ctrName, err := naming_hdl.Global.NewContainerName("aux-dep")
	if err != nil {
		return lib_model.AuxDepContainer{}, lib_model.NewInternalError(err)
	}
	ctrAlias := naming_hdl.Global.NewContainerAlias(auxDep.DepID, auxDep.ID)
	labels := map[string]string{
		naming_hdl.CoreIDLabel:           h.coreID,
		naming_hdl.ManagerIDLabel:        h.managerID,
		naming_hdl.DeploymentIDLabel:     auxDep.DepID,
		naming_hdl.AuxDeploymentID:       auxDep.ID,
		naming_hdl.AuxDeploymentRefLabel: auxDep.Ref,
	}
	cewContainer := newCewContainer(handleRunConfig(auxSrv.RunConfig, auxDep.RunConfig), auxDep.Image, ctrName, ctrAlias, h.moduleNet, labels, envVars, mounts)
	auxDepContainer := lib_model.AuxDepContainer{
		Alias: ctrAlias,
	}
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	defer func() {
		if err != nil {
			if e := h.removeContainer(ctx, auxDepContainer.ID, true); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
	if auxDepContainer.ID, err = h.cewClient.CreateContainer(ctxWt, cewContainer); err != nil {
		return lib_model.AuxDepContainer{}, lib_model.NewInternalError(err)
	}
	return auxDepContainer, nil
}

func (h *Handler) pullImage(ctx context.Context, img string, alwaysPull bool) error {
	if !alwaysPull {
		ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
		defer cf()
		_, err := h.cewClient.GetImage(ctxWt, img)
		if err != nil {
			var nfe *cew_model.NotFoundError
			if !errors.As(err, &nfe) {
				return lib_model.NewInternalError(err)
			}
		} else {
			return nil
		}
	}
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	jID, err := h.cewClient.AddImage(ctxWt, img)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	job, err := job_hdl_lib.Await(ctx, h.cewClient, jID, time.Second, h.httpTimeout, util.Logger)
	if err != nil {
		return err
	}
	if job.Error != nil {
		return lib_model.NewInternalError(fmt.Errorf("%v", job.Error))
	}
	return nil
}

func (h *Handler) createVolumes(ctx context.Context, volumes map[string]string, dID, aID string) error {
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
	for ref, name := range volumes {
		var n string
		n, err = h.cewClient.CreateVolume(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cew_model.Volume{
			Name:   name,
			Labels: map[string]string{naming_hdl.CoreIDLabel: h.coreID, naming_hdl.ManagerIDLabel: h.managerID, naming_hdl.DeploymentIDLabel: dID, naming_hdl.AuxDeploymentID: aID, naming_hdl.VolumeRefLabel: ref},
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

func getGlobalConfigs(modConfigs module.Configs, depConfigs map[string]lib_model.DepConfig, auxReqConfigs, configMap map[string]string) (map[string]string, error) {
	configs := make(map[string]string)
	for refVar, ref := range configMap {
		modConfig, ok := modConfigs[ref]
		if !ok {
			return nil, fmt.Errorf("config '%s' not defined", ref)
		}
		if _, ok = auxReqConfigs[refVar]; !ok {
			depConfig, ok := depConfigs[ref]
			val := depConfig.Value
			if !ok {
				if modConfig.Required {
					if modConfig.Default != nil {
						val = modConfig.Default
					} else {
						return nil, fmt.Errorf("config '%s' required", ref)
					}
				} else {
					if modConfig.Default != nil {
						val = modConfig.Default
					} else {
						continue
					}
				}
			}
			var s string
			var err error
			if modConfig.IsSlice {
				s, err = parser.DataTypeToStringList(val, modConfig.Delimiter, modConfig.DataType)
			} else {
				s, err = parser.DataTypeToString(val, modConfig.DataType)
			}
			if err != nil {
				return nil, err
			}
			configs[refVar] = s
		}
	}
	return configs, nil
}

func getEnvVars(auxSrv *module.AuxService, auxDep lib_model.AuxDeployment, globalConfigs map[string]string, depContainers map[string]lib_model.DepContainer, requiredDep map[string]lib_model.Deployment) (map[string]string, error) {
	envVars := make(map[string]string)
	for refVar, val := range globalConfigs {
		envVars[refVar] = val
	}
	for refVar, val := range auxDep.Configs {
		envVars[refVar] = val
	}
	for refVar, target := range auxSrv.SrvReferences {
		ctr, ok := depContainers[target.Ref]
		if !ok {
			return nil, fmt.Errorf("service '%s' not defined", target.Ref)
		}
		envVars[refVar] = target.FillTemplate(ctr.Alias)
	}
	for refVar, target := range auxSrv.ExtDependencies {
		reqDep, ok := requiredDep[target.ID]
		if !ok {
			return nil, fmt.Errorf("service '%s' of '%s' not deployed but required", target.Service, target.ID)
		}
		depContainer, ok := reqDep.Containers[target.Service]
		if !ok {
			return nil, fmt.Errorf("module '%s' service '%s' not defined", target.ID, target.Service)
		}
		envVars[refVar] = target.FillTemplate(depContainer.Alias)
	}
	envVars[naming_hdl.DeploymentIDEnvVar] = auxDep.DepID
	envVars[naming_hdl.AuxDeploymentIDEnvVar] = auxDep.ID
	return envVars, nil
}

func newMounts(auxSrv *module.AuxService, depDir, depHostPath string, auxDepVolumes, modVolumes, auxVolumes map[string]string) []cew_model.Mount {
	var mounts []cew_model.Mount
	for mntPoint, name := range auxSrv.Volumes {
		if vol, ok := modVolumes[name]; ok {
			mounts = append(mounts, cew_model.Mount{
				Type:   cew_model.VolumeMount,
				Source: vol,
				Target: mntPoint,
			})
		}
	}
	for name, mntPoint := range auxDepVolumes {
		if vol, ok := auxVolumes[name]; ok {
			mounts = append(mounts, cew_model.Mount{
				Type:   cew_model.VolumeMount,
				Source: vol,
				Target: mntPoint,
			})
		}
	}
	for mntPoint, mount := range auxSrv.BindMounts {
		mounts = append(mounts, cew_model.Mount{
			Type:     cew_model.BindMount,
			Source:   path.Join(depHostPath, depDir, mount.Source),
			Target:   mntPoint,
			ReadOnly: mount.ReadOnly,
		})
	}
	for mntPoint, mount := range auxSrv.Tmpfs {
		mounts = append(mounts, cew_model.Mount{
			Type:   cew_model.TmpfsMount,
			Target: mntPoint,
			Size:   int64(mount.Size),
			Mode:   mount.Mode,
		})
	}
	return mounts
}

func newCewContainer(runConfig module.RunConfig, image, name, alias, moduleNet string, labels, envVars map[string]string, mounts []cew_model.Mount) cew_model.Container {
	retries := int(runConfig.MaxRetries)
	stopTimeout := runConfig.StopTimeout
	return cew_model.Container{
		Name:    name,
		Image:   image,
		EnvVars: envVars,
		Labels:  labels,
		Mounts:  mounts,
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
			StopSignal:      runConfig.StopSignal,
			PseudoTTY:       runConfig.PseudoTTY,
		},
	}
}

func handleRunConfig(rc module.RunConfig, reqRC lib_model.AuxDepRunConfig) module.RunConfig {
	if reqRC.PseudoTTY != rc.PseudoTTY {
		rc.PseudoTTY = reqRC.PseudoTTY
	}
	if reqRC.Command != nil {
		rc.Command = strings.Split(*reqRC.Command, "")
	}
	return rc
}

func validImage(auxImgSrc map[string]struct{}, image string) (bool, error) {
	for src := range auxImgSrc {
		s := strings.ReplaceAll(src, ".", "\\.")
		if strings.Contains(src, "*") {
			s = strings.ReplaceAll(s, "*", ".+")
		} else {
			s = s + "(?:$|:.+$)"
		}
		s = "^" + s
		re, err := regexp.Compile(s)
		if err != nil {
			return false, fmt.Errorf("invalid regex pattern '%s'", s)
		}
		if re.MatchString(image) {
			return true, nil
		}
	}
	return false, nil
}
