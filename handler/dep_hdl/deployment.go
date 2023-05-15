/*
 * Copyright 2022 InfAI (CC SES)
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
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/parser"
	"io/fs"
	"os"
	"path"
	"time"
)

type Handler struct {
	storageHandler handler.DepStorageHandler
	cfgVltHandler  handler.CfgValidationHandler
	cewJobHandler  handler.CewJobHandler
	cewClient      client.CewClient
	dbTimeout      time.Duration
	httpTimeout    time.Duration
	wrkSpcPath     string
	perm           fs.FileMode
}

func New(storageHandler handler.DepStorageHandler, cfgVltHandler handler.CfgValidationHandler, cewJobHandler handler.CewJobHandler, cewClient client.CewClient, dbTimeout time.Duration, httpTimeout time.Duration, workspacePath string, perm fs.FileMode) (*Handler, error) {
	if !path.IsAbs(workspacePath) {
		return nil, fmt.Errorf("workspace path must be absolute")
	}
	return &Handler{
		storageHandler: storageHandler,
		cfgVltHandler:  cfgVltHandler,
		cewJobHandler:  cewJobHandler,
		cewClient:      cewClient,
		dbTimeout:      dbTimeout,
		httpTimeout:    httpTimeout,
		wrkSpcPath:     workspacePath,
		perm:           perm,
	}, nil
}

func (h *Handler) InitWorkspace() error {
	if err := os.MkdirAll(h.wrkSpcPath, h.perm); err != nil {
		return err
	}
	return nil
}

func (h *Handler) List(ctx context.Context, filter model.DepFilter) ([]model.DepMeta, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	return h.storageHandler.ListDep(ctxWt, filter)
}

func (h *Handler) Get(ctx context.Context, id string) (*model.Deployment, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	return h.storageHandler.ReadDep(ctxWt, id)
}

func (h *Handler) Update(ctx context.Context, mod *module.Module, dep *model.Deployment, depReq model.DepRequestBase) error {
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
	currentInst, err := h.getCurrentInst(ctx, dep.ID)
	if err != nil {
		return err
	}
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = h.wipeDep(ctx, tx, dep.ID); err != nil {
		return err
	}
	if err = h.storeDep(ctx, tx, dep.ID, hostRes, secrets, mod.Configs, userConfigs); err != nil {
		return err
	}
	_, ctrIDs, err := h.createInstance(ctx, tx, mod, dep.ID, h.getDepDirName(dep.ID), stringValues, hostRes, secrets, reqModDepMap)
	if err != nil {
		return err
	}
	ch := context_hdl.New()
	defer ch.CancelAll()
	if !dep.Stopped {
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
	if err = h.storageHandler.UpdateDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dep.ID, name, dep.Stopped, dep.Indirect, time.Now().UTC()); err != nil {
		return err
	}
	return nil
}

func (h *Handler) getReqDep(ctx context.Context, dep *model.Deployment, reqDep map[string]*model.Deployment) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, dID := range dep.RequiredDep {
		if _, ok := reqDep[dID]; !ok {
			d, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID)
			if err != nil {
				return err
			}
			reqDep[dID] = d
			if err = h.getReqDep(ctx, d, reqDep); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) prepareDep(mod *module.Module, depReq model.DepRequestBase) (name string, userConfigs map[string]any, hostRes, secrets map[string]string, err error) {
	name = getDepName(mod.Name, depReq.Name)
	userConfigs, err = h.getUserConfigs(mod.Configs, depReq.Configs)
	if err != nil {
		return "", nil, nil, nil, model.NewInvalidInputError(err)
	}
	hostRes, err = h.getHostRes(mod.HostResources, depReq.HostResources)
	if err != nil {
		return "", nil, nil, nil, err
	}
	secrets, err = h.getSecrets(mod.Secrets, depReq.Secrets)
	if err != nil {
		return "", nil, nil, nil, err
	}
	return
}

func (h *Handler) getUserConfigs(modConfigs module.Configs, userInput map[string]any) (map[string]any, error) {
	userConfigs, err := parser.UserInputToConfigs(userInput, modConfigs)
	if err != nil {
		return nil, err
	}
	for ref, val := range userConfigs {
		mC := modConfigs[ref]
		if err = h.cfgVltHandler.ValidateValue(mC.Type, mC.TypeOpt, val, mC.IsSlice, mC.DataType); err != nil {
			return nil, err
		}
		if mC.Options != nil && !mC.OptExt {
			if err = h.cfgVltHandler.ValidateValInOpt(mC.Options, val, mC.IsSlice, mC.DataType); err != nil {
				return nil, err
			}
		}
	}
	return userConfigs, nil
}

func (h *Handler) getHostRes(mHostRes map[string]module.HostResource, userInput map[string]string) (map[string]string, error) {
	hostRes, missing, err := getUserHostRes(userInput, mHostRes)
	if err != nil {
		return nil, model.NewInvalidInputError(err)
	}
	if len(missing) > 0 {
		return nil, model.NewInternalError(errors.New("host resource discovery not implemented"))
	}
	return hostRes, nil
}

func (h *Handler) getSecrets(mSecrets map[string]module.Secret, userInput map[string]string) (map[string]string, error) {
	secrets, missing, err := getUserSecrets(userInput, mSecrets)
	if err != nil {
		return nil, model.NewInvalidInputError(err)
	}
	if len(missing) > 0 {
		return nil, model.NewInternalError(errors.New("secret discovery not implemented"))
	}
	return secrets, nil
}

func (h *Handler) storeDep(ctx context.Context, tx driver.Tx, dID string, hostRes, secrets map[string]string, modConfigs module.Configs, userConfigs map[string]any) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	if len(hostRes) > 0 {
		if err := h.storageHandler.CreateDepHostRes(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, hostRes, dID); err != nil {
			return err
		}
	}
	if len(secrets) > 0 {
		if err := h.storageHandler.CreateDepSecrets(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, secrets, dID); err != nil {
			return err
		}
	}
	if len(userConfigs) > 0 {
		if err := h.storageHandler.CreateDepConfigs(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, modConfigs, userConfigs, dID); err != nil {
			return err
		}
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

func (h *Handler) getReqModDepMap(ctx context.Context, reqMod map[string]string) (map[string]string, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	depMap := make(map[string]string)
	for mID := range reqMod {
		depList, err := h.storageHandler.ListDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepFilter{ModuleID: mID})
		if err != nil {
			return nil, err
		}
		if len(depList) == 0 {
			return nil, model.NewInternalError(fmt.Errorf("dependency '%s' not deployed", mID))
		}
		depMap[mID] = depList[0].ID
	}
	return depMap, nil
}

func (h *Handler) getDepDirName(s string) string {
	return path.Join(h.wrkSpcPath, s)
}

func getDepName(mName string, userInput *string) string {
	if userInput != nil {
		return *userInput
	}
	return mName
}

func getSrvName(s, r string) string {
	return "MGW_" + genHash(s, r)
}

func getVolumeName(dID, name string) string {
	return "MGW_" + genHash(dID, name)
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
