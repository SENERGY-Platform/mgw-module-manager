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

package deployments

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_errors "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/errors"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func (h *Handler) updateSecretValuesCache(
	ctx context.Context,
	userDataSecrets map[string]pkg_models.DeploymentSecret,
	cacheSecretValues map[string]external_models.SmSecretValueVariant,
) error {
	var errs []error
	for _, secret := range userDataSecrets {
		for _, secretItem := range secret.Items {
			if secretItem.AsMount {
				continue
			}
			cacheKey := secret.Id + secretItem.Name
			var reqItem *string
			if secretItem.Name != "" {
				reqItem = &secretItem.Name
			}
			_, ok := cacheSecretValues[cacheKey]
			if !ok {
				var err error
				valueVariant, err, _ := h.secretManagerClient.GetValueVariant(ctx, external_models.SmSecretVariantRequest{
					ID:   secret.Id,
					Item: reqItem,
				})
				if err != nil {
					errs = append(errs, fmt.Errorf("'%s' %w", secret.Id, err))
					continue
				}
				cacheSecretValues[cacheKey] = valueVariant
			}
		}
	}
	if len(errs) > 0 {
		return helper_errors.Join(errs...)
	}
	return nil
}

func (h *Handler) createSecretMounts(
	ctx context.Context,
	deploymentId string,
	userDataSecrets map[string]pkg_models.DeploymentSecret,
) (map[string]external_models.SmSecretPathVariant, error) {
	secretMounts := make(map[string]external_models.SmSecretPathVariant)
	var errs []error
	for _, secret := range userDataSecrets {
		for _, secretItem := range secret.Items {
			if secretItem.AsEnv {
				continue
			}
			key := secret.Id + secretItem.Name
			var reqItem *string
			if secretItem.Name != "" {
				reqItem = &secretItem.Name
			}
			_, ok := secretMounts[key]
			if !ok {
				pathVariant, err, _ := h.secretManagerClient.InitPathVariant(ctx, external_models.SmSecretVariantRequest{
					ID:        secret.Id,
					Item:      reqItem,
					Reference: deploymentId,
				})
				if err != nil {
					errs = append(errs, fmt.Errorf("'%s' %w", secret.Id, err))
					continue
				}
				secretMounts[key] = pathVariant
			}
		}
	}
	if len(errs) > 0 {
		err := h.removeSecretMounts(ctx, deploymentId)
		if err != nil {
			logger.ErrorContext(ctx, "remove created secret mounts", slog_keys.DeploymentId, deploymentId, slog_keys.Error, err)
		}
		return nil, helper_errors.Join(errs...)
	}
	return secretMounts, nil
}

func (h *Handler) removeSecretMounts(ctx context.Context, deploymentId string) error {
	err, _ := h.secretManagerClient.CleanPathVariants(ctx, deploymentId)
	if err != nil {
		return err
	}
	return nil
}

func getSelectedSecrets(
	module pkg_models.Module,
	userInputSecrets map[string]string,
	deploymentID string,
) (map[string]pkg_models.DeploymentSecret, error) {
	secrets := make(map[string]pkg_models.DeploymentSecret)
	var required []string
	for reference, secret := range module.Secrets {
		id, ok := userInputSecrets[reference]
		if !ok {
			if secret.Required {
				required = append(required, reference)
			}
			continue
		}
		secrets[reference] = pkg_models.DeploymentSecret{
			Id:           id,
			DeploymentId: deploymentID,
			Reference:    reference,
			Items:        getSecretItems(reference, module.Services),
		}
	}
	if len(required) > 0 {
		return nil, errors.New(fmt.Sprintf("required secrets: %s", strings.Join(required, ", ")))
	}
	return secrets, nil
}

func getSecretItems(
	reference string,
	moduleServices map[string]external_models.ModuleLibService,
) []lib_models.DeploymentSecretItem {
	items := make(map[string]lib_models.DeploymentSecretItem)
	for _, moduleService := range moduleServices {
		for _, target := range moduleService.SecretVars {
			if target.Ref == reference {
				item, ok := items[target.Item]
				if !ok {
					item.Name = target.Item
				}
				item.AsEnv = true
				items[target.Item] = item
			}
		}
		for _, target := range moduleService.SecretMounts {
			if target.Ref == reference {
				item, ok := items[target.Item]
				if !ok {
					item.Name = target.Item
				}
				item.AsMount = true
				items[target.Item] = item
			}
		}
	}
	return slices.Collect(maps.Values(items))
}
