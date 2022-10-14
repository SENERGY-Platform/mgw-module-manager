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
	"os/exec"
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

func (h *FileHandler) Create(srcPath string) error {
	cmd := exec.Command("find", srcPath, "-not", "-type", "d", "-not", "-type", "f")
	out, err := cmd.Output()
	if err != nil {
		return srv_base_types.NewError(http.StatusInternalServerError, "creating module failed: find returned", removeStrFromErr(err, srcPath))
	}
	if len(out) > 0 {
		return srv_base_types.NewError(http.StatusBadRequest, "creating module failed", fmt.Errorf("includes files with illigal types: %s", strings.TrimSuffix(strings.Replace(strings.Replace(string(out), srcPath, "", -1), "\n", ", ", -1), ", ")))
	}
	m, err := read(path.Join(srcPath, itf.ModFile))
	if err != nil {
		return srv_base_types.NewError(http.StatusBadRequest, "creating module failed", removeStrFromErr(err, srcPath))
	}
	dstPath := path.Join(h.WorkdirPath, idToDir(string(m.ID), h.Delimiter))
	if _, err := os.Stat(dstPath); err != nil {
		if !os.IsNotExist(err) {
			return srv_base_types.NewError(http.StatusInternalServerError, "creating module failed", removeStrFromErr(err, h.WorkdirPath))
		}
	} else {
		return srv_base_types.NewError(http.StatusBadRequest, "creating module failed", fmt.Errorf("'%s' already exists", m.ID))
	}
	cmd = exec.Command("cp", "-R", "--no-dereference", "--preserve=mode,timestamps", "--no-preserve=context,links,xattr", srcPath, dstPath)
	err = cmd.Run()
	if err != nil {
		if e := os.RemoveAll(dstPath); e != nil {
			srv_base.Logger.Errorf("cleanup failed: ", e)
		}
		return srv_base_types.NewError(http.StatusInternalServerError, "creating module failed: cp returned", err)
	}
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

func (h *FileHandler) Copy(id string, dstPath string) error {
	cmd := exec.Command("cp", "-R", "--no-dereference", "--preserve=mode,timestamps", "--no-preserve=context,links,xattr", path.Join(h.WorkdirPath, idToDir(id, h.Delimiter)), dstPath)
	return cmd.Run()
}

func read(mPath string) (itf.Module, error) {
	var module itf.Module
	mf, err := os.Open(mPath)
	if err != nil {
		return module, err
	}
	defer mf.Close()
	jd := json.NewDecoder(mf)
	if err := jd.Decode(&module); err != nil {
		return module, err
	}
	return module, nil
}

func removeStrFromErr(err error, str string) error {
	return errors.New(strings.Replace(err.Error(), str, "", -1))
}

func idToDir(id string, delimiter string) string {
	return strings.Replace(id, "/", delimiter, -1)
}

func dirToId(dir string, delimiter string) string {
	return strings.Replace(dir, delimiter, "/", -1)
}
