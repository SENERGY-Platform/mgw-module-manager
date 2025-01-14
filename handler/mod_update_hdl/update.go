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

package mod_update_hdl

import (
	"context"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-lib/validation/sem_ver"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/model"
	"sync"
	"time"
)

type Handler struct {
	transferHandler ModTransferHandler
	modFileHandler  ModFileHandler
	updates         map[string]update
	mu              sync.RWMutex
}

type update struct {
	lib_model.ModUpdate
	stage  model.Stage
	newIDs map[string]struct{}
	uptIDs map[string]struct{}
	ophIDs map[string]struct{}
}

func New(transferHandler ModTransferHandler, modFileHandler ModFileHandler) *Handler {
	return &Handler{
		transferHandler: transferHandler,
		modFileHandler:  modFileHandler,
	}
}

func (h *Handler) Check(ctx context.Context, modules map[string]*module.Module) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.checkForPending(); err != nil {
		return err
	}
	var modRepos []model.ModRepo
	defer func() {
		for _, modRepo := range modRepos {
			modRepo.Remove()
		}
	}()
	updates := make(map[string]update)
	for _, mod := range modules {
		if ctx.Err() != nil {
			return lib_model.NewInternalError(ctx.Err())
		}
		modRepo, err := h.transferHandler.Get(ctx, mod.ID)
		if err != nil {
			continue
		}
		modRepos = append(modRepos, modRepo)
		var versions []string
		for _, ver := range modRepo.Versions() {
			if ctx.Err() != nil {
				return lib_model.NewInternalError(ctx.Err())
			}
			res, err := sem_ver.CompareSemVer(mod.Version, ver)
			if err != nil {
				continue
			}
			if res < 0 {
				versions = append(versions, ver)
			}
		}
		if len(versions) > 0 {
			updates[mod.ID] = update{
				ModUpdate: lib_model.ModUpdate{
					Versions: versions,
					Checked:  time.Now().UTC(),
				},
			}
		}
	}
	h.updates = updates
	return nil
}

func (h *Handler) List(_ context.Context) (updates map[string]lib_model.ModUpdate) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.updates) > 0 {
		updates = make(map[string]lib_model.ModUpdate)
		for mID, upt := range h.updates {
			updates[mID] = upt.ModUpdate
		}
	}
	return updates
}

func (h *Handler) Get(_ context.Context, mID string) (lib_model.ModUpdate, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	upt, ok := h.updates[mID]
	if !ok {
		return lib_model.ModUpdate{}, lib_model.NewNotFoundError(fmt.Errorf("no update available for '%s'", mID))
	}
	return upt.ModUpdate, nil
}

func (h *Handler) Remove(_ context.Context, mID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	upt, ok := h.updates[mID]
	if !ok {
		return lib_model.NewNotFoundError(fmt.Errorf("no update available for '%s'", mID))
	}
	if upt.stage != nil {
		if err := upt.stage.Remove(); err != nil {
			return lib_model.NewInternalError(err)
		}
	}
	delete(h.updates, mID)
	return nil
}

func (h *Handler) Prepare(ctx context.Context, modules map[string]*module.Module, stage model.Stage, mID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.checkForPending(); err != nil {
		return err
	}
	upt, ok := h.updates[mID]
	if !ok {
		return lib_model.NewNotFoundError(fmt.Errorf("no update available for '%s'", mID))
	}
	upt.PendingVersions = make(map[string]string)
	stgItems := stage.Items()
	mod, ok := modules[mID]
	if !ok {
		return lib_model.NewInternalError(fmt.Errorf("module '%s' not found", mID))
	}
	reqMod := make(map[string]*module.Module)
	err := getRequiredMod(mod, modules, reqMod)
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	ophIDs := make(map[string]struct{})
	for id := range reqMod {
		if _, ok := stage.Get(id); !ok {
			ophIDs[id] = struct{}{}
		}
	}
	newIDs := make(map[string]struct{})
	uptIDs := make(map[string]struct{})
	for id, item := range stgItems {
		if ctx.Err() != nil {
			return lib_model.NewInternalError(ctx.Err())
		}
		modNew := item.Module()
		modOld, ok := modules[id]
		if !ok {
			newIDs[id] = struct{}{}
		} else {
			if modOld.Version == modNew.Version {
				continue
			}
			modReq := getModRequiring(id, modules)
			if len(modReq) > 0 {
				for mrID, verRng := range modReq {
					if ctx.Err() != nil {
						return lib_model.NewInternalError(ctx.Err())
					}
					if _, ok := stgItems[mrID]; !ok {
						k, err := sem_ver.InSemVerRange(verRng, modNew.Version)
						if err != nil {
							return err
						}
						if !k {
							return fmt.Errorf("module '%s' update '%s' -> '%s' but '%s' requires '%s'", id, modOld.Version, modNew.Version, mrID, verRng)
						}
					}
				}
			}
			uptIDs[id] = struct{}{}
		}
		upt.PendingVersions[mID] = modNew.Version
	}
	upt.stage = stage
	upt.newIDs = newIDs
	upt.uptIDs = uptIDs
	upt.ophIDs = ophIDs
	upt.Pending = true
	h.updates[mID] = upt
	return nil
}

func (h *Handler) GetPending(_ context.Context, mID string) (model.Stage, map[string]struct{}, map[string]struct{}, map[string]struct{}, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	upt, ok := h.updates[mID]
	if !ok {
		return nil, nil, nil, nil, lib_model.NewNotFoundError(fmt.Errorf("no update available for '%s'", mID))
	}
	if !upt.Pending {
		return nil, nil, nil, nil, lib_model.NewInternalError(fmt.Errorf("no update pending for '%s'", mID))
	}
	return upt.stage, upt.newIDs, upt.uptIDs, upt.ophIDs, nil
}

func (h *Handler) CancelPending(_ context.Context, mID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	upt, ok := h.updates[mID]
	if !ok {
		return lib_model.NewNotFoundError(fmt.Errorf("no update available for '%s'", mID))
	}
	if !upt.Pending {
		return lib_model.NewInternalError(fmt.Errorf("no update pending for '%s'", mID))
	}
	if err := upt.stage.Remove(); err != nil {
		return lib_model.NewInternalError(err)
	}
	upt.stage = nil
	upt.newIDs = nil
	upt.uptIDs = nil
	upt.Pending = false
	upt.PendingVersions = nil
	h.updates[mID] = upt
	return nil
}

func (h *Handler) checkForPending() error {
	for id, u := range h.updates {
		if u.Pending {
			return lib_model.NewInternalError(fmt.Errorf("update pending for '%s'", id))
		}
	}
	return nil
}

func getModRequiring(mID string, modules map[string]*module.Module) map[string]string {
	modReq := make(map[string]string)
	for _, mod := range modules {
		if verRng, ok := mod.Dependencies[mID]; ok {
			modReq[mod.ID] = verRng
		}
	}
	return modReq
}

func getRequiredMod(mod *module.Module, modules map[string]*module.Module, reqMod map[string]*module.Module) error {
	for id := range mod.Dependencies {
		if _, ok := reqMod[id]; !ok {
			m, k := modules[id]
			if !k {
				return fmt.Errorf("module '%s' not found", id)
			}
			reqMod[id] = m
			if err := getRequiredMod(m, modules, reqMod); err != nil {
				return err
			}
		}
	}
	return nil
}
