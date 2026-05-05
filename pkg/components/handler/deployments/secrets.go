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

package handler_deployments

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	lib_models_service "github.com/SENERGY-Platform/mgw-module-manager/lib/models/service"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_database "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
	models_handler_modules "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/modules"
)

func (h *Handler) updateSecretValuesCache(
	ctx context.Context,
	userDataSecrets map[string]models_handler_database.DeploymentSecret,
	cacheSecretValues map[string]models_external.SecretValueVariant,
) error {
	var errs []string
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
				valueVariant, err, _ := h.secretManagerClient.GetValueVariant(ctx, models_external.SecretVariantRequest{
					ID:   secret.Id,
					Item: reqItem,
				})
				if err != nil {
					errs = append(errs, err.Error())
					continue
				}
				cacheSecretValues[cacheKey] = valueVariant
			}
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func (h *Handler) createSecretMounts(
	ctx context.Context,
	deploymentId string,
	userDataSecrets map[string]models_handler_database.DeploymentSecret,
) (map[string]models_external.SecretPathVariant, error) {
	secretMounts := make(map[string]models_external.SecretPathVariant)
	var errs []string
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
				pathVariant, err, _ := h.secretManagerClient.InitPathVariant(ctx, models_external.SecretVariantRequest{
					ID:        secret.Id,
					Item:      reqItem,
					Reference: deploymentId,
				})
				if err != nil {
					errs = append(errs, err.Error())
					continue
				}
				secretMounts[key] = pathVariant
			}
		}
	}
	if len(errs) > 0 {
		err := h.removeSecretMounts(ctx, deploymentId)
		if err != nil {
			errs = append(errs, err.Error())
		}
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
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
	module models_handler_modules.Module,
	userInputSecrets map[string]string,
	deploymentID string,
) (map[string]models_handler_database.DeploymentSecret, error) {
	secrets := make(map[string]models_handler_database.DeploymentSecret)
	var errs []string
	for reference, secret := range module.Secrets {
		id, ok := userInputSecrets[reference]
		if !ok {
			if secret.Required {
				errs = append(errs, fmt.Sprintf("secret %s required", reference))
			}
			continue
		}
		secrets[reference] = models_handler_database.DeploymentSecret{
			Id:           id,
			DeploymentId: deploymentID,
			Reference:    reference,
			Items:        getSecretItems(reference, module.Services),
		}
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return secrets, nil
}

func getSecretItems(
	reference string,
	moduleServices map[string]models_external.ModuleLibService,
) []lib_models_service.DeploymentSecretItem {
	items := make(map[string]lib_models_service.DeploymentSecretItem)
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
