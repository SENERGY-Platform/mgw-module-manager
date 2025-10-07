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

package mod_staging_hdl

import (
	"context"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"io/fs"
)

type ModTransferHandler interface {
	Get(ctx context.Context, mID string) (model.ModRepo, error)
}

type ModFileHandler interface {
	GetModule(file fs.File) (*module_lib.Module, error)
	GetModFile(dir dir_fs.DirFS) (fs.File, string, error)
}

type CfgValidationHandler interface {
	ValidateBase(cType string, cTypeOpt module_lib.ConfigTypeOptions, dataType module_lib.DataType) error
	ValidateTypeOptions(cType string, cTypeOpt module_lib.ConfigTypeOptions) error
	ValidateValue(cType string, cTypeOpt module_lib.ConfigTypeOptions, value any, isSlice bool, dataType module_lib.DataType) error
	ValidateValInOpt(cOpt any, value any, isSlice bool, dataType module_lib.DataType) error
}
