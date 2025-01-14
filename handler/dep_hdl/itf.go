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

package dep_hdl

import (
	"context"
	"database/sql/driver"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

type StorageHandler interface {
	BeginTransaction(ctx context.Context) (driver.Tx, error)
	ListDep(ctx context.Context, filter lib_model.DepFilter, dependencyInfo, assets, containers bool) (map[string]lib_model.Deployment, error)
	ReadDep(ctx context.Context, dID string, dependencyInfo, assets, containers bool) (lib_model.Deployment, error)
	CreateDep(ctx context.Context, tx driver.Tx, depBase lib_model.DepBase) (string, error)
	UpdateDep(ctx context.Context, tx driver.Tx, depBase lib_model.DepBase) error
	DeleteDep(ctx context.Context, tx driver.Tx, dID string) error
	ReadDepTree(ctx context.Context, dID string, assets, containers bool) (map[string]lib_model.Deployment, error)
	AppendDepTree(ctx context.Context, tree map[string]lib_model.Deployment, assets, containers bool) error
	CreateDepAssets(ctx context.Context, tx driver.Tx, dID string, depAssets lib_model.DepAssets) error
	DeleteDepAssets(ctx context.Context, tx driver.Tx, dID string) error
	CreateDepDependencies(ctx context.Context, tx driver.Tx, dID string, dIDs []string) error
	DeleteDepDependencies(ctx context.Context, tx driver.Tx, dID string) error
	CreateDepContainers(ctx context.Context, tx driver.Tx, dID string, depContainers map[string]lib_model.DepContainer) error
	DeleteDepContainers(ctx context.Context, tx driver.Tx, dID string) error
}

type CfgValidationHandler interface {
	ValidateBase(cType string, cTypeOpt module.ConfigTypeOptions, dataType module.DataType) error
	ValidateTypeOptions(cType string, cTypeOpt module.ConfigTypeOptions) error
	ValidateValue(cType string, cTypeOpt module.ConfigTypeOptions, value any, isSlice bool, dataType module.DataType) error
	ValidateValInOpt(cOpt any, value any, isSlice bool, dataType module.DataType) error
}
