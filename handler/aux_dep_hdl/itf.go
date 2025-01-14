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

package aux_dep_hdl

import (
	"context"
	"database/sql/driver"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

type StorageHandler interface {
	BeginTransaction(ctx context.Context) (driver.Tx, error)
	ListAuxDep(ctx context.Context, dID string, filter lib_model.AuxDepFilter, assets bool) (map[string]lib_model.AuxDeployment, error)
	ReadAuxDep(ctx context.Context, aID string, assets bool) (lib_model.AuxDeployment, error)
	CreateAuxDep(ctx context.Context, tx driver.Tx, auxDep lib_model.AuxDepBase) (string, error)
	UpdateAuxDep(ctx context.Context, tx driver.Tx, auxDep lib_model.AuxDepBase) error
	DeleteAuxDep(ctx context.Context, tx driver.Tx, aID string) error
	CreateAuxDepContainer(ctx context.Context, tx driver.Tx, aID string, auxDepContainer lib_model.AuxDepContainer) error
	DeleteAuxDepContainer(ctx context.Context, tx driver.Tx, aID string) error
}
