/*
 * Copyright 2024 InfAI (CC SES)
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

package adv_hdl

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"sync"
)

type advertisement struct {
	DepID string
	lib_model.Advertisement
}

type Handler struct {
	ads map[string]map[string]advertisement // {dID:{ref:advertisement}}
	mu  sync.RWMutex
}

func New() *Handler {
	return &Handler{
		ads: make(map[string]map[string]advertisement),
	}
}

func (h *Handler) List(filter lib_model.AdvFilter) ([]lib_model.Advertisement, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var ads []lib_model.Advertisement
	for dID, depAds := range h.ads {
		if filter.DeploymentID != "" && filter.DeploymentID != dID {
			continue
		}
		for _, adv := range depAds {
			if filter.ModuleID != "" && filter.ModuleID != adv.ModuleID {
				continue
			}
			ads = append(ads, adv.Advertisement)
		}
	}
	return ads, nil
}

func (h *Handler) Get(dID, ref string) (lib_model.Advertisement, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	depAds, ok := h.ads[dID]
	if !ok {
		return lib_model.Advertisement{}, lib_model.NewNotFoundError(errors.New("not found"))
	}
	adv, ok := depAds[ref]
	if !ok {
		return lib_model.Advertisement{}, lib_model.NewNotFoundError(errors.New("not found"))
	}
	return adv.Advertisement, nil
}

func (h *Handler) Put(mID, dID string, adv lib_model.AdvertisementBase) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	depAds, ok := h.ads[dID]
	if !ok {
		depAds = make(map[string]advertisement)
		h.ads[dID] = depAds
	}
	hash := sha256.New()
	hash.Write([]byte(dID))
	depAds[adv.Ref] = advertisement{
		DepID: dID,
		Advertisement: lib_model.Advertisement{
			ModuleID:          mID,
			Origin:            hex.EncodeToString(hash.Sum(nil)),
			AdvertisementBase: adv,
		},
	}
	return nil
}

func (h *Handler) Delete(dID, ref string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	depAds, ok := h.ads[dID]
	if !ok {
		return lib_model.NewNotFoundError(errors.New("not found"))
	}
	delete(depAds, ref)
	return nil
}

func (h *Handler) DeleteAll(dID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.ads[dID]
	if !ok {
		return lib_model.NewNotFoundError(errors.New("not found"))
	}
	delete(h.ads, dID)
	return nil
}
