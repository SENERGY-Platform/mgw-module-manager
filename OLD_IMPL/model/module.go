/*
 * Copyright 2023 InfAI (CC SES)
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

package model

import (
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"path"
)

type Module struct {
	lib_model.Module
	RequiredMod  []string // modules required by this module
	ModRequiring []string // modules requiring this module
	Path         string
	Dir          string
	ModFile      string
}

type ModFilter struct {
	IDs []string
}

func (m Module) GetDirFS() (dir_fs.DirFS, error) {
	dirFS, err := dir_fs.New(path.Join(m.Path, m.Dir))
	if err != nil {
		return "", lib_model.NewInternalError(err)
	}
	return dirFS, nil
}
