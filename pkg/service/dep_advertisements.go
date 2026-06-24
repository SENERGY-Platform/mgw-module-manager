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

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

func (s *Service) QueryDeploymentAdvertisements(
	ctx context.Context,
	filter lib_models.DeploymentAdvertisementsFilter,
) ([]lib_models.DeploymentAdvertisementReduced, error) {
	depAdvMap, err := s.depAdvertisementsHandler.GetAdvertisements(ctx, filter)
	if err != nil {
		return nil, err
	}
	var depAdvs []lib_models.DeploymentAdvertisementReduced
	for _, depAdv := range depAdvMap {
		depAdvs = append(depAdvs, lib_models.DeploymentAdvertisementReduced{
			Id:        depAdv.Id,
			ModuleId:  depAdv.ModuleId,
			Reference: depAdv.Reference,
			Timestamp: depAdv.Timestamp,
			Items:     depAdv.Items,
		})
	}
	return depAdvs, nil
}

func (s *Service) QueryDeploymentAdvertisement(ctx context.Context, id string) (lib_models.DeploymentAdvertisementReduced, error) {
	depAdv, err := s.depAdvertisementsHandler.GetAdvertisementById(ctx, id)
	if err != nil {
		return lib_models.DeploymentAdvertisementReduced{}, err
	}
	return lib_models.DeploymentAdvertisementReduced{
		Id:        depAdv.Id,
		ModuleId:  depAdv.ModuleId,
		Reference: depAdv.Reference,
		Timestamp: depAdv.Timestamp,
		Items:     depAdv.Items,
	}, nil
}

func (s *Service) GetDeploymentAdvertisement(
	ctx context.Context,
	deploymentId string,
	reference string,
) (lib_models.DeploymentAdvertisement, error) {
	err := s.deploymentsHandler.CheckDeployment(ctx, deploymentId)
	if err != nil {
		return lib_models.DeploymentAdvertisement{}, err
	}
	return s.depAdvertisementsHandler.GetAdvertisement(ctx, deploymentId, reference)
}

func (s *Service) GetDeploymentAdvertisementById(
	ctx context.Context,
	deploymentId string,
	id string,
) (lib_models.DeploymentAdvertisement, error) {
	err := s.deploymentsHandler.CheckDeployment(ctx, deploymentId)
	if err != nil {
		return lib_models.DeploymentAdvertisement{}, err
	}
	return s.depAdvertisementsHandler.GetAdvertisementById(ctx, id)
}

func (s *Service) GetDeploymentAdvertisements(
	ctx context.Context,
	deploymentId string,
	filter lib_models.DeploymentAdvertisementsFilterReduced,
) (map[string]lib_models.DeploymentAdvertisement, error) {
	err := s.deploymentsHandler.CheckDeployment(ctx, deploymentId)
	if err != nil {
		return nil, err
	}
	return s.depAdvertisementsHandler.GetAdvertisements(ctx, lib_models.DeploymentAdvertisementsFilter{
		DeploymentId: deploymentId,
		Ids:          filter.Ids,
		References:   filter.References,
	})
}

func (s *Service) PutDeploymentAdvertisement(
	ctx context.Context,
	deploymentId string,
	reference string,
	items map[string]string,
) (string, error) {
	deployment, err := s.deploymentsHandler.GetDeployment(ctx, deploymentId)
	if err != nil {
		return "", err
	}
	return s.depAdvertisementsHandler.PutAdvertisement(
		ctx,
		deployment.ModuleId,
		deployment.Id,
		reference,
		items,
	)
}

func (s *Service) PutDeploymentAdvertisements(
	ctx context.Context,
	deploymentId string,
	inputs []lib_models.DeploymentAdvertisementInput,
	incremental bool,
) (map[string]string, error) {
	deployment, err := s.deploymentsHandler.GetDeployment(ctx, deploymentId)
	if err != nil {
		return nil, err
	}
	return s.depAdvertisementsHandler.PutAdvertisements(
		ctx,
		deployment.ModuleId,
		deployment.Id,
		inputs,
		incremental,
	)
}

func (s *Service) DeleteDeploymentAdvertisement(ctx context.Context, deploymentId string, reference string) error {
	err := s.deploymentsHandler.CheckDeployment(ctx, deploymentId)
	if err != nil {
		return err
	}
	return s.depAdvertisementsHandler.DeleteAdvertisements(
		ctx,
		deploymentId,
		lib_models.DeploymentAdvertisementsFilterReduced{References: []string{reference}},
		false,
	)
}

func (s *Service) DeleteDeploymentAdvertisements(
	ctx context.Context,
	deploymentId string,
	filter lib_models.DeploymentAdvertisementsFilterReduced,
	allowAll bool,
) error {
	err := s.deploymentsHandler.CheckDeployment(ctx, deploymentId)
	if err != nil {
		return err
	}
	return s.depAdvertisementsHandler.DeleteAdvertisements(ctx, deploymentId, filter, allowAll)
}
