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
//	job_hdl_lib "github.com/SENERGY-Platform/go-service-base/job-hdl/lib"
//	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
//	"github.com/SENERGY-Platform/mgw-module-manager/util"
//	"github.com/SENERGY-Platform/go-service-base/context-hdl"
//	"net/http"
//	"time"
//)
//
//func (h *Handler) Start(ctx context.Context, aID string) error {
//	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
//	defer cf()
//	auxDeployment, err := h.storageHandler.ReadAuxDep(ctxWt, aID, false)
//	if err != nil {
//		return err
//	}
//	ctxWt2, cf2 := context.WithTimeout(ctx, h.httpTimeout)
//	defer cf2()
//	if err = h.cewClient.StartContainer(ctxWt2, auxDeployment.Container.ID); err != nil {
//		return lib_model.NewInternalError(err)
//	}
//	return nil
//}
//
//func (h *Handler) StartAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter) error {
//	ch := context_hdl.New()
//	defer ch.CancelAll()
//	auxDeployments, err := h.storageHandler.ListAuxDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID, filter, false)
//	if err != nil {
//		return err
//	}
//	for _, auxDeployment := range auxDeployments {
//		if err = h.cewClient.StartContainer(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), auxDeployment.Container.ID); err != nil {
//			return lib_model.NewInternalError(err)
//		}
//	}
//	return nil
//}
//
//func (h *Handler) Stop(ctx context.Context, aID string) error {
//	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
//	defer cf()
//	auxDeployment, err := h.storageHandler.ReadAuxDep(ctxWt, aID, false)
//	if err != nil {
//		return err
//	}
//	if err = h.stopContainer(ctx, auxDeployment.Container.ID); err != nil {
//		return lib_model.NewInternalError(err)
//	}
//	return nil
//}
//
//func (h *Handler) StopAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter) error {
//	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
//	defer cf()
//	auxDeployments, err := h.storageHandler.ListAuxDep(ctxWt, dID, filter, false)
//	if err != nil {
//		return err
//	}
//	for _, auxDeployment := range auxDeployments {
//		if err = h.stopContainer(ctx, auxDeployment.Container.ID); err != nil {
//			return lib_model.NewInternalError(err)
//		}
//	}
//	return nil
//}
//
//func (h *Handler) Restart(ctx context.Context, aID string) error {
//	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
//	defer cf()
//	auxDeployment, err := h.storageHandler.ReadAuxDep(ctxWt, aID, false)
//	if err != nil {
//		return err
//	}
//	return h.restart(ctx, auxDeployment.Container.ID)
//}
//
//func (h *Handler) RestartAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter) error {
//	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
//	defer cf()
//	auxDeployments, err := h.storageHandler.ListAuxDep(ctxWt, dID, filter, false)
//	if err != nil {
//		return err
//	}
//	for _, auxDeployment := range auxDeployments {
//		if err = h.restart(ctx, auxDeployment.Container.ID); err != nil {
//			return err
//		}
//	}
//	return nil
//}
//
//func (h *Handler) stopContainer(ctx context.Context, cID string) error {
//	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
//	defer cf()
//	jID, err := h.cewClient.StopContainer(ctxWt, cID)
//	if err != nil {
//		return lib_model.NewInternalError(err)
//	}
//	job, err := job_hdl_lib.Await(ctx, h.cewClient, jID, time.Second, h.httpTimeout, util.Logger)
//	if err != nil {
//		return lib_model.NewInternalError(err)
//	}
//	if job.Error != nil {
//		if job.Error.Code != nil && *job.Error.Code == http.StatusNotFound {
//			return lib_model.NewNotFoundError(errors.New(job.Error.Message))
//		}
//		return lib_model.NewInternalError(errors.New(job.Error.Message))
//	}
//	return nil
//}
//
//func (h *Handler) restart(ctx context.Context, cID string) error {
//	if err := h.stopContainer(ctx, cID); err != nil {
//		return err
//	}
//	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
//	defer cf()
//	if err := h.cewClient.StartContainer(ctxWt, cID); err != nil {
//		return lib_model.NewInternalError(err)
//	}
//	return nil
//}
