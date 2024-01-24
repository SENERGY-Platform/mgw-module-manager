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
	"github.com/SENERGY-Platform/go-service-base/context-hdl"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"github.com/SENERGY-Platform/mgw-module-manager/util/naming_hdl"
	"os"
	"path"
	"time"
)

func (h *Handler) Update(ctx context.Context, id string, mod *module.Module, depInput lib_model.DepInput, incl dir_fs.DirFS) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	oldDep, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), id, true, true, true)
	if err != nil {
		return err
	}
	modDependencyDeps, err := h.getModDependencyDeployments(ctx, mod.Dependencies)
	if err != nil {
		return err
	}
	if err = h.stopContainers(ctx, oldDep.Containers); err != nil {
		return err
	}
	if oldDep.Enabled {
		if err = h.unloadSecrets(ctx, oldDep.ID); err != nil {
			return err
		}
	}
	hostResources, secrets, userConfigs, err := h.getDepAssets(ctx, mod, id, depInput)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := h.restore(oldDep); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
	newDep := oldDep
	newDep.Module.Version = mod.Version
	newDep.Name = newDepName(mod.Name, depInput.Name)
	newDep.Updated = time.Now().UTC()
	if incl != "" {
		newDep.Dir, err = h.mkInclDir(incl)
		if err != nil {
			return err
		}
		defer func() {
			var e error
			if err != nil {
				e = os.RemoveAll(path.Join(h.wrkSpcPath, newDep.Dir))
			} else {
				e = os.RemoveAll(path.Join(h.wrkSpcPath, oldDep.Dir))
			}
			if e != nil {
				util.Logger.Error(e)
			}
		}()
	}
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = h.storageHandler.DeleteDepDependencies(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, id); err != nil {
		return err
	}
	if err = h.storageHandler.DeleteDepAssets(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, id); err != nil {
		return err
	}
	if err = h.storageHandler.DeleteDepContainers(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, id); err != nil {
		return err
	}
	if err = h.storageHandler.UpdateDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, newDep.DepBase); err != nil {
		return err
	}
	newDep.RequiredDep = newDepDependencies(modDependencyDeps)
	if err = h.storageHandler.CreateDepDependencies(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, id, newDep.RequiredDep); err != nil {
		return err
	}
	newDep.DepAssets = h.newDepAssets(hostResources, secrets, userConfigs)
	if err = h.storageHandler.CreateDepAssets(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, id, newDep.DepAssets); err != nil {
		return err
	}
	volumes, newVolumes, orphanVolumes, err := h.diffVolumes(ctx, id, mod.Volumes)
	if err != nil {
		return err
	}
	err = h.createVolumes(ctx, newVolumes, id)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			var nv []string
			for _, v := range newVolumes {
				nv = append(nv, v)
			}
			if e := h.removeVolumes(context.Background(), nv, true); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
	newDep.Containers, err = h.createContainers(ctx, mod, newDep.DepBase, userConfigs, hostResources, secrets, modDependencyDeps, oldDep.Containers, volumes)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := h.removeContainers(context.Background(), newDep.Containers, true); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
	if err = h.storageHandler.CreateDepContainers(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, id, newDep.Containers); err != nil {
		return err
	}
	if oldDep.Enabled {
		if err = h.loadSecrets(ctx, newDep); err != nil {
			return err
		}
		if err = h.startContainers(ctx, newDep.Containers); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if e := h.removeContainers(ctx, oldDep.Containers, true); e != nil {
		util.Logger.Error(e)
	}
	if e := h.removeVolumes(ctx, orphanVolumes, true); e != nil {
		util.Logger.Error(e)
	}
	return nil
}

func (h *Handler) diffVolumes(ctx context.Context, dID string, mVolumes map[string]struct{}) (map[string]string, map[string]string, []string, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	cewVolumes, err := h.cewClient.GetVolumes(ctxWt, cew_model.VolumeFilter{Labels: map[string]string{naming_hdl.DeploymentIDLabel: dID}})
	if err != nil {
		return nil, nil, nil, err
	}
	hashVolMap := make(map[string]string)
	hashVolDeprecatedMap := make(map[string]string)
	for mName := range mVolumes {
		hashVolMap[naming_hdl.Global.NewVolumeName(dID, mName)] = mName
		hashVolDeprecatedMap[naming_hdl.NewDeprecatedVolumeName(dID, mName)] = mName
	}
	volumes := make(map[string]string)
	var orphanVolumes []string
	for _, v := range cewVolumes {
		mName, ok := hashVolMap[v.Name]
		if !ok {
			if mName, ok = hashVolDeprecatedMap[v.Name]; !ok {
				orphanVolumes = append(orphanVolumes, v.Name)
				continue
			}
		}
		volumes[mName] = v.Name
	}
	newVolumes := make(map[string]string)
	for hsh, mName := range hashVolMap {
		if _, ok := volumes[mName]; !ok {
			volumes[mName] = hsh
			newVolumes[mName] = hsh
		}
	}
	return volumes, newVolumes, orphanVolumes, nil
}

func (h *Handler) restore(dep lib_model.Deployment) error {
	if dep.Enabled {
		if err := h.unloadSecrets(context.Background(), dep.ID); err != nil {
			return err
		}
		if err := h.loadSecrets(context.Background(), dep); err != nil {
			return err
		}
		if err := h.startContainers(context.Background(), dep.Containers); err != nil {
			return err
		}
	}
	return nil
}
