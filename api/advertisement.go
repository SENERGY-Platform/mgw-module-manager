/*
 * Copyright 2024 InfAI (CC SES)
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

package api

import (
	"context"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

func (a *Api) QueryAdvertisements(ctx context.Context, filter model.DepAdvFilter) ([]model.DepAdvertisement, error) {
	list, err := a.advHandler.List(ctx, filter)
	if err != nil {
		return nil, newApiErr(fmt.Sprintf("query advertisements (module_id=%s origin=%s ref=%s)", filter.ModuleID, filter.Origin, filter.Ref), err)
	}
	return list, nil
}

func (a *Api) GetAdvertisement(ctx context.Context, dID, ref string) (model.DepAdvertisement, error) {
	metaStr := fmt.Sprintf("get advertisement (deployment_id=%s ref=%s)", dID, ref)
	_, err := a.deploymentHandler.Get(ctx, dID, false, false, false, false)
	if err != nil {
		return model.DepAdvertisement{}, newApiErr(metaStr, err)
	}
	adv, err := a.advHandler.Get(ctx, dID, ref)
	if err != nil {
		return model.DepAdvertisement{}, newApiErr(metaStr, err)
	}
	return adv, nil
}

func (a *Api) GetAdvertisements(ctx context.Context, dID string) (map[string]model.DepAdvertisement, error) {
	metaStr := fmt.Sprintf("get advertisements (deployment_id=%s)", dID)
	_, err := a.deploymentHandler.Get(ctx, dID, false, false, false, false)
	if err != nil {
		return nil, newApiErr(metaStr, err)
	}
	ads, err := a.advHandler.GetAll(ctx, dID)
	if err != nil {
		return nil, newApiErr(metaStr, err)
	}
	return ads, nil
}

func (a *Api) PutAdvertisement(ctx context.Context, dID string, adv model.DepAdvertisementBase) error {
	metaStr := fmt.Sprintf("put advertisement (deployment_id=%s)", dID)
	dep, err := a.deploymentHandler.Get(ctx, dID, false, false, false, false)
	if err != nil {
		return newApiErr(metaStr, err)
	}
	if err = a.advHandler.Put(ctx, dep.Module.ID, dID, adv); err != nil {
		return newApiErr(metaStr, err)
	}
	return nil
}

func (a *Api) PutAdvertisements(ctx context.Context, dID string, ads map[string]model.DepAdvertisementBase) error {
	metaStr := fmt.Sprintf("put advertisements (deployment_id=%s)", dID)
	dep, err := a.deploymentHandler.Get(ctx, dID, false, false, false, false)
	if err != nil {
		return newApiErr(metaStr, err)
	}
	if err = a.advHandler.PutAll(ctx, dep.Module.ID, dID, ads); err != nil {
		return newApiErr(metaStr, err)
	}
	return nil
}

func (a *Api) DeleteAdvertisement(ctx context.Context, dID, ref string) error {
	metaStr := fmt.Sprintf("delete advertisement (deployment_id=%s ref=%s)", dID, ref)
	_, err := a.deploymentHandler.Get(ctx, dID, false, false, false, false)
	if err != nil {
		return newApiErr(metaStr, err)
	}
	if err = a.advHandler.Delete(ctx, dID, ref); err != nil {
		return newApiErr(metaStr, err)
	}
	return nil
}

func (a *Api) DeleteAdvertisements(ctx context.Context, dID string) error {
	metaStr := fmt.Sprintf("delete advertisements (deployment_id=%s)", dID)
	_, err := a.deploymentHandler.Get(ctx, dID, false, false, false, false)
	if err != nil {
		return newApiErr(metaStr, err)
	}
	if err = a.advHandler.DeleteAll(ctx, dID); err != nil {
		return newApiErr(metaStr, err)
	}
	return nil
}
