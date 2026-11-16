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

package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"slices"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/configs"
	helper_modfile "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/modfile"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	pkg_constants "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (s *Service) GetDeploymentsCarryOver(ctx context.Context, moduleIds []string) ([]lib_models.DeploymentCarryOver, error) {
	var secrets secretTypesSnapshot
	if len(moduleIds) > 0 {
		// Read before the lock, a slow secret manager must not block the service.
		secrets = s.readSecretTypesOf(ctx, moduleIds)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	currentJob, ok := s.jobsHandler.CurrentSlotJob(moduleJobSlotNum)
	if ok {
		return nil, lib_errors.New[lib_errors.ErrActiveJob](activeJobErrMsg(currentJob))
	}
	if len(moduleIds) == 0 {
		return nil, nil
	}
	modules, err := s.modulesHandler.GetModules(
		ctx,
		pkg_models.ModulesFilterWithName{ModulesFilter: pkg_models.ModulesFilter{Ids: moduleIds}},
		false,
	)
	if err != nil {
		return nil, err
	}
	deployments, err := s.deploymentsHandler.GetDeploymentsByModuleIds(ctx, pkg_models.DeploymentsFilterWithState{
		DeploymentsFilter: pkg_models.DeploymentsFilter{ModuleIds: slices.Collect(maps.Keys(modules))},
	})
	if err != nil {
		return nil, err
	}
	refs, err := s.readCarryOverRefs(ctx, deployments, secrets)
	if err != nil {
		return nil, err
	}
	results := make([]lib_models.DeploymentCarryOver, 0, len(moduleIds))
	for _, id := range slices.Sorted(slices.Values(moduleIds)) {
		result := lib_models.DeploymentCarryOver{ModuleId: id}
		module, ok := modules[id]
		if !ok {
			result.ErrorResult = lib_models.NewErrorResult("module not installed")
			results = append(results, result)
			continue
		}
		deployment, ok := deployments[id]
		if !ok {
			result.ErrorResult = lib_models.NewErrorResult("module not deployed")
			results = append(results, result)
			continue
		}
		input, missing, err := carryOverInstalled(module, deployment, refs)
		switch {
		case err != nil:
			result.ErrorResult = lib_models.NewErrorResult(err.Error())
		case len(missing) > 0:
			result.Missing = missing
		default:
			result.Input = &input
		}
		results = append(results, result)
	}
	return results, nil
}

// carryOverRefs holds what deployments refer to outside of themselves.
type carryOverRefs struct {
	globalConfigs  map[string]pkg_models.Config // by ID, a deleted one is absent
	secretTypes    map[string]string            // secret manager type by secret ID, a deleted secret is absent
	secretTypesErr error                        // set if the secret manager could not be read, no stored secret is carried then
}

// errSecretTypesNotRead marks a secret stored after the secret types were read.
var errSecretTypesNotRead = errors.New("secret types not read")

// secretTypesSnapshot holds the secret types as read from the secret manager, a remote service that is asked before
// s.mu is taken.
type secretTypesSnapshot struct {
	read  bool // false if no deployment stored a secret, the secret manager was not asked then
	types map[string]string
	err   error
}

// readSecretTypesOf reads the deployments of the modules, all if moduleIds is empty, and then their secret types. A
// failed read is kept in the snapshot, so only deployments with secrets need input. Must be called without s.mu held.
func (s *Service) readSecretTypesOf(ctx context.Context, moduleIds []string) secretTypesSnapshot {
	deployments, err := s.deploymentsHandler.GetDeploymentsByModuleIds(ctx, pkg_models.DeploymentsFilterWithState{
		DeploymentsFilter: pkg_models.DeploymentsFilter{ModuleIds: moduleIds},
	})
	if err != nil {
		logger.WarnContext(ctx, "carry over, read deployments for secret types", slog_keys.Error, err)
		return secretTypesSnapshot{read: true, err: err}
	}
	return s.readSecretTypes(ctx, deployments)
}

