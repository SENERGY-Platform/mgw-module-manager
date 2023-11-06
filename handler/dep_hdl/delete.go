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

package dep_hdl

import (
	"context"
	"errors"
	"fmt"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"os"
	"path"
	"strings"
)

func (h *Handler) Delete(ctx context.Context, id string, force bool) error {
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	dep, err := h.storageHandler.ReadDep(ctxWt, id, true)
	if err != nil {
		return err
	}
	if !force {
		if dep.Started {
			return model.NewInvalidInputError(errors.New("deployment is started"))
		}
		if len(dep.DepRequiring) > 0 {
			depReq, err := h.getDepFromIDs(ctx, dep.DepRequiring)
			if err != nil {
				return err
			}
			var reqBy []string
			for _, dr := range depReq {
				reqBy = append(reqBy, fmt.Sprintf("%s (%s)", dr.Name, dr.ID))
			}
			return model.NewInternalError(fmt.Errorf("required by: %s", strings.Join(reqBy, ", ")))
		}
	}
	return h.delete(ctx, dep, force)
}

func (h *Handler) delete(ctx context.Context, dep model.Deployment, force bool) error {
	if err := h.removeInstance(ctx, dep, force); err != nil {
		return err
	}
	if err := h.unloadSecrets(ctx, dep.ID); err != nil {
		return err
	}
	ch := context_hdl.New()
	defer ch.CancelAll()
	volumes, err := h.cewClient.GetVolumes(ch.Add(context.WithTimeout(ctx, h.httpTimeout)), cew_model.VolumeFilter{Labels: map[string]string{handler.DeploymentIDLabel: dep.ID}})
	if err != nil {
		return err
	}
	var vols []string
	for _, v := range volumes {
		vols = append(vols, v.Name)
	}
	if err = h.removeVolumes(ctx, vols, force); err != nil {
		return model.NewInternalError(err)
	}
	if err = os.RemoveAll(path.Join(h.wrkSpcPath, dep.Dir)); err != nil {
		if !os.IsNotExist(err) {
			return model.NewInternalError(err)
		}
	}
	if err = h.storageHandler.DeleteDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dep.ID); err != nil {
		return err
	}
	return nil
}
