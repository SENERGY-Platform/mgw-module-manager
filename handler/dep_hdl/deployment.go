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
	"encoding/hex"
	"errors"
	"fmt"
	cew_client "github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	hm_client "github.com/SENERGY-Platform/mgw-host-manager/client"
	hm_model "github.com/SENERGY-Platform/mgw-host-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"github.com/SENERGY-Platform/mgw-module-manager/util/parser"
	sm_model "github.com/SENERGY-Platform/mgw-secret-manager/pkg/api_model"
	sm_client "github.com/SENERGY-Platform/mgw-secret-manager/pkg/client"
	"github.com/google/uuid"
	"io/fs"
	"os"
	"path"
	"time"
)

type Handler struct {
	storageHandler handler.DepStorageHandler
	cfgVltHandler  handler.CfgValidationHandler
	cewClient      cew_client.CewClient
	hmClient       hm_client.HmClient
	smClient       sm_client.Client
	dbTimeout      time.Duration
	httpTimeout    time.Duration
	wrkSpcPath     string
	depHostPath    string
	secHostPath    string
}

func New(storageHandler handler.DepStorageHandler, cfgVltHandler handler.CfgValidationHandler, cewClient cew_client.CewClient, hmClient hm_client.HmClient, smClient sm_client.Client, dbTimeout time.Duration, httpTimeout time.Duration, workspacePath, depHostPath, secHostPath string) *Handler {
	return &Handler{
		storageHandler: storageHandler,
		cfgVltHandler:  cfgVltHandler,
		cewClient:      cewClient,
		hmClient:       hmClient,
		smClient:       smClient,
		dbTimeout:      dbTimeout,
		httpTimeout:    httpTimeout,
		wrkSpcPath:     workspacePath,
		depHostPath:    depHostPath,
		secHostPath:    secHostPath,
	}
}

type secretVariant struct {
	Item  *string
	Path  string
	AsEnv bool
	Value string
}

type secret struct {
	ID       string
	Variants map[string]secretVariant
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

func (h *Handler) List(ctx context.Context, filter model.DepFilter) ([]model.DepBase, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	return h.storageHandler.ListDep(ctxWt, filter)
}

func (h *Handler) Get(ctx context.Context, id string) (*model.Deployment, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	return h.storageHandler.ReadDep(ctxWt, id)
}

func (h *Handler) ListInstances(ctx context.Context) (map[string]model.DepInstance, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	listDep, err := h.storageHandler.ListDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), model.DepFilter{})
	if err != nil {
		return nil, err
	}
	depInstances := make(map[string]model.DepInstance)
	for _, dep := range listDep {
		inst, err := h.getCurrentInst(ctx, dep.ID)
		if err != nil {
			return nil, err
		}
		ctrs, err := h.storageHandler.ListInstCtr(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), inst.ID, model.CtrFilter{SortOrder: model.Descending})
		if err != nil {
			return nil, err
		}
		depInstances[dep.ID] = model.DepInstance{
			ID:         inst.ID,
			Created:    inst.Created,
			Containers: ctrs,
		}
	}
	return depInstances, nil
}

