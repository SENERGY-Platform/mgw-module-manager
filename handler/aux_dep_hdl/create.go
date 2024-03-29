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

//import (
//	"context"
//	"errors"
//	"fmt"
//	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
//	"github.com/SENERGY-Platform/mgw-module-lib/module"
//	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
//	"github.com/SENERGY-Platform/mgw-module-manager/model"
//	"github.com/SENERGY-Platform/mgw-module-manager/util"
//	"github.com/SENERGY-Platform/go-service-base/context-hdl"
//	"github.com/SENERGY-Platform/mgw-module-manager/util/naming_hdl"
//	"github.com/SENERGY-Platform/mgw-module-manager/util/parser"
//	"path"
//	"regexp"
//	"strings"
//	"time"
//)
//
//func (h *Handler) Create(ctx context.Context, mod model.Module, dep lib_model.Deployment, requiredDep map[string]lib_model.Deployment, auxReq lib_model.AuxDepReq) (string, error) {
//	auxSrv, ok := mod.AuxServices[auxReq.Ref]
//	if !ok {
//		return "", lib_model.NewInvalidInputError(fmt.Errorf("aux service ref '%s' not defined", auxReq.Ref))
//	}
//	ok, err := validImage(mod.AuxImgSrc, auxReq.Image)
//	if err != nil {
//		return "", lib_model.NewInternalError(err)
//	}
//	if !ok {
//		return "", lib_model.NewInvalidInputError(errors.New("image can't be validated"))
//	}
//	timestamp := time.Now().UTC()
//	auxDep := lib_model.AuxDeployment{
//		AuxDepBase: lib_model.AuxDepBase{
//			DepID:   dep.ID,
//			Image:   auxReq.Image,
//			Labels:  auxReq.Labels,
//			Configs: auxReq.Configs,
//			Ref:     auxReq.Ref,
//			Name:    auxSrv.Name,
//			Created: timestamp,
//			Updated: timestamp,
//		},
//	}
//	if auxReq.Name != nil && *auxReq.Name != "" {
//		auxDep.Name = *auxReq.Name
//	}
//	tx, err := h.storageHandler.BeginTransaction(ctx)
//	if err != nil {
//		return "", err
//	}
//	defer tx.Rollback()
//	ch := context_hdl.New()
//	defer ch.CancelAll()
//	auxDep.ID, err = h.storageHandler.CreateAuxDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, auxDep.AuxDepBase)
//	if err != nil {
//		return "", err
//	}
//	volumes := make(map[string]string)
//	for ref := range mod.Volumes {
//		volumes[ref] = naming_hdl.Global.NewVolumeName(dep.ID, ref)
//	}
//	auxDep.Container, err = h.createContainer(ctx, auxSrv, auxDep, mod.Module.Module, dep, requiredDep, volumes)
//	if err != nil {
//		return "", err
//	}
//	if err = h.storageHandler.CreateAuxDepContainer(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, auxDep.ID, auxDep.Container); err != nil {
//		return "", err
//	}
//	err = tx.Commit()
//	if err != nil {
//		return "", lib_model.NewInternalError(err)
//	}
//	return auxDep.ID, nil
//}
//
//func (h *Handler) createContainer(ctx context.Context, auxSrv *module.AuxService, auxDep lib_model.AuxDeployment, mod *module.Module, dep lib_model.Deployment, requiredDep map[string]lib_model.Deployment, volumes map[string]string) (lib_model.AuxDepContainer, error) {
//	globalConfigs, err := getGlobalConfigs(mod.Configs, dep.Configs, auxDep.Configs, auxSrv.Configs)
//	if err != nil {
//		return lib_model.AuxDepContainer{}, lib_model.NewInternalError(err)
//	}
//	envVars, err := getEnvVars(auxSrv, auxDep, globalConfigs, dep.Containers, requiredDep)
//	if err != nil {
//		return lib_model.AuxDepContainer{}, lib_model.NewInternalError(err)
//	}
//	mounts := newMounts(auxSrv, dep.Dir, h.depHostPath, volumes)
//	ctrName, err := naming_hdl.Global.NewContainerName("aux-dep")
//	if err != nil {
//		return lib_model.AuxDepContainer{}, lib_model.NewInternalError(err)
//	}
//	ctrAlias := naming_hdl.Global.NewContainerAlias(auxDep.DepID, auxDep.ID)
//	labels := map[string]string{
//		naming_hdl.CoreIDLabel:           h.coreID,
//		naming_hdl.ManagerIDLabel:        h.managerID,
//		naming_hdl.DeploymentIDLabel:     auxDep.DepID,
//		naming_hdl.AuxDeploymentID:       auxDep.ID,
//		naming_hdl.AuxDeploymentRefLabel: auxDep.Ref,
//	}
//	cewContainer := newCewContainer(auxSrv.RunConfig, auxDep.Image, ctrName, ctrAlias, h.moduleNet, labels, envVars, mounts)
//	auxDepContainer := lib_model.AuxDepContainer{
//		Alias: ctrAlias,
//	}
//	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
//	defer cf()
//	defer func() {
//		if err != nil {
//			if e := h.removeContainer(ctx, auxDepContainer.ID, true); e != nil {
//				util.Logger.Error(e)
//			}
//		}
//	}()
//	if auxDepContainer.ID, err = h.cewClient.CreateContainer(ctxWt, cewContainer); err != nil {
//		return lib_model.AuxDepContainer{}, lib_model.NewInternalError(err)
//	}
//	return auxDepContainer, nil
//}
//
//func getGlobalConfigs(modConfigs module.Configs, depConfigs map[string]lib_model.DepConfig, auxReqConfigs, configMap map[string]string) (map[string]string, error) {
//	configs := make(map[string]string)
//	for refVar, ref := range configMap {
//		modConfig, ok := modConfigs[ref]
//		if !ok {
//			return nil, fmt.Errorf("config '%s' not defined", ref)
//		}
//		if _, ok = auxReqConfigs[refVar]; !ok {
//			depConfig, ok := depConfigs[ref]
//			val := depConfig.Value
//			if !ok {
//				if modConfig.Required {
//					if modConfig.Default != nil {
//						val = modConfig.Default
//					} else {
//						return nil, fmt.Errorf("config '%s' required", ref)
//					}
//				} else {
//					if modConfig.Default != nil {
//						val = modConfig.Default
//					} else {
//						continue
//					}
//				}
//			}
//			var s string
//			var err error
//			if modConfig.IsSlice {
//				s, err = parser.DataTypeToStringList(val, modConfig.Delimiter, modConfig.DataType)
//			} else {
//				s, err = parser.DataTypeToString(val, modConfig.DataType)
//			}
//			if err != nil {
//				return nil, err
//			}
//			configs[refVar] = s
//		}
//	}
//	return configs, nil
//}
//
//func getEnvVars(auxSrv *module.AuxService, auxDep lib_model.AuxDeployment, globalConfigs map[string]string, depContainers map[string]lib_model.DepContainer, requiredDep map[string]lib_model.Deployment) (map[string]string, error) {
//	envVars := make(map[string]string)
//	for refVar, val := range globalConfigs {
//		envVars[refVar] = val
//	}
//	for refVar, val := range auxDep.Configs {
//		envVars[refVar] = val
//	}
//	for refVar, target := range auxSrv.SrvReferences {
//		ctr, ok := depContainers[target.Ref]
//		if !ok {
//			return nil, fmt.Errorf("service '%s' not defined", target.Ref)
//		}
//		envVars[refVar] = target.FillTemplate(ctr.Alias)
//	}
//	for refVar, target := range auxSrv.ExtDependencies {
//		reqDep, ok := requiredDep[target.ID]
//		if !ok {
//			return nil, fmt.Errorf("service '%s' of '%s' not deployed but required", target.Service, target.ID)
//		}
//		depContainer, ok := reqDep.Containers[target.Service]
//		if !ok {
//			return nil, fmt.Errorf("module '%s' service '%s' not defined", target.ID, target.Service)
//		}
//		envVars[refVar] = target.FillTemplate(depContainer.Alias)
//	}
//	envVars[naming_hdl.DeploymentIDEnvVar] = auxDep.DepID
//	envVars[naming_hdl.AuxDeploymentIDEnvVar] = auxDep.ID
//	return envVars, nil
//}
//
//func newMounts(auxSrv *module.AuxService, depDir, depHostPath string, volumes map[string]string) []cew_model.Mount {
//	var mounts []cew_model.Mount
//	for mntPoint, name := range auxSrv.Volumes {
//		if vol, ok := volumes[name]; ok {
//			mounts = append(mounts, cew_model.Mount{
//				Type:   cew_model.VolumeMount,
//				Source: vol,
//				Target: mntPoint,
//			})
//		}
//	}
//	for mntPoint, mount := range auxSrv.BindMounts {
//		mounts = append(mounts, cew_model.Mount{
//			Type:     cew_model.BindMount,
//			Source:   path.Join(depHostPath, depDir, mount.Source),
//			Target:   mntPoint,
//			ReadOnly: mount.ReadOnly,
//		})
//	}
//	for mntPoint, mount := range auxSrv.Tmpfs {
//		mounts = append(mounts, cew_model.Mount{
//			Type:   cew_model.TmpfsMount,
//			Target: mntPoint,
//			Size:   int64(mount.Size),
//			Mode:   mount.Mode,
//		})
//	}
//	return mounts
//}
//
//func newCewContainer(runConfig module.RunConfig, image, name, alias, moduleNet string, labels, envVars map[string]string, mounts []cew_model.Mount) cew_model.Container {
//	retries := int(runConfig.MaxRetries)
//	stopTimeout := runConfig.StopTimeout
//	return cew_model.Container{
//		Name:    name,
//		Image:   image,
//		EnvVars: envVars,
//		Labels:  labels,
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
//			StopSignal:      runConfig.StopSignal,
//			PseudoTTY:       runConfig.PseudoTTY,
//		},
//	}
//}
//
//func validImage(auxImgSrc map[string]struct{}, image string) (bool, error) {
//	for src := range auxImgSrc {
//		s := strings.ReplaceAll(src, ".", "\\.")
//		if strings.Contains(src, "*") {
//			s = strings.ReplaceAll(s, "*", ".+")
//		} else {
//			s = s + "(?:$|:.+$)"
//		}
//		s = "^" + s
//		re, err := regexp.Compile(s)
//		if err != nil {
//			return false, fmt.Errorf("invalid regex pattern '%s'", s)
//		}
//		if re.MatchString(image) {
//			return true, nil
//		}
//	}
//	return false, nil
//}
