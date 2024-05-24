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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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

func (h *Handler) List(_ context.Context, filter lib_model.AdvFilter) ([]lib_model.Advertisement, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var ads []lib_model.Advertisement
	for _, depAds := range h.ads {
		for _, adv := range depAds {
			if filter.ModuleID != "" && filter.ModuleID != adv.ModuleID {
				continue
			}
			if filter.Origin != "" && filter.Origin != adv.Origin {
				continue
			}
			if filter.Ref != "" && filter.Ref != adv.Ref {
				continue
			}
			ads = append(ads, adv.Advertisement)
		}
	}
	return ads, nil
}

func (h *Handler) Get(_ context.Context, dID, ref string) (lib_model.Advertisement, error) {
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

func (h *Handler) GetAll(_ context.Context, dID string) (map[string]lib_model.Advertisement, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	depAds, ok := h.ads[dID]
	if !ok {
		return nil, lib_model.NewNotFoundError(errors.New("not found"))
	}
	ads := make(map[string]lib_model.Advertisement)
	for ref, adv := range depAds {
		ads[ref] = adv.Advertisement
	}
	return ads, nil
}

func (h *Handler) Put(_ context.Context, mID, dID string, adv lib_model.AdvertisementBase) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	depAds, ok := h.ads[dID]
	if !ok {
		depAds = make(map[string]advertisement)
		h.ads[dID] = depAds
	}
	depAds[adv.Ref] = newAdv(mID, dID, adv)
	return nil
}

func (h *Handler) PutAll(_ context.Context, mID, dID string, ads map[string]lib_model.AdvertisementBase) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	depAds := make(map[string]advertisement)
	for ref, adv := range ads {
		if ref != adv.Ref {
			return lib_model.NewInvalidInputError(fmt.Errorf("reference mismatch: %s != %s", adv.Ref, ref))
		}
		depAds[ref] = newAdv(mID, dID, adv)
	}
	h.ads[dID] = depAds
	return nil
}

func (h *Handler) Delete(_ context.Context, dID, ref string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	depAds, ok := h.ads[dID]
	if !ok {
		return lib_model.NewNotFoundError(errors.New("not found"))
	}
	delete(depAds, ref)
	return nil
}

func (h *Handler) DeleteAll(_ context.Context, dID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.ads[dID]
	if !ok {
		return lib_model.NewNotFoundError(errors.New("not found"))
	}
	delete(h.ads, dID)
	return nil
}

func newAdv(mID, dID string, adv lib_model.AdvertisementBase) advertisement {
	hash := sha256.New()
	hash.Write([]byte(dID))
	return advertisement{
		DepID: dID,
		Advertisement: lib_model.Advertisement{
			ModuleID:          mID,
			Origin:            hex.EncodeToString(hash.Sum(nil)),
			AdvertisementBase: adv,
		},
	}
}
