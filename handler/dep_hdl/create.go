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
	"github.com/SENERGY-Platform/mgw-module-manager/util/parser"
	"os"
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
	p := h.getDepDirName(dID)
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

func getVolumeName(s, v string) string {
	return "MGW_" + genHash(s, v)
}