// readSecretTypes asks the secret manager only if a deployment stores a secret. Must be called without s.mu held.
func (s *Service) readSecretTypes(ctx context.Context, deployments map[string]pkg_models.Deployment) secretTypesSnapshot {
	if !storesSecrets(deployments) {
		return secretTypesSnapshot{}
	}
	types, err := s.deploymentsHandler.GetSecretTypes(ctx)
	if err != nil {
		logger.WarnContext(ctx, "carry over, read secret types", slog_keys.Error, err)
	}
	return secretTypesSnapshot{read: true, types: types, err: err}
}

func storesSecrets(deployments map[string]pkg_models.Deployment) bool {
	for _, deployment := range deployments {
		for _, secret := range deployment.Secrets {
			if secret.Id != "" {
				return true
			}
		}
	}
	return false
}

// readCarryOverRefs reads the global configs the deployments refer to and takes their secret types from secrets. If
// the secret types could not be read, the error is kept in the result instead of returned, so only deployments with
// secrets need input.
func (s *Service) readCarryOverRefs(ctx context.Context, deployments map[string]pkg_models.Deployment, secrets secretTypesSnapshot) (carryOverRefs, error) {
	var refs carryOverRefs
	var err error
	if refs.globalConfigs, err = s.readCarryOverGlobalConfigs(ctx, deployments); err != nil {
		return carryOverRefs{}, err
	}
	if storesSecrets(deployments) {
		refs.secretTypes, refs.secretTypesErr = secrets.types, secrets.err
		if !secrets.read {
			refs.secretTypesErr = errSecretTypesNotRead
		}
	}
	return refs, nil
}

// readCarryOverGlobalConfigs reads the global configs the deployments refer to, by ID. A deleted one is absent.
func (s *Service) readCarryOverGlobalConfigs(ctx context.Context, deployments map[string]pkg_models.Deployment) (map[string]pkg_models.Config, error) {
	ids := make(map[string]struct{})
	for _, deployment := range deployments {
		for _, globalConfig := range deployment.GlobalConfigs {
			if globalConfig.Id != "" {
				ids[globalConfig.Id] = struct{}{}
			}
		}
	}
	if len(ids) == 0 {
		// The handler reads all global configs for an empty ID list.
		return nil, nil
	}
	return s.globalConfigsHandler.GetGlobalConfigs(ctx, slices.Sorted(maps.Keys(ids)))
}

// carryOverInstalled checks the deployment of an installed module against the declaration of that module.
func carryOverInstalled(
	module pkg_models.Module,
	deployment pkg_models.Deployment,
	refs carryOverRefs,
) (lib_models.DeploymentUserInput, []lib_models.CarryOverMissing, error) {
	if module.Err != nil {
		return lib_models.DeploymentUserInput{}, nil, fmt.Errorf("module: %w", module.Err)
	}
	return carryOver(module.ModuleLibModule, fileDefaults(module.Files), deployment, refs)
}

// carryOverRepository checks the deployment against a module that is not installed yet. The default data is read
// from the repository the way the installation reads it, an unreadable file fails the check as it fails the update.
func carryOverRepository(
	mod modWrapper,
	deployment pkg_models.Deployment,
	refs carryOverRefs,
) (lib_models.DeploymentUserInput, []lib_models.CarryOverMissing, error) {
	files, err := helper_modfile.GetFiles(mod.FS, mod.Mod.Files)
	if err != nil {
		return lib_models.DeploymentUserInput{}, nil, fmt.Errorf("read files: %w", err)
	}
	return carryOver(mod.Mod, fileDefaults(files), deployment, refs)
}

// fileDefaults states per file reference whether the file has default data, an empty file is no default to the
// deployment either.
func fileDefaults(files map[string]pkg_models.ModuleFile) map[string]bool {
	defaults := make(map[string]bool, len(files))
	for ref, file := range files {
		defaults[ref] = len(file.DefaultData) > 0
	}
	return defaults
}

