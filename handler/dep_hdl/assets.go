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
	"errors"
	"fmt"
	hm_model "github.com/SENERGY-Platform/mgw-host-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/parser"
	sm_model "github.com/SENERGY-Platform/mgw-secret-manager/pkg/api_model"
)

func (h *Handler) getDepAssets(ctx context.Context, mod *module.Module, dID string, depInput model.DepInput) (map[string]hm_model.HostResource, map[string]secret, map[string]model.DepConfig, error) {
	hostResources, err := h.getHostResources(ctx, mod.HostResources, depInput.HostResources)
	if err != nil {
		return nil, nil, nil, err
	}
	userConfigs, err := h.getUserConfigs(mod.Configs, depInput.Configs)
	if err != nil {
		return nil, nil, nil, err
	}
	secrets, err := h.getSecrets(ctx, mod, dID, depInput.Secrets)
	if err != nil {
		return nil, nil, nil, err
	}
	return hostResources, secrets, userConfigs, nil
}

func (h *Handler) newDepAssets(hostResources map[string]hm_model.HostResource, secrets map[string]secret, userConfigs map[string]model.DepConfig) model.DepAssets {
	depAssets := model.DepAssets{
		HostResources: make(map[string]string),
		Secrets:       make(map[string]model.DepSecret),
		Configs:       userConfigs,
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
	return depAssets
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
	defer func() {
		if err != nil {
			if e := h.unloadSecrets(context.Background(), dID); e != nil {
				util.Logger.Error(e)
			}
		}
	}()
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
					vID := newSecretVariantID(sID, target.Item)
					variant, ok := variants[vID]
					if variant.Path == "" {
						var v sm_model.SecretPathVariant
						v, err, _ = h.smClient.InitPathVariant(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), sm_model.SecretVariantRequest{
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
					vID := newSecretVariantID(sID, target.Item)
					variant, ok := variants[vID]
					if !variant.AsEnv {
						var v sm_model.SecretValueVariant
						v, err, _ = h.smClient.GetValueVariant(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), sm_model.SecretVariantRequest{
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

func newSecretVariantID(id string, item *string) string {
	if item != nil {
		return id + *item
	}
	return id
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
