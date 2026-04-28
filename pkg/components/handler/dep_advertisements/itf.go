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

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

type databaseHandler interface {
	ReadDeploymentAdvertisement(
		ctx context.Context,
		deploymentId string,
		reference string,
	) (models_handler_database.DeploymentAdvertisement, error)
	ReadDeploymentAdvertisements(
		ctx context.Context,
		filter models_handler_database.DeploymentAdvertisementsFilter,
	) (map[string]models_handler_database.DeploymentAdvertisement, error)
	WriteDeploymentAdvertisements(
		ctx context.Context,
		deploymentId string,
		advertisements []models_handler_database.DeploymentAdvertisement,
		incremental bool,
	) error
	DeleteDeploymentAdvertisements(ctx context.Context, deploymentId string, references []string) error
}
