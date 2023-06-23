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
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"sync"
	"time"
)

type Handler struct {
	transferHandler handler.ModTransferHandler
	modFileHandler  handler.ModFileHandler
	updates         map[string]update
	mu              sync.RWMutex
}

type update struct {
	model.ModUpdate
	stage  handler.Stage
	newIDs map[string]struct{}
	uptIDs map[string]struct{}
}

func New(transferHandler handler.ModTransferHandler, modFileHandler handler.ModFileHandler) *Handler {
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
	var modRepos []handler.ModRepo
	defer func() {
		for _, modRepo := range modRepos {
			modRepo.Remove()
		}
	}()
	updates := make(map[string]update)
	for _, mod := range modules {
		if ctx.Err() != nil {
			return model.NewInternalError(ctx.Err())
		}
		modRepo, err := h.transferHandler.Get(ctx, mod.ID)
		if err != nil {
			continue
		}
		modRepos = append(modRepos, modRepo)
		var versions []string
		for _, ver := range modRepo.Versions() {
			if ctx.Err() != nil {
				return model.NewInternalError(ctx.Err())
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
				ModUpdate: model.ModUpdate{
					Versions: versions,
					Checked:  time.Now().UTC(),
				},
			}
		}
	}
	h.updates = updates
	return nil
}

func (h *Handler) List(_ context.Context) map[string]model.ModUpdate {
	h.mu.RLock()
	defer h.mu.RUnlock()
	updates := make(map[string]model.ModUpdate)
	for mID, upt := range h.updates {
		updates[mID] = upt.ModUpdate
	}
	return updates
}

func (h *Handler) Get(_ context.Context, mID string) (model.ModUpdate, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	upt, ok := h.updates[mID]
	if !ok {
		return model.ModUpdate{}, model.NewNotFoundError(fmt.Errorf("no update available for '%s'", mID))
	}
	return upt.ModUpdate, nil
}

func (h *Handler) Remove(_ context.Context, mID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	upt, ok := h.updates[mID]
	if !ok {
		return model.NewNotFoundError(fmt.Errorf("no update available for '%s'", mID))
	}
	if upt.stage != nil {
		if err := upt.stage.Remove(); err != nil {
			return model.NewInternalError(err)
		}
	}
	delete(h.updates, mID)
	return nil
}

func (h *Handler) Prepare(ctx context.Context, modules map[string]*module.Module, stage handler.Stage, mID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.checkForPending(); err != nil {
		return err
	}
	upt, ok := h.updates[mID]
	if !ok {
		return model.NewNotFoundError(fmt.Errorf("no update available for '%s'", mID))
	}
	stgItems := stage.Items()
	newIDs := make(map[string]struct{})
	uptIDs := make(map[string]struct{})
	for id, item := range stgItems {
		if ctx.Err() != nil {
			return model.NewInternalError(ctx.Err())
		}
		modOld, ok := modules[id]
		if !ok {
			newIDs[id] = struct{}{}
		} else {
			modNew := item.Module()
			if modOld.Version == modNew.Version {
				continue
			}
			modReq := getModRequiring(id, modules)
			if len(modReq) > 0 {
				for mrID, verRng := range modReq {
					if ctx.Err() != nil {
						return model.NewInternalError(ctx.Err())
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
	}
	upt.stage = stage
	upt.newIDs = newIDs
	upt.uptIDs = uptIDs
	upt.Pending = true
	h.updates[mID] = upt
	return nil
}

func (h *Handler) GetPending(_ context.Context, mID string) (handler.Stage, map[string]struct{}, map[string]struct{}, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	upt, ok := h.updates[mID]
	if !ok {
		return nil, nil, nil, model.NewNotFoundError(fmt.Errorf("no update available for '%s'", mID))
	}
	if !upt.Pending {
		return nil, nil, nil, model.NewInternalError(fmt.Errorf("no update pending for '%s'", mID))
	}
	return upt.stage, upt.newIDs, upt.uptIDs, nil
}

func (h *Handler) CancelPending(_ context.Context, mID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	upt, ok := h.updates[mID]
	if !ok {
		return model.NewNotFoundError(fmt.Errorf("no update available for '%s'", mID))
	}
	if !upt.Pending {
		return model.NewInternalError(fmt.Errorf("no update pending for '%s'", mID))
	}
	if err := upt.stage.Remove(); err != nil {
		return model.NewInternalError(err)
	}
	upt.stage = nil
	upt.newIDs = nil
	upt.uptIDs = nil
	upt.Pending = false
	h.updates[mID] = upt
	return nil
}

func (h *Handler) checkForPending() error {
	for id, u := range h.updates {
		if u.Pending {
			return model.NewInternalError(fmt.Errorf("update pending for '%s'", id))
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
