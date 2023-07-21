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
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"os"
	"path"
	"time"
)

func (h *Handler) Create(ctx context.Context, mod *module.Module, depInput model.DepInput, incl dir_fs.DirFS, indirect bool) (string, error) {
	inclDir, err := h.mkInclDir(incl)
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(path.Join(h.wrkSpcPath, inclDir))
		}
	}()
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	dID, err := h.createDepBase(ctx, tx, mod, depInput, inclDir, indirect)
	if err != nil {
		return "", err
	}
	hostResources, secrets, userConfigs, reqModDepMap, err := h.getDepAssets(ctx, mod, dID, depInput)
	if err != nil {
		return "", err
	}
	err = h.createDepAssets(ctx, tx, mod, dID, hostResources, secrets, userConfigs, reqModDepMap)
	if err != nil {
		return "", err
	}
	// [REMINDER] remove volumes if error
	if err = h.createVolumes(ctx, mod.Volumes, dID); err != nil {
		return "", err
	}
	// [REMINDER] remove containers if error
	_, _, err = h.createInstance(ctx, tx, mod, dID, inclDir, userConfigs, hostResources, secrets, reqModDepMap)
	if err != nil {
		return "", err
	}
	err = tx.Commit()
	if err != nil {
		return "", model.NewInternalError(err)
	}
	return dID, nil
}

func (h *Handler) createDepBase(ctx context.Context, tx driver.Tx, mod *module.Module, depInput model.DepInput, inclDir string, indirect bool) (string, error) {
	timestamp := time.Now().UTC()
	depBase := model.DepBase{
		ModuleID: mod.ID,
		Name:     getDepName(mod.Name, depInput.Name),
		Dir:      inclDir,
		Enabled:  false,
		Indirect: indirect,
		Created:  timestamp,
		Updated:  timestamp,
	}
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	dID, err := h.storageHandler.CreateDep(ctxWt, tx, depBase)
	if err != nil {
		return "", err
	}
	return dID, nil
}
