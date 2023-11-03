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
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
)

func (h *Handler) Disable(ctx context.Context, id string) error {
	ch := context_hdl.New()
	defer ch.CancelAll()
	d, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), id, true)
	if err != nil {
		return err
	}
	d.Enabled = false
	return h.storageHandler.UpdateDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), nil, d.DepBase)
}

func (h *Handler) getDepFromIDs(ctx context.Context, dIDs []string) ([]model.Deployment, error) {
	ch := context_hdl.New()
	defer ch.CancelAll()
	var dep []model.Deployment
	for _, dID := range dIDs {
		d, err := h.storageHandler.ReadDep(ch.Add(context.WithTimeout(ctx, h.dbTimeout)), dID, false)
		if err != nil {
			return nil, err
		}
		dep = append(dep, d)
	}
	return dep, nil
}
