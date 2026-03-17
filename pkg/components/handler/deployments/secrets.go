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

	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_deployment "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployment"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func (h *Handler) updateSecretValuesCache(
	ctx context.Context,
	userData userDataCollection,
	cache cacheCollection,
) error {
	var errs []string
	for _, secret := range userData.Secrets {
		for _, secretItem := range secret.Items {
			if secretItem.AsMount {
				continue
			}
			cacheKey := secret.Id + secretItem.Name
			var reqItem *string
			if secretItem.Name != "" {
				reqItem = &secretItem.Name
			}
			_, ok := cache.SecretValues[cacheKey]
			if !ok {
				var err error
				valueVariant, err, _ := h.smClient.GetValueVariant(ctx, models_external.SecretVariantRequest{
					ID:   secret.Id,
					Item: reqItem,
				})
				if err != nil {
					errs = append(errs, err.Error())
					continue
				}
				cache.SecretValues[cacheKey] = valueVariant
			}
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n")) // TODO
	}
	return nil
}

func (h *Handler) ensureSecretMounts(
	ctx context.Context,
	deployment extendedDeployment,
	userData userDataCollection,
) (map[string]models_external.SecretPathVariant, error) {
	secretMounts := make(map[string]models_external.SecretPathVariant)
	var errs []string
	err, _ := h.smClient.CleanPathVariants(ctx, deployment.Id)
	if err != nil {
		errs = append(errs, err.Error()) // TODO log instead?
	}
	for _, secret := range userData.Secrets {
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
				pathVariant, err, _ := h.smClient.InitPathVariant(ctx, models_external.SecretVariantRequest{
					ID:        secret.Id,
					Item:      reqItem,
					Reference: deployment.Id,
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
		err, _ := h.smClient.CleanPathVariants(ctx, deployment.Id)
		if err != nil {
			errs = append(errs, err.Error())
		}
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return secretMounts, nil
}

func getSelectedSecrets(
	module models_handler_module.Module,
	userInputs models_handler_deployment.UserInput,
	deploymentID string,
) (map[string]models_handler_storage.DeploymentSecret, error) {
	secrets := make(map[string]models_handler_storage.DeploymentSecret)
	var errs []string
	for reference, secret := range module.Secrets {
		id, ok := userInputs.Secrets[reference]
		if !ok {
			if secret.Required {
				errs = append(errs, fmt.Sprintf("secret %s required", reference))
			}
			continue
		}
		secrets[reference] = models_handler_storage.DeploymentSecret{
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

func getSecretItems(reference string, moduleServices map[string]models_external.ModuleService) []models_handler_storage.DeploymentSecretItem {
	items := make(map[string]models_handler_storage.DeploymentSecretItem)
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
