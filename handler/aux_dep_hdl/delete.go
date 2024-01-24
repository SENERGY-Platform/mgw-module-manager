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

package aux_dep_hdl

//import (
//	"context"
//	"errors"
//	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
//	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
//	"github.com/SENERGY-Platform/go-service-base/context-hdl"
//)
//
//func (h *Handler) Delete(ctx context.Context, aID string, force bool) error {
//	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
//	defer cf()
//	auxDeployment, err := h.storageHandler.ReadAuxDep(ctxWt, aID, false)
//	if err != nil {
//		return err
//	}
//	if err = h.removeContainer(ctx, auxDeployment.Container.ID, force); err != nil {
//		return err
//	}
//	ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
//	defer cf2()
//	return h.storageHandler.DeleteAuxDep(ctxWt2, nil, aID)
//}
//
//func (h *Handler) DeleteAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter, force bool) error {
//	ch := context_hdl.New()
//	defer ch.CancelAll()
//	auxDeployments, err := h.storageHandler.ListAuxDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID, filter, false)
//	if err != nil {
//		return err
//	}
//	for aID, auxDeployment := range auxDeployments {
//		if err = h.removeContainer(ctx, auxDeployment.Container.ID, force); err != nil {
//			return err
//		}
//		if err = h.storageHandler.DeleteAuxDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), nil, aID); err != nil {
//			return err
//		}
//	}
//	return nil
//}
//
//func (h *Handler) removeContainer(ctx context.Context, id string, force bool) error {
//	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
//	defer cf()
//	if err := h.cewClient.RemoveContainer(ctxWt, id, force); err != nil {
//		var nfe *cew_model.NotFoundError
//		if !errors.As(err, &nfe) {
//			return lib_model.NewInternalError(err)
//		}
//	}
//	return nil
//}
