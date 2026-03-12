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

package manager

import (
	"context"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

func (m *Manager) QueryDepAdvertisements(ctx context.Context, filter model.DepAdvFilter) ([]model.DepAdvertisement, error) {
	list, err := m.advHandler.List(ctx, filter)
	if err != nil {
		return nil, newApiErr(fmt.Sprintf("query advertisements (module_id=%s origin=%s ref=%s)", filter.ModuleID, filter.Origin, filter.Ref), err)
	}
	return list, nil
}

func (m *Manager) GetDepAdvertisement(ctx context.Context, dID, ref string) (model.DepAdvertisement, error) {
	metaStr := fmt.Sprintf("get advertisement (deployment_id=%s ref=%s)", dID, ref)
	_, err := m.deploymentHandler.Get(ctx, dID, false, false, false, false)
	if err != nil {
		return model.DepAdvertisement{}, newApiErr(metaStr, err)
	}
	adv, err := m.advHandler.Get(ctx, dID, ref)
	if err != nil {
		return model.DepAdvertisement{}, newApiErr(metaStr, err)
	}
	return adv, nil
}

func (m *Manager) GetDepAdvertisements(ctx context.Context, dID string) (map[string]model.DepAdvertisement, error) {
	metaStr := fmt.Sprintf("get advertisements (deployment_id=%s)", dID)
	_, err := m.deploymentHandler.Get(ctx, dID, false, false, false, false)
	if err != nil {
		return nil, newApiErr(metaStr, err)
	}
	ads, err := m.advHandler.GetAll(ctx, dID)
	if err != nil {
		return nil, newApiErr(metaStr, err)
	}
	return ads, nil
}

func (m *Manager) PutDepAdvertisement(ctx context.Context, dID string, adv model.DepAdvertisementBase) error {
	metaStr := fmt.Sprintf("put advertisement (deployment_id=%s)", dID)
	dep, err := m.deploymentHandler.Get(ctx, dID, false, false, false, false)
	if err != nil {
		return newApiErr(metaStr, err)
	}
	if err = m.advHandler.Put(ctx, dep.Module.ID, dID, adv); err != nil {
		return newApiErr(metaStr, err)
	}
	return nil
}

func (m *Manager) PutDepAdvertisements(ctx context.Context, dID string, ads map[string]model.DepAdvertisementBase) error {
	metaStr := fmt.Sprintf("put advertisements (deployment_id=%s)", dID)
	dep, err := m.deploymentHandler.Get(ctx, dID, false, false, false, false)
	if err != nil {
		return newApiErr(metaStr, err)
	}
	if err = m.advHandler.PutAll(ctx, dep.Module.ID, dID, ads); err != nil {
		return newApiErr(metaStr, err)
	}
	return nil
}

func (m *Manager) DeleteDepAdvertisement(ctx context.Context, dID, ref string) error {
	metaStr := fmt.Sprintf("delete advertisement (deployment_id=%s ref=%s)", dID, ref)
	_, err := m.deploymentHandler.Get(ctx, dID, false, false, false, false)
	if err != nil {
		return newApiErr(metaStr, err)
	}
	if err = m.advHandler.Delete(ctx, dID, ref); err != nil {
		return newApiErr(metaStr, err)
	}
	return nil
}

func (m *Manager) DeleteDepAdvertisements(ctx context.Context, dID string) error {
	metaStr := fmt.Sprintf("delete advertisements (deployment_id=%s)", dID)
	_, err := m.deploymentHandler.Get(ctx, dID, false, false, false, false)
	if err != nil {
		return newApiErr(metaStr, err)
	}
	if err = m.advHandler.DeleteAll(ctx, dID); err != nil {
		return newApiErr(metaStr, err)
	}
	return nil
}
