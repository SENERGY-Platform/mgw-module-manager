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
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	ml_util "github.com/SENERGY-Platform/mgw-module-lib/util"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"github.com/SENERGY-Platform/mgw-module-manager/util/parser"
	"os"
	"path"
	"time"
)

func (h *Handler) Update(ctx context.Context, mod *module.Module, depInput model.DepInput, incl dir_fs.DirFS, dID, inclDir string, stopped, indirect bool) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	reqModDepMap, err := h.getReqModDepMap(ctx, mod.Dependencies)
	if err != nil {
		return err
	}
	name, userConfigs, hostRes, secrets, err := h.prepareDep(mod, depInput)
	if err != nil {
		return err
	}
	stringValues, err := parser.ConfigsToStringValues(mod.Configs, userConfigs)
	if err != nil {
		return err
	}
	currentInst, err := h.getCurrentInst(ctx, dID)
	if err != nil {
		return err
	}
	missingVol, orphanVol, err := h.diffVolumes(ctx, mod.Volumes, dID)
	if err != nil {
		return err
	}
	// [REMINDER] remove new volumes if error
	err = h.createVolumes(ctx, missingVol, dID)
	if err != nil {
		return err
	}
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = h.wipeDep(ctx, tx, dID); err != nil {
		return err
	}
	if err = h.storeDepAssets(ctx, tx, dID, hostRes, secrets, mod.Configs, userConfigs); err != nil {
		return err
	}
	if len(mod.Dependencies) > 0 {
		var dIDs []string
		for mID := range mod.Dependencies {
			dIDs = append(dIDs, reqModDepMap[mID])
		}
		if err = h.storageHandler.CreateDepReq(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dIDs, dID); err != nil {
			return err
		}
	}
	if incl != "" {
		inclDir, err = h.mkInclDir(incl)
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				os.RemoveAll(path.Join(h.wrkSpcPath, inclDir))
			}
		}()
	}
	var ctrIDs []string
	_, ctrIDs, err = h.createInstance(ctx, tx, mod, dID, inclDir, stringValues, hostRes, secrets, reqModDepMap)
	if err != nil {
		return err
	}
	defer func() {
		ch2 := context_hdl.New()
		if err != nil {
			for _, cID := range ctrIDs {
				if !stopped {
					_ = h.stopContainer(ctx, cID)
				}
				_ = h.cewClient.RemoveContainer(ch2.Add(context.WithTimeout(ctx, h.httpTimeout)), cID)
			}
			if !stopped {
				_ = h.startInstance(ctx, currentInst.ID)
			}
		}
		ch2.CancelAll()
	}()
	if !stopped {
		if err = h.stopInstance(ctx, currentInst.ID); err != nil {
			return err
		}
		for _, cID := range ctrIDs {
			err = h.cewClient.StartContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cID)
			if err != nil {
				return model.NewInternalError(err)
			}
		}
	}
	if err = h.removeInstance(ctx, currentInst.ID); err != nil {
		// [REMINDER] can lead to state with no running instances
		return err
	}
	if err = h.removeVolumes(ctx, orphanVol); err != nil {
		// [REMINDER] can lead to inconsistent state
		return err
	}
	err = tx.Commit()
	if err != nil {
		return model.NewInternalError(err)
	}
	if err = h.storageHandler.UpdateDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID, name, inclDir, stopped, indirect, time.Now().UTC()); err != nil {
		return err
	}
	return nil
}

func (h *Handler) wipeDep(ctx context.Context, tx driver.Tx, dID string) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	if err := h.storageHandler.DeleteDepHostRes(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dID); err != nil {
		return err
	}
	if err := h.storageHandler.DeleteDepSecrets(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dID); err != nil {
		return err
	}
	if err := h.storageHandler.DeleteDepConfigs(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dID); err != nil {
		return err
	}
	if err := h.storageHandler.DeleteDepReq(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dID); err != nil {
		return err
	}
	return nil
}

func (h *Handler) diffVolumes(ctx context.Context, volumes ml_util.Set[string], dID string) (ml_util.Set[string], []string, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	vols, err := h.cewClient.GetVolumes(ctxWt, cew_model.VolumeFilter{Labels: map[string]string{"d_id": dID}})
	if err != nil {
		return nil, nil, err
	}
	var orphans []string
	existing := make(ml_util.Set[string])
	for _, v := range vols {
		if _, ok := volumes[v.Name]; !ok {
			orphans = append(orphans, v.Name)
		}
		existing[v.Name] = struct{}{}
	}
	missing := make(ml_util.Set[string])
	for name := range volumes {
		if _, ok := existing[name]; !ok {
			missing[name] = struct{}{}
		}
	}
	return missing, orphans, nil
}
