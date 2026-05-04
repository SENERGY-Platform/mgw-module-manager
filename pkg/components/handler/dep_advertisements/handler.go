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

package handler_dep_advertisements

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/uuid"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/dep_advertisements"
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
) (models_handler_database.DeploymentAdvertisement, error) {
	return h.databaseHandler.ReadDeploymentAdvertisement(ctx, deploymentId, reference)
}

func (h *Handler) GetAdvertisementById(ctx context.Context, id string) (models_handler_database.DeploymentAdvertisement, error) {
	advertisements, err := h.databaseHandler.ReadDeploymentAdvertisements(ctx, models_handler_database.DeploymentAdvertisementsFilter{
		Ids: []string{id},
	})
	if err != nil {
		return models_handler_database.DeploymentAdvertisement{}, err
	}
	if len(advertisements) == 0 {
		return models_handler_database.DeploymentAdvertisement{}, models_error.NotFoundErr
	}
	return advertisements[id], nil
}

func (h *Handler) GetAdvertisements(
	ctx context.Context,
	filter models_handler_database.DeploymentAdvertisementsFilter,
) (map[string]models_handler_database.DeploymentAdvertisement, error) {
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
		[]models_handler_database.DeploymentAdvertisement{advertisement},
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
	inputs []models_handler_dep_advertisements.DeploymentAdvertisementInput,
	incremental bool,
) (map[string]string, error) {
	timestamp := helper_time.Now()
	var advertisements []models_handler_database.DeploymentAdvertisement
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
	filter models_handler_database.DeploymentAdvertisementsFilterReduced,
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
) (models_handler_database.DeploymentAdvertisement, error) {
	id, err := helper_uuid.New()
	if err != nil {
		return models_handler_database.DeploymentAdvertisement{}, err
	}
	originHash := sha256.New()
	originHash.Write([]byte(deploymentId))
	return models_handler_database.DeploymentAdvertisement{
		Id:        id,
		ModuleId:  moduleId,
		Origin:    hex.EncodeToString(originHash.Sum(nil)),
		Reference: reference,
		Timestamp: timestamp,
		Items:     items,
	}, nil
}

func filterEmpty(filter models_handler_database.DeploymentAdvertisementsFilterReduced) bool {
	switch {
	case len(filter.References) > 0:
		return false
	case len(filter.Ids) > 0:
		return false
	}
	return true
}
