/*
 * Copyright 2026 InfAI (CC SES)
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

package dep_advertisements

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	helper_uuid "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/uuid"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

type Handler struct {
	databaseHandler databaseHandler
}

func New(databaseHandler databaseHandler) *Handler {
	return &Handler{databaseHandler: databaseHandler}
}

func (h *Handler) GetAdvertisement(
	ctx context.Context,
	deploymentId string,
	reference string,
) (lib_models.DeploymentAdvertisement, error) {
	return h.databaseHandler.ReadDeploymentAdvertisement(ctx, deploymentId, reference)
}

func (h *Handler) GetAdvertisementById(ctx context.Context, id string) (lib_models.DeploymentAdvertisement, error) {
	advertisements, err := h.databaseHandler.ReadDeploymentAdvertisements(ctx, lib_models.DeploymentAdvertisementsFilter{
		Ids: []string{id},
	})
	if err != nil {
		return lib_models.DeploymentAdvertisement{}, err
	}
	if len(advertisements) == 0 {
		return lib_models.DeploymentAdvertisement{}, pkg_models.NotFoundErr
	}
	return advertisements[id], nil
}

func (h *Handler) GetAdvertisements(
	ctx context.Context,
	filter lib_models.DeploymentAdvertisementsFilter,
) (map[string]lib_models.DeploymentAdvertisement, error) {
	return h.databaseHandler.ReadDeploymentAdvertisements(ctx, filter)
}

func (h *Handler) PutAdvertisement(
	ctx context.Context,
	moduleId string,
	deploymentId string,
	reference string,
	items map[string]string,
) (string, error) {
	advertisement, err := newDatabaseAdvertisement(moduleId, deploymentId, helper_time.Now(), reference, items)
	if err != nil {
		return "", err
	}
	err = h.databaseHandler.WriteDeploymentAdvertisements(
		ctx,
		deploymentId,
		[]lib_models.DeploymentAdvertisement{advertisement},
		true,
	)
	if err != nil {
		return "", err
	}
	return advertisement.Id, nil
}

func (h *Handler) PutAdvertisements(
	ctx context.Context,
	moduleId string,
	deploymentId string,
	inputs []lib_models.DeploymentAdvertisementInput,
	incremental bool,
) (map[string]string, error) {
	timestamp := helper_time.Now()
	var advertisements []lib_models.DeploymentAdvertisement
	res := make(map[string]string)
	for _, input := range inputs {
		advertisement, err := newDatabaseAdvertisement(moduleId, deploymentId, timestamp, input.Reference, input.Items)
		if err != nil {
			return nil, err
		}
		advertisements = append(advertisements, advertisement)
		res[input.Reference] = advertisement.Id
	}
	err := h.databaseHandler.WriteDeploymentAdvertisements(ctx, deploymentId, advertisements, incremental)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (h *Handler) DeleteAdvertisements(
	ctx context.Context,
	deploymentId string,
	filter lib_models.DeploymentAdvertisementsFilterReduced,
	allowAll bool,
) error {
	if !allowAll && filterEmpty(filter) {
		return nil
	}
	return h.databaseHandler.DeleteDeploymentAdvertisements(ctx, deploymentId, filter)
}

func newDatabaseAdvertisement(
	moduleId string,
	deploymentId string,
	timestamp time.Time,
	reference string,
	items map[string]string,
) (lib_models.DeploymentAdvertisement, error) {
	id, err := helper_uuid.New()
	if err != nil {
		return lib_models.DeploymentAdvertisement{}, err
	}
	originHash := sha256.New()
	originHash.Write([]byte(deploymentId))
	return lib_models.DeploymentAdvertisement{
		Id:        id,
		ModuleId:  moduleId,
		Origin:    hex.EncodeToString(originHash.Sum(nil)),
		Reference: reference,
		Timestamp: timestamp,
		Items:     items,
	}, nil
}

func filterEmpty(filter lib_models.DeploymentAdvertisementsFilterReduced) bool {
	switch {
	case len(filter.References) > 0:
		return false
	case len(filter.Ids) > 0:
		return false
	}
	return true
}
