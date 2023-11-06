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
	"database/sql/driver"
	"errors"
	"fmt"
	cew_lib "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	hm_lib "github.com/SENERGY-Platform/mgw-host-manager/lib"
	hm_model "github.com/SENERGY-Platform/mgw-host-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
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
	cewClient      cew_lib.Api
	hmClient       hm_lib.Api
	smClient       sm_client.Client
	dbTimeout      time.Duration
	httpTimeout    time.Duration
	wrkSpcPath     string
	depHostPath    string
	secHostPath    string
	managerID      string
	coreID         string
	moduleNet      string
}

func New(storageHandler handler.DepStorageHandler, cfgVltHandler handler.CfgValidationHandler, cewClient cew_lib.Api, hmClient hm_lib.Api, smClient sm_client.Client, dbTimeout time.Duration, httpTimeout time.Duration, workspacePath, depHostPath, secHostPath, managerID, moduleNet, coreID string) *Handler {
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
		managerID:      managerID,
		coreID:         coreID,
		moduleNet:      moduleNet,
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

func (h *Handler) Get(ctx context.Context, id string, assets, instance bool) (model.Deployment, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	dep, err := h.storageHandler.ReadDep(ctxWt, id, assets)
	if err != nil {
		return model.Deployment{}, err
	}
	if instance {
		dep.Instance, err = h.getDepInstance(ctx, id)
		if err != nil {
			return model.Deployment{}, err
		}
	}
	return dep, err
}

func (h *Handler) getDepFromIDs(ctx context.Context, dIDs []string) ([]model.Deployment, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	var dep []model.Deployment
	for _, dID := range dIDs {
		d, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID, false)
		if err != nil {
			return nil, err
		}
		dep = append(dep, d)
	}
	return dep, nil
}

func (h *Handler) getReqDep(ctx context.Context, dep model.Deployment, reqDep map[string]model.Deployment) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, dID := range dep.RequiredDep {
		if _, ok := reqDep[dID]; !ok {
			d, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID, true)
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

func (h *Handler) getDepAssets(ctx context.Context, mod *module.Module, dID string, depInput model.DepInput) (map[string]hm_model.HostResource, map[string]secret, map[string]model.DepConfig, map[string]string, error) {
	hostResources, err := h.getHostResources(ctx, mod.HostResources, depInput.HostResources)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	secrets, err := h.getSecrets(ctx, mod, dID, depInput.Secrets)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	userConfigs, err := h.getUserConfigs(mod.Configs, depInput.Configs)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	reqModDepMap, err := h.getReqModDepMap(ctx, mod.Dependencies)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return hostResources, secrets, userConfigs, reqModDepMap, nil
}

func (h *Handler) createDepAssets(ctx context.Context, tx driver.Tx, mod *module.Module, dID string, hostResources map[string]hm_model.HostResource, secrets map[string]secret, userConfigs map[string]model.DepConfig, reqModDepMap map[string]string) (model.DepAssets, error) {
	var requiredDep []string
	for mID := range mod.Dependencies {
		requiredDep = append(requiredDep, reqModDepMap[mID])
	}
	depAssets := model.DepAssets{
		HostResources: make(map[string]string),
		Secrets:       make(map[string]model.DepSecret),
		Configs:       userConfigs,
		RequiredDep:   requiredDep,
	}
	for ref, hostResource := range hostResources {
		depAssets.HostResources[ref] = hostResource.ID
	}
	for ref, sec := range secrets {
		var variants []model.DepSecretVariant
		for _, variant := range sec.Variants {
			variants = append(variants, model.DepSecretVariant{
				Item:    variant.Item,
				AsMount: variant.Path != "",
				AsEnv:   variant.AsEnv,
			})
		}
		depAssets.Secrets[ref] = model.DepSecret{
			ID:       sec.ID,
			Variants: variants,
		}
	}
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	err := h.storageHandler.CreateDepAssets(ctxWt, tx, dID, depAssets)
	if err != nil {
		return model.DepAssets{}, err
	}
	return depAssets, nil
}

func (h *Handler) getUserConfigs(mConfigs module.Configs, userInput map[string]any) (map[string]model.DepConfig, error) {
	userConfigs := make(map[string]model.DepConfig)
	for ref, mConfig := range mConfigs {
		val, ok := userInput[ref]
		if !ok || val == nil {
			if mConfig.Default == nil && mConfig.Required {
				return nil, model.NewInvalidInputError(fmt.Errorf("config '%s' requried", ref))
			}
		} else {
			var v any
			var err error
			if mConfig.IsSlice {
				v, err = parser.AnyToDataTypeSlice(val, mConfig.DataType)
			} else {
				v, err = parser.AnyToDataType(val, mConfig.DataType)
			}
			if err != nil {
				return nil, model.NewInvalidInputError(fmt.Errorf("parsing user input '%s' failed: %s", ref, err))
			}
			if err = h.cfgVltHandler.ValidateValue(mConfig.Type, mConfig.TypeOpt, v, mConfig.IsSlice, mConfig.DataType); err != nil {
				return nil, model.NewInvalidInputError(err)
			}
			if mConfig.Options != nil && !mConfig.OptExt {
				if err = h.cfgVltHandler.ValidateValInOpt(mConfig.Options, v, mConfig.IsSlice, mConfig.DataType); err != nil {
					return nil, model.NewInvalidInputError(err)
				}
			}
			userConfigs[ref] = model.DepConfig{
				Value:    v,
				DataType: mConfig.DataType,
				IsSlice:  mConfig.IsSlice,
			}
		}
	}
	return userConfigs, nil
}

func (h *Handler) getHostResources(ctx context.Context, mHostRes map[string]module.HostResource, userInput map[string]string) (map[string]hm_model.HostResource, error) {
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
	variants := make(map[string]secretVariant)
	for ref, sID := range usrSecrets {
		sec, ok := secrets[ref]
		if !ok {
			sec.ID = sID
			sec.Variants = make(map[string]secretVariant)
		}
		for _, service := range mod.Services {
			for _, target := range service.SecretMounts {
				if target.Ref == ref {
					vID := genSecretVariantID(sID, target.Item)
					variant, ok := variants[vID]
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
						variants[vID] = variant
					}
					sec.Variants[vID] = variant
				}
			}
			for _, target := range service.SecretVars {
				if target.Ref == ref {
					vID := genSecretVariantID(sID, target.Item)
					variant, ok := variants[vID]
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
						variants[vID] = variant
					}
					sec.Variants[vID] = variant
				}
			}
		}
		secrets[ref] = sec
	}
	return secrets, nil
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

