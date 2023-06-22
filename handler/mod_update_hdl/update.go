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
	model.ModUpdateInfo
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
	var modRepos []handler.ModRepo
	defer func() {
		for _, modRepo := range modRepos {
			modRepo.Remove()
		}
	}()
	updates := make(map[string]update)
	for _, mod := range modules {
		modRepo, err := h.transferHandler.Get(ctx, mod.ID)
		if err != nil {
			continue
		}
		modRepos = append(modRepos, modRepo)
		var versions []string
		for _, ver := range modRepo.Versions() {
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
				ModUpdateInfo: model.ModUpdateInfo{
					Versions: versions,
					Checked:  time.Now().UTC(),
				},
			}
		}
	}
	h.updates = updates
	return nil
}

func (h *Handler) List(_ context.Context) map[string]model.ModUpdateInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()
	updates := make(map[string]model.ModUpdateInfo)
	for mID, upt := range h.updates {
		updates[mID] = upt.ModUpdateInfo
	}
	return updates
}