// carryOver builds the user input that carries the setup of the deployment over to the declaration mod. Only the
// references mod declares as inputs are evaluated and carried, anything else would be rejected as unknown. A non-empty
// missing list means the setup does not answer all of them and the returned input must not be used. fileDefaults states
// per file reference whether mod provides default data, refs holds what the deployment refers to outside of itself.
func carryOver(
	mod external_models.ModuleLibModule,
	fileDefaults map[string]bool,
	deployment pkg_models.Deployment,
	refs carryOverRefs,
) (lib_models.DeploymentUserInput, []lib_models.CarryOverMissing, error) {
	if deployment.Err != nil {
		return lib_models.DeploymentUserInput{}, nil, fmt.Errorf("deployment: %w", deployment.Err)
	}
	input := lib_models.DeploymentUserInput{
		ModuleId:      mod.ID,
		HostResources: make(map[string]string),
		Secrets:       make(map[string]string),
		Configs:       make(map[string]interface{}),
		GlobalConfigs: make(map[string]string),
		Files:         make(map[string]string),
		FileGroups:    make(map[string]map[string]lib_models.DeploymentFileGroupUserInput),
	}
	var missing []lib_models.CarryOverMissing
	need := func(kind, ref, reason string) {
		missing = append(missing, lib_models.CarryOverMissing{Kind: kind, Ref: ref, Reason: reason})
	}

	for _, ref := range slices.Sorted(maps.Keys(mod.Inputs.Configs)) {
		if reason := carryOverConfig(mod.Configs, deployment, refs.globalConfigs, ref, &input); reason != "" {
			need(lib_constants.CarryOverKindConfig, ref, reason)
		}
	}

	// An unanswered optional secret or host resource stays unanswered; only a required one is a question.
	for _, ref := range slices.Sorted(maps.Keys(mod.Inputs.Secrets)) {
		declared, ok := mod.Secrets[ref]
		if !ok {
			need(lib_constants.CarryOverKindSecret, ref, lib_constants.CarryOverReasonNotDeclared)
			continue
		}
		if secret := deployment.Secrets[ref]; secret.Id != "" {
			if reason := carryOverSecret(declared.Type, secret.Id, refs); reason != "" {
				need(lib_constants.CarryOverKindSecret, ref, reason)
				continue
			}
			input.Secrets[ref] = secret.Id
			continue
		}
		if declared.Required {
			need(lib_constants.CarryOverKindSecret, ref, lib_constants.CarryOverReasonNotSet)
		}
	}
	for _, ref := range slices.Sorted(maps.Keys(mod.Inputs.Resources)) {
		if resource := deployment.HostResources[ref]; resource.Id != "" {
			input.HostResources[ref] = resource.Id
			continue
		}
		declared, ok := mod.HostResources[ref]
		switch {
		case !ok:
			need(lib_constants.CarryOverKindHostResource, ref, lib_constants.CarryOverReasonNotDeclared)
		case declared.Required:
			need(lib_constants.CarryOverKindHostResource, ref, lib_constants.CarryOverReasonNotSet)
		}
	}

	for _, ref := range slices.Sorted(maps.Keys(mod.Inputs.Files)) {
		if stored, ok := deployment.Files[ref]; ok {
			if len(stored.Data) == 0 {
				// Neither content to carry nor an absence the default answers.
				need(lib_constants.CarryOverKindFile, ref, lib_constants.CarryOverReasonEmptyValue)
				continue
			}
			input.Files[ref] = base64.StdEncoding.EncodeToString(stored.Data)
			continue
		}
		declared, ok := mod.Files[ref]
		switch {
		case !ok:
			need(lib_constants.CarryOverKindFile, ref, lib_constants.CarryOverReasonNotDeclared)
		case declared.Required && !fileDefaults[ref]:
			need(lib_constants.CarryOverKindFile, ref, lib_constants.CarryOverReasonNotSet)
		}
	}

	for _, ref := range slices.Sorted(maps.Keys(mod.Inputs.FileGroups)) {
		files := deployment.FileGroups[ref].Files
		if len(files) == 0 {
			// There is no required flag for a file group, so an empty one is not a question.
			continue
		}
		group := make(map[string]lib_models.DeploymentFileGroupUserInput, len(files))
		valid := true
		for _, file := range files {
			if file.Path == "" {
				need(lib_constants.CarryOverKindFileGroup, ref, lib_constants.CarryOverReasonMissingPath)
				valid = false
				break
			}
			group[file.Path] = lib_models.DeploymentFileGroupUserInput{
				Format: file.Format,
				Data:   base64.StdEncoding.EncodeToString(file.Data),
			}
		}
		if valid {
			input.FileGroups[ref] = group
		}
	}
	return input, missing, nil
}

