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

package deployment

import (
	"context"
	"errors"
	"github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"time"
)

type Handler struct {
	storageHandler handler.DepStorageHandler
	cfgVltHandler  handler.CfgValidationHandler
	cewClient      client.CewClient
	dbTimeout      time.Duration
	httpTimeout    time.Duration
}

func NewHandler(storageHandler handler.DepStorageHandler, cfgVltHandler handler.CfgValidationHandler, cewClient client.CewClient, dbTimeout time.Duration, httpTimeout time.Duration) *Handler {
	return &Handler{
		storageHandler: storageHandler,
		cfgVltHandler:  cfgVltHandler,
		cewClient:      cewClient,
		dbTimeout:      dbTimeout,
		httpTimeout:    httpTimeout,
	}
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

func (h *Handler) Create(ctx context.Context, m *module.Module, name *string, hostRes map[string]string, secrets map[string]string, configs map[string]any) (string, error) {
	hRes, hResAD, err := parseHostRes(hostRes, m.HostResources)
	if err != nil {
		return "", model.NewInvalidInputError(err)
	}
	sec, secAD, err := parseSecrets(secrets, m.Secrets)
	if err != nil {
		return "", model.NewInvalidInputError(err)
	}
	if len(hResAD) > 0 || len(secAD) > 0 {
		return "", model.NewInternalError(errors.New("host resource and secret discovery not implemented"))
	}
	cfg, err := parseConfigs(configs, m.Configs)
	if err != nil {
		return "", model.NewInvalidInputError(err)
	}
	if err = h.validateConfigs(cfg, m.Configs); err != nil {
		return "", err
	}
	cfgEnvVals, err := genConfigEnvValues(m.Configs, cfg)
	if err != nil {
		return "", model.NewInvalidInputError(err)
	}
	dName := m.Name
	if name != nil {
		dName = *name
	}
	timestamp := time.Now().UTC()
	dbCtx, dbCf := context.WithTimeout(ctx, h.dbTimeout)
	defer dbCf()
	tx, err := h.storageHandler.BeginTransaction(dbCtx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	dID, err := h.storageHandler.CreateDep(dbCtx, tx, m.ID, dName, timestamp)
	if err != nil {
		return "", err
	}
	if len(hRes) > 0 {
		err = h.storageHandler.CreateDepHostRes(dbCtx, tx, hRes, dID)
		if err != nil {
			return "", err
		}
	}
	if len(sec) > 0 {
		err = h.storageHandler.CreateDepSecrets(dbCtx, tx, sec, dID)
		if err != nil {
			return "", err
		}
	}
	if len(cfg) > 0 {
		err = h.storageHandler.CreateDepConfigs(dbCtx, tx, m.Configs, cfg, dID)
		if err != nil {
			return "", err
		}
	}
	iID, err := h.storageHandler.CreateInst(dbCtx, tx, dID, timestamp)
	if err != nil {
		return "", err
	}

	for v := range m.Volumes {
		vName, err := h.createVolume(ctx, dID, iID, v)
		if err != nil {
			return "", err
		}
	}

	err = tx.Commit()
	if err != nil {
		return "", model.NewInternalError(err)
	}
	return dID, nil
}

func (h *Handler) Delete(ctx context.Context, id string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	return h.storageHandler.DeleteDep(ctxWt, id)
}

func (h *Handler) Update(ctx context.Context, m *module.Module, id string, name *string, hostRes map[string]string, secrets map[string]string, configs map[string]any) error {
	panic("not implemented")
}

func (h *Handler) Start(ctx context.Context, id string) error {
	panic("not implemented")
}

func (h *Handler) Stop(ctx context.Context, id string) error {
	panic("not implemented")
}

func (h *Handler) validateConfigs(dCs map[string]any, mCs module.Configs) error {
	for ref, val := range dCs {
		mC := mCs[ref]
		if err := h.cfgVltHandler.ValidateValue(mC.Type, mC.TypeOpt, val, mC.IsSlice, mC.DataType); err != nil {
			return model.NewInvalidInputError(err)
		}
		if mC.Options != nil && !mC.OptExt {
			if err := h.cfgVltHandler.ValidateValInOpt(mC.Options, val, mC.IsSlice, mC.DataType); err != nil {
				return model.NewInvalidInputError(err)
			}
		}
	}
	return nil
}

func (h *Handler) getConfigs(mConfigs module.Configs, userInput map[string]any) (map[string]string, map[string]any, error) {
	userValues, err := parseConfigs(userInput, mConfigs)
	if err != nil {
		return nil, nil, model.NewInvalidInputError(err)
	}
	if err = h.validateConfigs(userValues, mConfigs); err != nil {
		return nil, nil, err
	}
	envValues, err := genConfigEnvValues(mConfigs, userValues)
	if err != nil {
		return nil, nil, model.NewInvalidInputError(err)
	}
	return envValues, userValues, nil
}

func (h *Handler) getHostRes(mHostRes map[string]module.HostResource, userInput map[string]string) (map[string]string, error) {
	hostRes, missing, err := parseHostRes(userInput, mHostRes)
	if err != nil {
		return nil, model.NewInvalidInputError(err)
	}
	if len(missing) > 0 {
		return nil, model.NewInternalError(errors.New("host resource discovery not implemented"))
	}
	return hostRes, nil
}

func (h *Handler) getSecrets(mSecrets map[string]module.Secret, userInput map[string]string) (map[string]string, error) {
	secrets, missing, err := parseSecrets(userInput, mSecrets)
	if err != nil {
		return nil, model.NewInvalidInputError(err)
	}
	if len(missing) > 0 {
		return nil, model.NewInternalError(errors.New("secret discovery not implemented"))
	}
	return secrets, nil
}

func (h *Handler) createVolume(ctx context.Context, dID, iID, v string) (string, error) {
	httpCtx, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	vName, err := h.cewClient.CreateVolume(httpCtx, cew_model.Volume{
		Name:   genVolumeName(iID, v),
		Labels: map[string]string{"d_id": dID, "i_id": iID},
	})
	if err != nil {
		return "", model.NewInternalError(err)
	}
	return vName, nil
}

func (h *Handler) getVolumes(ctx context.Context, mVolumes util.Set[string], dID, iID string) (map[string]string, error) {
	volumes := make(map[string]string)
	for ref := range mVolumes {
		name, err := h.createVolume(ctx, dID, iID, ref)
		if err != nil {
			return nil, err
		}
		volumes[ref] = name
	}
	return volumes, nil
}

func (h *Handler) getDeployments(ctx context.Context, modules map[string]*module.Module, deployments map[string]string) error {
	for mID := range modules {
		ds, err := h.storageHandler.ListDep(ctx, model.DepFilter{ModuleID: mID})
		if err != nil {
			return err
		}
		if len(ds) > 0 {
			deployments[mID] = ds[0].ID
		}
	}
	return nil
}

func getName(mName string, userInput *string) string {
	if userInput != nil {
		return *userInput
	}
	return mName
}
