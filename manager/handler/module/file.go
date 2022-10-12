/*
 * Copyright 2022 InfAI (CC SES)
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

package module

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/go-service-base/srv-base"
	"github.com/SENERGY-Platform/go-service-base/srv-base/types"
	"module-manager/manager/itf"
	"net/http"
	"os"
	"path"
	"strings"
)

type FileHandler struct {
	WorkdirPath string
	Delimiter   string
}

func NewFileHandler(workdirPath string, delimiter string) (itf.ModuleStorageHandler, error) {
	fh := &FileHandler{}
	if !path.IsAbs(workdirPath) {
		return fh, fmt.Errorf("workdir path must be absolute")
	}
	fh.WorkdirPath = workdirPath
	fh.Delimiter = delimiter
	return fh, nil
}

func (h *FileHandler) List() ([]itf.Module, error) {
	var modules []itf.Module
	de, err := os.ReadDir(h.WorkdirPath)
	if err != nil {
		return modules, srv_base_types.NewError(http.StatusInternalServerError, "listing modules failed", removeStrFromErr(err, h.WorkdirPath))
	}
	for _, entry := range de {
		if entry.IsDir() {
			module, err := read(path.Join(h.WorkdirPath, entry.Name(), itf.ModFile))
			if err != nil {
				srv_base.Logger.Errorf("reading module '%s' failed: %s", dirToId(entry.Name(), h.Delimiter), err)
			}
			modules = append(modules, module)
		}
	}
	return modules, nil
}

func (h *FileHandler) Create() error {
	return nil
}

func (h *FileHandler) Read(id string) (itf.Module, error) {
	m, err := read(path.Join(h.WorkdirPath, idToDir(id, h.Delimiter), itf.ModFile))
	if err != nil {
		code := http.StatusInternalServerError
		if os.IsNotExist(err) {
			code = http.StatusNotFound
		}
		return m, srv_base_types.NewError(code, fmt.Sprintf("reading module '%s' failed", id), removeStrFromErr(err, h.WorkdirPath))
	}
	return m, nil
}

func (h *FileHandler) Update() error {
	return nil
}

func (h *FileHandler) Delete(id string) error {
	err := os.RemoveAll(path.Join(h.WorkdirPath, idToDir(id, h.Delimiter)))
	if err != nil {
		code := http.StatusInternalServerError
		if os.IsNotExist(err) {
			code = http.StatusNotFound
		}
		return srv_base_types.NewError(code, "deleting module failed", removeStrFromErr(err, h.WorkdirPath))
	}
	return nil
}

func read(mPath string) (itf.Module, error) {
	var module itf.Module
	mf, err := os.Open(mPath)
	if err != nil {
		return module, err
	}
	jd := json.NewDecoder(mf)
	if err := jd.Decode(&module); err != nil {
		return module, err
	}
	return module, nil
}

func removeStrFromErr(err error, str string) error {
	s := strings.Replace(err.Error(), str, "", -1)
	return errors.New(s)
}

func idToDir(id string, delimiter string) string {
	return strings.Replace(id, "/", delimiter, -1)
}

func dirToId(dir string, delimiter string) string {
	return strings.Replace(dir, delimiter, "/", -1)
}
