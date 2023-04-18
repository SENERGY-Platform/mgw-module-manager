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
	"fmt"
	"github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-lib/tsort"
	"github.com/SENERGY-Platform/mgw-module-lib/util"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"time"
)

type Handler struct {
	storageHandler handler.DepStorageHandler
	cfgVltHandler  handler.CfgValidationHandler
	moduleHandler  handler.ModuleHandler
	cewClient      client.CewClient
	dbTimeout      time.Duration
	httpTimeout    time.Duration
}

func NewHandler(storageHandler handler.DepStorageHandler, cfgVltHandler handler.CfgValidationHandler, moduleHandler handler.ModuleHandler, cewClient client.CewClient, dbTimeout time.Duration, httpTimeout time.Duration) *Handler {
	return &Handler{
		storageHandler: storageHandler,
		cfgVltHandler:  cfgVltHandler,
		moduleHandler:  moduleHandler,
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

func (h *Handler) Create(ctx context.Context, dr model.DepRequest) (string, error) {
	m, dms, err := h.moduleHandler.GetWithDep(ctx, dr.ModuleID)
	if err != nil {
		return "", err
	}
	if m.DeploymentType == module.SingleDeployment {
		ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
		defer cf()
		if l, err := h.storageHandler.ListDep(ctxWt, model.DepFilter{ModuleID: m.ID}); err != nil {
			return "", err
		} else if len(l) > 0 {
			return "", model.NewInvalidInputError(errors.New("already deployed"))
		}
	}
	depMap := make(map[string]string)
	if len(dms) > 0 {
		for dmID := range dms {
			if l, err := h.storageHandler.ListDep(ctx, model.DepFilter{ModuleID: dmID}); err != nil {
				return "", err
			} else if len(l) > 0 {
				depMap[dmID] = l[0].ID
			}
		}
		order, err := getModOrder(dms)
		if err != nil {
			return "", model.NewInternalError(err)
		}
		var depNew []string
		for _, dmID := range order {
			if _, ok := depMap[dmID]; !ok {
				dID, err := h.create(ctx, dms[dmID], dr.Dependencies[dmID], depMap)
				if err != nil {
					//for _, id := range depNew {
					//	h.Delete(ctx, id)
					//}
					return "", err
				}
				depMap[dmID] = dID
				depNew = append(depNew, dID)
			}
		}
	}
	return h.create(ctx, m, dr.DepRequestBase, depMap)
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
		Name:   getVolumeName(iID, v),
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

func (h *Handler) create(ctx context.Context, m *module.Module, drb model.DepRequestBase, depMap map[string]string) (string, error) {
	configs, userConfigs, err := h.getConfigs(m.Configs, drb.Configs)
	if err != nil {
		return "", err
	}
	hostRes, err := h.getHostRes(m.HostResources, drb.HostResources)
	if err != nil {
		return "", err
	}
	secrets, err := h.getSecrets(m.Secrets, drb.Secrets)
	if err != nil {
		return "", err
	}
	name := getName(m.Name, drb.Name)
	timestamp := time.Now().UTC()
	dbCtx, dbCf := context.WithTimeout(ctx, h.dbTimeout)
	defer dbCf()
	tx, err := h.storageHandler.BeginTransaction(dbCtx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	dID, err := h.storageHandler.CreateDep(dbCtx, tx, m.ID, name, timestamp)
	if err != nil {
		return "", err
	}
	if len(hostRes) > 0 {
		if err = h.storageHandler.CreateDepHostRes(dbCtx, tx, hostRes, dID); err != nil {
			return "", err
		}
	}
	if len(secrets) > 0 {
		if err = h.storageHandler.CreateDepSecrets(dbCtx, tx, secrets, dID); err != nil {
			return "", err
		}
	}
	if len(userConfigs) > 0 {
		if err = h.storageHandler.CreateDepConfigs(dbCtx, tx, m.Configs, userConfigs, dID); err != nil {
			return "", err
		}
	}
	iID, err := h.storageHandler.CreateInst(dbCtx, tx, dID, timestamp)
	if err != nil {
		return "", err
	}
	inclDirPath, err := h.moduleHandler.CreateInclDir(ctx, m.ID, dID)
	if err != nil {
		return "", err
	}
	volumes, err := h.getVolumes(ctx, m.Volumes, dID, iID)
	order, err := getSrvOrder(m.Services)
	if err != nil {
		return "", model.NewInternalError(err)
	}
	for _, ref := range order {
		cID, err := h.createContainer(ctx, m.Services[ref], ref, dID, iID, m.DeploymentType, envValues, volumes, depMap, hostRes, secrets)
		if err != nil {
			return "", err
		}
		err = h.storageHandler.CreateInstCtr(dbCtx, tx, iID, cID, ref)
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

func getName(mName string, userInput *string) string {
	if userInput != nil {
		return *userInput
	}
	return mName
}

func getModOrder(modules map[string]*module.Module) (order []string, err error) {
	if len(modules) > 1 {
		nodes := make(tsort.Nodes)
		for _, m := range modules {
			if len(m.Dependencies) > 0 {
				reqIDs := make(map[string]struct{})
				for i := range m.Dependencies {
					reqIDs[i] = struct{}{}
				}
				nodes.Add(m.ID, reqIDs, nil)
			}
		}
		order, err = tsort.GetTopOrder(nodes)
		if err != nil {
			return nil, err
		}
	} else if len(modules) > 0 {
		for _, m := range modules {
			order = append(order, m.ID)
		}
	}
	return
}

func getSrvOrder(services map[string]*module.Service) (order []string, err error) {
	if len(services) > 1 {
		nodes := make(tsort.Nodes)
		for ref, srv := range services {
			nodes.Add(ref, srv.RequiredSrv, srv.RequiredBySrv)
		}
		order, err = tsort.GetTopOrder(nodes)
		if err != nil {
			return nil, err
		}
	} else if len(services) > 0 {
		for ref := range services {
			order = append(order, ref)
		}
	}
	return
}
