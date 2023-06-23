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
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"github.com/SENERGY-Platform/mgw-module-manager/util/parser"
	"github.com/google/uuid"
	"io/fs"
	"os"
	"path"
	"time"
)

type Handler struct {
	storageHandler handler.DepStorageHandler
	cfgVltHandler  handler.CfgValidationHandler
	cewClient      client.CewClient
	dbTimeout      time.Duration
	httpTimeout    time.Duration
	wrkSpcPath     string
	depHostPath    string
}

func New(storageHandler handler.DepStorageHandler, cfgVltHandler handler.CfgValidationHandler, cewClient client.CewClient, dbTimeout time.Duration, httpTimeout time.Duration, workspacePath, depHostPath string) *Handler {
	return &Handler{
		storageHandler: storageHandler,
		cfgVltHandler:  cfgVltHandler,
		cewClient:      cewClient,
		dbTimeout:      dbTimeout,
		httpTimeout:    httpTimeout,
		wrkSpcPath:     workspacePath,
		depHostPath:    depHostPath,
	}
}

func (h *Handler) InitWorkspace(perm fs.FileMode) error {
	if !path.IsAbs(h.wrkSpcPath) {
		return fmt.Errorf("workspace path must be absolute")
	}
	if err := os.MkdirAll(h.wrkSpcPath, perm); err != nil {
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

func (h *Handler) prepareDep(mod *module.Module, depReq model.DepInput) (name string, userConfigs map[string]any, hostRes, secrets map[string]string, err error) {
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

func (h *Handler) storeDepAssets(ctx context.Context, tx driver.Tx, dID string, hostRes, secrets map[string]string, modConfigs module.Configs, userConfigs map[string]any) error {
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

func (h *Handler) mkInclDir(inclDir dir_fs.DirFS) (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	strID := id.String()
	p := path.Join(h.wrkSpcPath, strID)
	if err = dir_fs.Copy(inclDir, p); err != nil {
		return "", model.NewInternalError(err)
	}
	return strID, nil
}

func (h *Handler) removeVolumes(ctx context.Context, volumes []string) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, name := range volumes {
		err := h.cewClient.RemoveVolume(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), name)
		if err != nil {
			var nfe *cew_model.NotFoundError
			if !errors.As(err, &nfe) {
				return err
			}
		}
	}
	return nil
}

func getDepName(mName string, userInput *string) string {
	if userInput != nil && *userInput != "" {
		return *userInput
	}
	return mName
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
