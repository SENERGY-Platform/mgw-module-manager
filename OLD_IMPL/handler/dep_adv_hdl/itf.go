/*
 * Copyright 2025 InfAI (CC SES)
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

package dep_adv_hdl

import (
	"context"
	"database/sql/driver"
	"github.com/SENERGY-Platform/mgw-module-manager/model"
)

type StorageHandler interface {
	BeginTransaction(ctx context.Context) (driver.Tx, error)
	ListDepAdv(ctx context.Context, filter model.DepAdvFilter) (map[string]model.DepAdvertisement, error)
	ReadDepAdv(ctx context.Context, dID, ref string) (model.DepAdvertisement, error)
	CreateDepAdv(ctx context.Context, tx driver.Tx, adv model.DepAdvertisement) (string, error)
	DeleteDepAdv(ctx context.Context, tx driver.Tx, dID, ref string) error
	DeleteAllDepAdv(ctx context.Context, tx driver.Tx, dID string) error
}