func (h *Handler) GetInstance(ctx context.Context, id string) (model.DepInstance, error) {
	inst, err := h.getCurrentInst(ctx, id)
	if err != nil {
		return model.DepInstance{}, err
	}
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	ctrs, err := h.storageHandler.ListInstCtr(ctxWt, inst.ID, model.CtrFilter{SortOrder: model.Descending})
	if err != nil {
		return model.DepInstance{}, err
	}
	return model.DepInstance{
		ID:         inst.ID,
		Created:    inst.Created,
		Containers: ctrs,
	}, nil
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

func (h *Handler) prepareDep(ctx context.Context, mod *module.Module, dID string, depReq model.DepInput) (userConfigs map[string]any, hostRes map[string]hm_model.HostResource, secrets map[string]secret, err error) {
	userConfigs, err = h.getUserConfigs(mod.Configs, depReq.Configs)
	if err != nil {
		return nil, nil, nil, model.NewInvalidInputError(err)
	}
	hostRes, err = h.getHostRes(ctx, mod.HostResources, depReq.HostResources)
	if err != nil {
		return nil, nil, nil, err
	}
	secrets, err = h.getSecrets(ctx, mod, dID, depReq.Secrets)
	if err != nil {
		return nil, nil, nil, err
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

func (h *Handler) getHostRes(ctx context.Context, mHostRes map[string]module.HostResource, userInput map[string]string) (map[string]hm_model.HostResource, error) {
	usrHostRes, missing, err := getUserHostRes(userInput, mHostRes)
	if err != nil {
		return nil, model.NewInvalidInputError(err)
	}
	if len(missing) > 0 {
		return nil, model.NewInternalError(errors.New("host resource discovery not implemented"))
	}
	ch := context_hdl.New()
	defer ch.CancelAll()
	hostRes := make(map[string]hm_model.HostResource)
	for ref, id := range usrHostRes {
		res, err := h.hmClient.GetHostResource(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), id)
		if err != nil {
			return nil, model.NewInternalError(err)
		}
		hostRes[ref] = res
	}
	return hostRes, nil
}

func (h *Handler) getSecrets(ctx context.Context, mod *module.Module, dID string, userInput map[string]string) (map[string]secret, error) {
	usrSecrets, missing, err := getUserSecrets(userInput, mod.Secrets)
	if err != nil {
		return nil, model.NewInvalidInputError(err)
	}
	if len(missing) > 0 {
		return nil, model.NewInternalError(errors.New("secret discovery not implemented"))
	}
	ch := context_hdl.New()
	defer ch.CancelAll()
	secrets := make(map[string]secret)
	for ref, sID := range usrSecrets {
		sec, ok := secrets[ref]
		if !ok {
			sec.ID = sID
			sec.Variants = make(map[string]secretVariant)
			secrets[ref] = sec
		}
		for _, service := range mod.Services {
			for _, target := range service.SecretMounts {
				sKey := genSecretMapKey(sID, target.Item)
				variant, ok := sec.Variants[sKey]
				if variant.Path == "" {
					v, err, _ := h.smClient.InitPathVariant(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), sm_model.SecretVariantRequest{
						ID:        sID,
						Item:      target.Item,
						Reference: dID,
					})
					if err != nil {
						return nil, model.NewInternalError(fmt.Errorf("initializing path variant for secret '%s' failed: %s", sID, err))
					}
					if !ok {
						variant.Item = target.Item
					}
					variant.Path = v.Path
					sec.Variants[sKey] = variant
				}
			}
			for _, target := range service.SecretVars {
				sKey := genSecretMapKey(sID, target.Item)
				variant, ok := sec.Variants[sKey]
				if !variant.AsEnv {
					v, err, _ := h.smClient.GetValueVariant(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), sm_model.SecretVariantRequest{
						ID:        sID,
						Item:      target.Item,
						Reference: dID,
					})
					if err != nil {
						return nil, model.NewInternalError(fmt.Errorf("retreiving value variant for secret '%s' failed: %s", sID, err))
					}
					if !ok {
						variant.Item = target.Item
					}
					variant.AsEnv = true
					variant.Value = v.Value
					sec.Variants[sKey] = variant
				}
			}
		}
	}
	return secrets, nil
}

func (h *Handler) storeDepAssets(ctx context.Context, tx driver.Tx, dID string, hostRes map[string]hm_model.HostResource, secrets map[string]secret, modConfigs module.Configs, userConfigs map[string]any) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	if len(hostRes) > 0 {
		hr := make(map[string]string)
		for ref, res := range hostRes {
			hr[ref] = res.ID
		}
		if err := h.storageHandler.CreateDepHostRes(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, hr, dID); err != nil {
			return err
		}
	}
	if len(secrets) > 0 {
		depSecrets := make(map[string]model.DepSecret)
		for ref, sec := range secrets {
			var variants []model.DepSecretVariant
			for _, variant := range sec.Variants {
				variants = append(variants, model.DepSecretVariant{
					Item:    variant.Item,
					AsMount: variant.Path != "",
					AsEnv:   variant.AsEnv,
				})
			}
			depSecrets[ref] = model.DepSecret{
				ID:       sec.ID,
				Variants: variants,
			}
		}
		if err := h.storageHandler.CreateDepSecrets(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, depSecrets, dID); err != nil {
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

func genSecretMapKey(id string, item *string) string {
	if item != nil {
		return id + *item
	}
	return id
}

func getDepName(mName string, userInput *string) string {
	if userInput != nil && *userInput != "" {
		return *userInput
	}
	return mName
}

func getVolumeName(dID, name string) string {
	return "mgw_" + genHash(dID, name)
}

func genHash(str ...string) string {
	hash := sha1.New()
	for _, s := range str {
		hash.Write([]byte(s))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func getUserHostRes(userInput map[string]string, mHostRes map[string]module.HostResource) (map[string]string, []string, error) {
	usrHostRes := make(map[string]string)
	var missing []string
	for ref, mHR := range mHostRes {
		id, ok := userInput[ref]
		if ok {
			usrHostRes[ref] = id
		} else {
			if mHR.Required {
				if len(mHR.Tags) > 0 {
					missing = append(missing, ref)
				} else {
					return nil, nil, fmt.Errorf("host resource '%s' required", ref)
				}
			}
		}
	}
	return usrHostRes, missing, nil
}

func getUserSecrets(userInput map[string]string, mSecrets map[string]module.Secret) (map[string]string, []string, error) {
	usrSecrets := make(map[string]string)
	var missing []string
	for ref, mS := range mSecrets {
		id, ok := userInput[ref]
		if ok {
			usrSecrets[ref] = id
		} else {
			if mS.Required {
				if len(mS.Tags) > 0 {
					missing = append(missing, ref)
				} else {
					return nil, nil, fmt.Errorf("secret '%s' required", ref)
				}
			}
		}
	}
	return usrSecrets, missing, nil
}
