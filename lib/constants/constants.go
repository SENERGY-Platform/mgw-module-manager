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
	ActionInstall = "install"
	ActionChange  = "change"
	ActionRemove  = "remove"
)

type DeploymentState = int

const (
	DeploymentHealthy DeploymentState = iota + 1
	DeploymentUnhealthy
)

// Kinds of module items a deployment setup can lack an answer for when it is carried over to another module version.
const (
	CarryOverKindConfig       = "config"
	CarryOverKindSecret       = "secret"
	CarryOverKindHostResource = "host_resource"
	CarryOverKindFile         = "file"
	CarryOverKindFileGroup    = "file_group"
)

// Reasons why a deployment setup cannot answer a module item when it is carried over to another module version.
const (
	CarryOverReasonNotDeclared          = "not_declared"            // the input has no definition in the module to check a value against
	CarryOverReasonNotSet               = "not_set"                 // required item without a stored value and without a default
	CarryOverReasonEmptyValue           = "empty_value"             // stored value is an empty string, an empty list or an empty file
	CarryOverReasonTypeMismatch         = "type_mismatch"           // stored or global config value has another data type or slice declaration than the module declares, or the stored secret has another type
	CarryOverReasonInvalidValue         = "invalid_value"           // stored or global config value violates the constraints the module declares
	CarryOverReasonMissingPath          = "missing_path"            // a file of the file group has no path
	CarryOverReasonGlobalConfigNotFound = "global_config_not_found" // the global config the deployment refers to does not exist
	CarryOverReasonSecretNotFound       = "secret_not_found"        // the secret the deployment refers to does not exist in the secret manager
	CarryOverReasonSecretTypeUnknown    = "secret_type_unknown"     // the secret manager could not be read, so the type of the stored secret is unknown
)

// Highest kind of version change the auto-update applies, compared on the SemVer components literally, so
// v0.4.x to v0.5.0 is a minor change.
const (
	AutoUpdateLevelPatch = "patch"
	AutoUpdateLevelMinor = "minor"
	AutoUpdateLevelMajor = "major"
)

// Reasons why the auto-update did not update a module that has a newer version, or a deployment that lags its installed module.
const (
	AutoUpdateReasonLevelExceeded            = "level_exceeded"             // the version change exceeds the configured level
	AutoUpdateReasonDependencyLevelExceeded  = "dependency_level_exceeded"  // a dependency would change beyond the configured level, to an older or invalid version, or to another source or channel
	AutoUpdateReasonNewDependency            = "new_dependency"             // the update would install a module that is not installed yet
	AutoUpdateReasonNeedsInput               = "needs_input"                // the setup of an affected deployment cannot be carried over to the new version
	AutoUpdateReasonInvalidVersion           = "invalid_version"            // the installed or the available version is not a valid SemVer version
	AutoUpdateReasonModuleError              = "module_error"               // an affected installed module could not be read
	AutoUpdateReasonDeploymentError          = "deployment_error"           // the deployment of an affected module could not be read
	AutoUpdateReasonDependencyNotDeployed    = "dependency_not_deployed"    // an affected deployment depends on a module that has no deployment
	AutoUpdateReasonDependentRangeViolated   = "dependent_range_violated"   // a module would end up outside the version range another installed module declares for it
	AutoUpdateReasonDeploymentVariantDiffers = "deployment_variant_differs" // an affected deployment was created for another source or channel than its installed module, or for a version not older than the new one
)
