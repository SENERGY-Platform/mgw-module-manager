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

import (
	"context"
	"errors"
	"github.com/SENERGY-Platform/go-service-base/context-hdl"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/naming_hdl"
)

func (h *Handler) Delete(ctx context.Context, dID, aID string, force bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	auxDeployment, err := h.storageHandler.ReadAuxDep(ctxWt, aID, false)
	if err != nil {
		return err
	}
	if auxDeployment.DepID != dID {
		return lib_model.NewForbiddenError(errors.New("deployment ID mismatch"))
	}
	return h.delete(ctx, auxDeployment, force)
}

func (h *Handler) DeleteAll(ctx context.Context, dID string, filter lib_model.AuxDepFilter, force bool) ([]string, error) {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	auxDeployments, err := h.storageHandler.ListAuxDep(ctxWt, dID, filter, false)
	if err != nil {
		return nil, err
	}
	var deleted []string
	for _, auxDeployment := range auxDeployments {
		if err = h.delete(ctx, auxDeployment, force); err != nil {
			return deleted, err
		}
		deleted = append(deleted, auxDeployment.ID)
	}
	return deleted, nil
}

func (h *Handler) delete(ctx context.Context, auxDep lib_model.AuxDeployment, force bool) error {
	if err := h.removeContainer(ctx, auxDep.Container.ID, force); err != nil {
		return err
	}
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	volumes, err := h.cewClient.GetVolumes(ctxWt, cew_model.VolumeFilter{Labels: map[string]string{naming_hdl.DeploymentIDLabel: auxDep.DepID, naming_hdl.AuxDeploymentID: auxDep.ID}})
	if err != nil {
		return err
	}
	var vols []string
	for _, v := range volumes {
		vols = append(vols, v.Name)
	}
	if err = h.removeVolumes(ctx, vols, force); err != nil {
		return lib_model.NewInternalError(err)
	}
	ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
	defer cf2()
	return h.storageHandler.DeleteAuxDep(ctxWt2, nil, auxDep.ID)
}

func (h *Handler) removeContainer(ctx context.Context, id string, force bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	if err := h.cewClient.RemoveContainer(ctxWt, id, force); err != nil {
		var nfe *cew_model.NotFoundError
		if !errors.As(err, &nfe) {
			return lib_model.NewInternalError(err)
		}
	}
	return nil
}

func (h *Handler) removeVolumes(ctx context.Context, volumes []string, force bool) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	for _, name := range volumes {
		err := h.cewClient.RemoveVolume(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), name, force)
		if err != nil {
			var nfe *cew_model.NotFoundError
			if !errors.As(err, &nfe) {
				return err
			}
		}
	}
	return nil
}