func (h *Handler) createVolumes(ctx context.Context, mVolumes []string, dID string) error {
	var err error
	var createdVols []string
	defer func() {
		if err != nil {
			h.removeVolumes(context.Background(), createdVols, true)
		}
	}()
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, ref := range mVolumes {
		name := getVolumeName(dID, ref)
		var n string
		n, err = h.cewClient.CreateVolume(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cew_model.Volume{
			Name:   name,
			Labels: map[string]string{handler.CoreIDLabel: h.coreID, handler.ManagerIDLabel: h.managerID, handler.DeploymentIDLabel: dID},
		})
		if err != nil {
			return model.NewInternalError(err)
		}
		if n != name {
			err = fmt.Errorf("volume name missmatch: %s != %s", n, name)
			return model.NewInternalError(err)
		}
		createdVols = append(createdVols, n)
	}
	return nil
}

func (h *Handler) removeVolumes(ctx context.Context, volumes []string, force bool) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, name := range volumes {
		err := h.cewClient.RemoveVolume(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), name, force)
		if err != nil {
			var nfe *cew_model.NotFoundError
			if !errors.As(err, &nfe) {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) loadSecrets(ctx context.Context, dep model.Deployment) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, depSecret := range dep.Secrets {
		for _, variant := range depSecret.Variants {
			if variant.AsMount {
				err, _ := h.smClient.LoadPathVariant(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), sm_model.SecretVariantRequest{
					ID:        depSecret.ID,
					Item:      variant.Item,
					Reference: dep.ID,
				})
				if err != nil {
					return model.NewInternalError(fmt.Errorf("loading path variant for secret '%s' failed: %s", depSecret.ID, err))
				}
			}
		}
	}
	return nil
}

func (h *Handler) unloadSecrets(ctx context.Context, dID string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	err, _ := h.smClient.CleanPathVariants(ctxWt, dID)
	if err != nil {
		return model.NewInternalError(fmt.Errorf("unloading path variants for secret '%s' failed: %s", dID, err))
	}
	return nil
}

func genSecretVariantID(id string, item *string) string {
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
	return "mgw_" + util.GenHash(dID, name)
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
