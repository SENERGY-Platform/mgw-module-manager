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

	lib_models_dep_advertisements "github.com/SENERGY-Platform/mgw-module-manager/lib/models/dep_advertisements"
)

type databaseHandler interface {
	ReadDeploymentAdvertisement(
		ctx context.Context,
		deploymentId string,
		reference string,
	) (lib_models_dep_advertisements.DeploymentAdvertisement, error)
	ReadDeploymentAdvertisements(
		ctx context.Context,
		filter lib_models_dep_advertisements.DeploymentAdvertisementsFilter,
	) (map[string]lib_models_dep_advertisements.DeploymentAdvertisement, error)
	WriteDeploymentAdvertisements(
		ctx context.Context,
		deploymentId string,
		advertisements []lib_models_dep_advertisements.DeploymentAdvertisement,
		incremental bool,
	) error
	DeleteDeploymentAdvertisements(
		ctx context.Context,
		deploymentId string,
		filter lib_models_dep_advertisements.DeploymentAdvertisementsFilterReduced,
	) error
}
