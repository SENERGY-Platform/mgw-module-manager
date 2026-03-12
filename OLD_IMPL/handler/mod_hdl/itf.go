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

package mod_hdl

import (
	"context"
	"database/sql/driver"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"io/fs"
)

type StorageHandler interface {
	BeginTransaction(ctx context.Context) (driver.Tx, error)
	ListMod(ctx context.Context, filter model.ModFilter, dependencyInfo bool) (map[string]model.Module, error)
	ReadMod(ctx context.Context, mID string, dependencyInfo bool) (model.Module, error)
	CreateMod(ctx context.Context, tx driver.Tx, mod model.Module) error
	UpdateMod(ctx context.Context, tx driver.Tx, mod model.Module) error
	DeleteMod(ctx context.Context, tx driver.Tx, mID string) error
	CreateModDependencies(ctx context.Context, tx driver.Tx, mID string, mIDs []string) error
	DeleteModDependencies(ctx context.Context, tx driver.Tx, mID string) error
}

type ModFileHandler interface {
	GetModule(file fs.File) (*module_lib.Module, error)
	GetModFile(dir dir_fs.DirFS) (fs.File, string, error)
}
