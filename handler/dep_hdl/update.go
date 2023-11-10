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
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	ml_util "github.com/SENERGY-Platform/mgw-module-lib/util"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"github.com/SENERGY-Platform/mgw-module-manager/util/naming_hdl"
	"os"
	"path"
	"time"
)

func (h *Handler) Update(ctx context.Context, id string, mod *module.Module, depInput model.DepInput, incl dir_fs.DirFS) error {
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
	defer func() {
		if e := tx.Rollback(); e != nil {
			util.Logger.Error(e)
		}
	}()
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
	newVol, orphanVol, err := h.diffVolumes(ctx, mod.Volumes, id)
	if err != nil {
		return err
	}
	err = h.createVolumes(ctx, newVol, id)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := h.removeVolumes(context.Background(), newVol, true); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
	newDep.Containers, err = h.createContainers(ctx, mod, newDep.DepBase, userConfigs, hostResources, secrets, modDependencyDeps)
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
	err = tx.Commit()
	if err != nil {
		return err
	}
	if e := h.removeContainers(ctx, oldDep.Containers, true); e != nil {
		util.Logger.Error(e)
	}
	if e := h.removeVolumes(ctx, orphanVol, true); e != nil {
		util.Logger.Error(e)
	}
	return nil
}

func (h *Handler) diffVolumes(ctx context.Context, volumes ml_util.Set[string], dID string) ([]string, []string, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	vols, err := h.cewClient.GetVolumes(ctxWt, cew_model.VolumeFilter{Labels: map[string]string{naming_hdl.DeploymentIDLabel: dID}})
	if err != nil {
		return nil, nil, err
	}
	hashedVols := make(map[string]string)
	for name := range volumes {
		hashedVols[naming_hdl.Global.NewVolumeName(dID, name)] = name
	}
	var orphans []string
	existing := make(ml_util.Set[string])
	for _, v := range vols {
		if _, ok := hashedVols[v.Name]; !ok {
			orphans = append(orphans, v.Name)
		} else {
			existing[v.Name] = struct{}{}
		}
	}
	var missing []string
	for hsh, name := range hashedVols {
		if _, ok := existing[hsh]; !ok {
			missing = append(missing, name)
		}
	}
	return missing, orphans, nil
}

func (h *Handler) restore(dep model.Deployment) error {
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