// carryOverSecret returns the reason if the stored secret is not known to have the declared type.
func carryOverSecret(declaredType, id string, refs carryOverRefs) string {
	if refs.secretTypesErr != nil {
		return lib_constants.CarryOverReasonSecretTypeUnknown
	}
	storedType, ok := refs.secretTypes[id]
	switch {
	case !ok:
		return lib_constants.CarryOverReasonSecretNotFound
	case storedType != declaredType:
		return lib_constants.CarryOverReasonTypeMismatch
	}
	return ""
}

// carryOverConfig carries one config into input and returns the reason if the setup does not answer it.
func carryOverConfig(
	configs external_models.ModuleLibConfigs,
	deployment pkg_models.Deployment,
	globalConfigs map[string]pkg_models.Config,
	ref string,
	input *lib_models.DeploymentUserInput,
) string {
	declared, ok := configs[ref]
	if !ok {
		return lib_constants.CarryOverReasonNotDeclared
	}
	if selected := deployment.GlobalConfigs[ref]; selected.Id != "" {
		// Passes what ValidateValue passes, the check the deployment update applies; the type check only names the reason.
		globalConfig, ok := globalConfigs[selected.Id]
		switch {
		case !ok:
			return lib_constants.CarryOverReasonGlobalConfigNotFound
		case globalConfig.DataType != helper_configs.GetDataType(declared.DataType) || globalConfig.IsSlice != declared.IsSlice:
			return lib_constants.CarryOverReasonTypeMismatch
		case helper_configs.ValidateValue(globalConfig.Value, declared) != nil:
			return lib_constants.CarryOverReasonInvalidValue
		}
		input.GlobalConfigs[ref] = selected.Id
		return ""
	}
	stored, ok := deployment.Configs[ref]
	if !ok {
		if declared.Required && declared.Default == nil {
			return lib_constants.CarryOverReasonNotSet
		}
		return ""
	}
	// An empty value is asked for rather than resolved: sending it pins the empty value over the new default, and an
	// empty list is rejected against declared options, which fails the whole update.
	if hasEmptyValue(stored.Value, false) {
		return lib_constants.CarryOverReasonEmptyValue
	}
	if stored.DataType != helper_configs.GetDataType(declared.DataType) || stored.IsSlice != declared.IsSlice {
		return lib_constants.CarryOverReasonTypeMismatch
	}
	if declared.Required && hasEmptyValue(stored.Value, true) {
		return lib_constants.CarryOverReasonEmptyValue
	}
	if err := helper_configs.ValidateValue(stored.Value, declared); err != nil {
		return lib_constants.CarryOverReasonInvalidValue
	}
	input.Configs[ref] = helper_configs.ValueToInterface(stored.Value)
	return ""
}

// hasEmptyValue reports whether the value is an empty string or an empty slice; with anyItem, also a slice that
// contains an empty string.
func hasEmptyValue(value pkg_models.Value, anyItem bool) bool {
	if value.IsSlice {
		if len(value.StringSlice)+len(value.Int64Slice)+len(value.Float64Slice)+len(value.BoolSlice) == 0 {
			return true
		}
		return anyItem && slices.Contains(value.StringSlice, "")
	}
	return value.DataType == pkg_constants.ValueDataTypeString && value.String == ""
}
