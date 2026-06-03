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

package constants

const (
	HttpPathModulesCollection                    = "modules"
	HttpPathModuleResource                       = "modules/:MOD_ID"
	HttpPathModulesChangeRequestResource         = "modules-change-request"
	HttpPathModulesAvailableUpdatesCountResource = "modules-available-updates"

	HttpPathRepositoriesCollection      = "repositories"
	HttpPathRepositoryResource          = "repositories/:SOURCE"
	HttpPathRepositoryModulesCollection = "repository-modules"

	HttpPathDeploymentRequestResource = "deployment-request"
	HttpPathDeploymentsCollection     = "deployments"
	HttpPathRecreateDeployments       = "deployments-recreate"
	HttpPathEnableDeployments         = "deployments-enable"
	HttpPathDisableDeployments        = "deployments-disable"

	HttpPathAuxiliaryDeploymentsCollection                 = "deployments/:DEP_ID/auxiliary/deployments"
	HttpPathAuxiliaryDeploymentResource                    = "deployments/:DEP_ID/auxiliary/deployments/:AUX_DEP_ID"
	HttpPathReducedAuxiliaryDeploymentsCollection          = "deployments/:DEP_ID/auxiliary/deployments-reduced"
	HttpPathRecreateAuxiliaryDeployments                   = "deployments/:DEP_ID/auxiliary/deployments-recreate"
	HttpPathEnableAuxiliaryDeployments                     = "deployments/:DEP_ID/auxiliary/deployments-enable"
	HttpPathDisableAuxiliaryDeployments                    = "deployments/:DEP_ID/auxiliary/deployments-disable"
	HttpPathAuxiliaryDeploymentVolumesCollection           = "deployments/:DEP_ID/auxiliary/volumes"
	HttpPathAuxiliaryDeploymentVolumeResource              = "deployments/:DEP_ID/auxiliary/volumes/:AUX_VOL_REF"
	HttpPathAuxiliaryDeploymentVolumesWithMountsCollection = "deployments/:DEP_ID/auxiliary/volumes-with-mounts"

	HttpPathDeploymentAdvertisementsQueryCollection = "deployment-advertisements"
	HttpPathDeploymentAdvertisementQueryResource    = "deployment-advertisements/:ADV_ID"
	HttpPathDeploymentAdvertisementsCollection      = "deployments/:DEP_ID/advertisements"
	HttpPathDeploymentAdvertisementResource         = "deployments/:DEP_ID/advertisements/:ADV_REF"
	HttpPathDeploymentAdvertisementByIdResource     = "deployments/:DEP_ID/advertisements-by-id/:ADV_ID"

	HttpPathGlobalConfigsCollection = "global-configs"
	HttpPathGlobalConfigResource    = "global-configs/:CFG_ID"

	HttpPathJobsCollection = "jobs"
	HttpPathJobResource    = "jobs/:JOB_ID"
	HttpPathCancelJobs     = "jobs-cancel"

	HttpPathDeploymentResultResource                = "results/deployments/:JOB_ID"
	HttpPathUpdateDeploymentResultResource          = "results/deployments-update/:JOB_ID"
	HttpPathChangeModulesResultResource             = "results/modules-change/:JOB_ID"
	HttpPathRefreshRepositoriesResultResource       = "results/repositories-refresh/:JOB_ID"
	HttpPathAuxiliaryDeploymentsResultResource      = "results/auxiliary-deployments/:JOB_ID"
	HttpPathCreateAuxiliaryDeploymentResultResource = "results/auxiliary-deployment-create/:JOB_ID"
	HttpPathUpdateAuxiliaryDeploymentResultResource = "results/auxiliary-deployment-update/:JOB_ID"

	HttpPathDeploymentsHealthCollection = "health/deployments"

	HttpPathServiceInfoResource = "info"
)

const (
	HttpHeaderCoreId    = "X-Core-Id"
	HttpHeaderManagerId = "X-Manager-Id"
	HttpHeaderRuntimeId = "X-Runtime-Id"
	HttpHeaderRequestId = "X-Request-Id"
	HttpHeaderErrorCode = "X-Err-Code"
	HttpHeaderApiVer    = "X-Version"
	HttpHeaderSrvName   = "X-Service"
)
