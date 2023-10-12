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
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"os"
	"path"
	"time"
)

func (h *Handler) Update(ctx context.Context, id string, mod *module.Module, depInput model.DepInput, incl dir_fs.DirFS) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	oldDep, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), id, true)
	if err != nil {
		return err
	}
	oldDep.Instance, err = h.getDepInstance(ctx, id)
	if err != nil {
		return err
	}
	if oldDep.Enabled {
		if err = h.stopInstance(ctx, oldDep); err != nil {
			return err
		}
		if err = h.unloadSecrets(ctx, id); err != nil {
			return err
		}
	}
	hostResources, secrets, userConfigs, reqModDepMap, err := h.getDepAssets(ctx, mod, id, depInput)
	if err != nil {
		return h.restore(err, oldDep)
	}
	newDep := oldDep
	newDep.Module.Version = mod.Version
	newDep.Name = getDepName(mod.Name, depInput.Name)
	newDep.Updated = time.Now().UTC()
	if incl != "" {
		newDep.Dir, err = h.mkInclDir(incl)
		if err != nil {
			return h.restore(err, oldDep)
		}
		defer func() {
			if err != nil {
				os.RemoveAll(path.Join(h.wrkSpcPath, newDep.Dir))
			} else {
				os.RemoveAll(path.Join(h.wrkSpcPath, oldDep.Dir))
			}
		}()
	}
	newVol, orphanVol, err := h.diffVolumes(ctx, mod.Volumes, id)
	if err != nil {
		return h.restore(err, oldDep)
	}
	err = h.createVolumes(ctx, newVol, id)
	if err != nil {
		return h.restore(err, oldDep)
	}
	defer func() {
		if err != nil {
			h.removeVolumes(context.Background(), newVol)
		}
	}()
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return h.restore(err, oldDep)
	}
	defer tx.Rollback()
	err = h.storageHandler.DeleteDepAssets(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, id)
	if err != nil {
		return h.restore(err, oldDep)
	}
	err = h.storageHandler.UpdateDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, newDep.DepBase)
	if err != nil {
		return h.restore(err, oldDep)
	}
	newDep.DepAssets, err = h.createDepAssets(ctx, tx, mod, id, hostResources, secrets, userConfigs, reqModDepMap)
	if err != nil {
		return h.restore(err, oldDep)
	}
	newDep.Instance, err = h.createInstance(ctx, tx, mod, id, newDep.Dir, userConfigs, hostResources, secrets, reqModDepMap)
	if err != nil {
		return h.restore(err, oldDep)
	}
	defer func() {
		if err != nil {
			h.removeInstance(context.Background(), newDep)
		}
	}()
	if oldDep.Enabled {
		if err = h.startDep(ctx, newDep); err != nil {
			return h.restore(err, oldDep)
		}
	}
	err = tx.Commit()
	if err != nil {
		return h.restore(model.NewInternalError(err), oldDep)
	}
	if err := h.removeInstance(ctx, oldDep); err != nil {
		return err
	}
	if err := h.removeVolumes(ctx, orphanVol); err != nil {
		return err
	}
	return nil
}

func (h *Handler) diffVolumes(ctx context.Context, volumes ml_util.Set[string], dID string) ([]string, []string, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	vols, err := h.cewClient.GetVolumes(ctxWt, cew_model.VolumeFilter{Labels: map[string]string{handler.DeploymentIDLabel: dID}})
	if err != nil {
		return nil, nil, err
	}
	hashedVols := make(map[string]string)
	for name := range volumes {
		hashedVols[getVolumeName(dID, name)] = name
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

func (h *Handler) restore(err error, dep model.Deployment) error {
	if dep.Enabled {
		if e := h.unloadSecrets(context.Background(), dep.ID); e != nil {
			return e
		}
		if e := h.startDep(context.Background(), dep); e != nil {
			return e
		}
	}
	return err
}
