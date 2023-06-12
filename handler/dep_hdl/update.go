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
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"github.com/SENERGY-Platform/mgw-module-manager/util/parser"
	"time"
)

func (h *Handler) Update(ctx context.Context, mod *module.Module, depReq model.DepRequestBase, incl dir_fs.DirFS, dID, inclDir string, stopped, indirect bool) error {
	reqModDepMap, err := h.getReqModDepMap(ctx, mod.Dependencies)
	if err != nil {
		return err
	}
	name, userConfigs, hostRes, secrets, err := h.prepareDep(mod, depReq)
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
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = h.wipeDep(ctx, tx, dID); err != nil {
		return err
	}
	if err = h.storeDep(ctx, tx, dID, hostRes, secrets, mod.Configs, userConfigs); err != nil {
		return err
	}
	if incl != "" {
		inclDir, err = h.mkInclDir(incl)
		if err != nil {
			return err
		}
	}
	_, ctrIDs, err := h.createInstance(ctx, tx, mod, dID, inclDir, stringValues, hostRes, secrets, reqModDepMap)
	if err != nil {
		return err
	}
	defer func() {
		ch := context_hdl.New()
		if err != nil {
			for _, cID := range ctrIDs {
				if !stopped {
					_ = h.stopContainer(ctx, cID)
				}
				_ = h.cewClient.RemoveContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cID)
			}
			if !stopped {
				_ = h.startInstance(ctx, currentInst.ID)
			}
		}
		ch.CancelAll()
	}()
	ch := context_hdl.New()
	defer ch.CancelAll()
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
	return nil
}
