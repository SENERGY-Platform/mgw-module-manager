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

package dep_adv_hdl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	context_hdl "github.com/SENERGY-Platform/go-service-base/context-hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/model"
	"time"
)

type Handler struct {
	storageHandler handler.DepAdvStorageHandler
	dbTimeout      time.Duration
}

func New(storageHandler handler.DepAdvStorageHandler, dbTimeout time.Duration) *Handler {
	return &Handler{
		storageHandler: storageHandler,
		dbTimeout:      dbTimeout,
	}
}

func (h *Handler) List(ctx context.Context, filter lib_model.DepAdvFilter) ([]lib_model.DepAdvertisement, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	depAds, err := h.storageHandler.ListDepAdv(ctxWt, model.DepAdvFilter{DepAdvFilter: filter})
	if err != nil {
		return nil, err
	}
	var ads []lib_model.DepAdvertisement
	for _, adv := range depAds {
		ads = append(ads, adv.DepAdvertisement)
	}
	return ads, nil
}

func (h *Handler) Get(ctx context.Context, dID, ref string) (lib_model.DepAdvertisement, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	depAdv, err := h.storageHandler.ReadDepAdv(ctxWt, dID, ref)
	if err != nil {
		return lib_model.DepAdvertisement{}, err
	}
	return depAdv.DepAdvertisement, nil
}

func (h *Handler) GetAll(ctx context.Context, dID string) (map[string]lib_model.DepAdvertisement, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	depAds, err := h.storageHandler.ListDepAdv(ctxWt, model.DepAdvFilter{DepID: dID})
	if err != nil {
		return nil, err
	}
	ads := make(map[string]lib_model.DepAdvertisement)
	for _, adv := range depAds {
		ads[adv.Ref] = adv.DepAdvertisement
	}
	return ads, nil
}

func (h *Handler) Put(ctx context.Context, mID, dID string, adv lib_model.DepAdvertisementBase) error {
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	err = h.storageHandler.DeleteDepAdv(ctxWt, tx, dID, adv.Ref)
	if err != nil {
		var nfe *lib_model.NotFoundError
		if !errors.As(err, &nfe) {
			return err
		}
	}
	ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
	defer cf2()
	_, err = h.storageHandler.CreateDepAdv(ctxWt2, tx, newAdv(mID, dID, adv))
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) PutAll(ctx context.Context, mID, dID string, ads map[string]lib_model.DepAdvertisementBase) error {
	tx, err := h.storageHandler.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ch := context_hdl.New()
	defer ch.CancelAll()
	err = h.storageHandler.DeleteAllDepAdv(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, dID)
	if err != nil {
		var nfe *lib_model.NotFoundError
		if !errors.As(err, &nfe) {
			return err
		}
	}
	for ref, adv := range ads {
		if ref != adv.Ref {
			return lib_model.NewInvalidInputError(fmt.Errorf("reference mismatch: %s != %s", adv.Ref, ref))
		}
		_, err = h.storageHandler.CreateDepAdv(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), tx, newAdv(mID, dID, adv))
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return lib_model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) Delete(ctx context.Context, dID, ref string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	err := h.storageHandler.DeleteDepAdv(ctxWt, nil, dID, ref)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) DeleteAll(ctx context.Context, dID string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	err := h.storageHandler.DeleteAllDepAdv(ctxWt, nil, dID)
	if err != nil {
		return err
	}
	return nil
}

func newAdv(mID, dID string, adv lib_model.DepAdvertisementBase) model.DepAdvertisement {
	hash := sha256.New()
	hash.Write([]byte(dID))
	return model.DepAdvertisement{
		DepID: dID,
		DepAdvertisement: lib_model.DepAdvertisement{
			ModuleID:             mID,
			Origin:               hex.EncodeToString(hash.Sum(nil)),
			Timestamp:            time.Now().UTC(),
			DepAdvertisementBase: adv,
		},
	}
}
