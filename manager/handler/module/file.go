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
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/go-service-base/srv-base"
	"io"
	"module-manager/manager/itf"
	"os"
	"os/exec"
	"path"
	"strings"
)

const FileName = "Modfile"

var FileExtensions = []string{"yaml", "yml"}

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

func (h *FileHandler) List() ([]string, error) {
	var mIds []string
	de, err := os.ReadDir(h.WorkdirPath)
	if err != nil {
		return mIds, newErr(h.WorkdirPath, err)
	}
	for _, entry := range de {
		if entry.IsDir() {
			mIds = append(mIds, dirToId(entry.Name(), h.Delimiter))
		}
	}
	return mIds, nil
}

func (h *FileHandler) Open(id string) (io.ReadCloser, error) {
	p := path.Join(h.WorkdirPath, idToDir(id, h.Delimiter))
	if _, err := os.Stat(p); err != nil {
		return nil, newErr(h.WorkdirPath, err)
	}
	p, err := detectModFile(p)
	if err != nil {
		return nil, newErr(h.WorkdirPath, err)
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, newErr(h.WorkdirPath, err)
	}
	return f, nil
}

func (h *FileHandler) Delete(id string) error {
	if err := os.RemoveAll(path.Join(h.WorkdirPath, idToDir(id, h.Delimiter))); err != nil {
		return newErr(h.WorkdirPath, err)
	}
	return nil
}

func (h *FileHandler) CopyTo(id string, dstPath string) error {
	return copyDir(path.Join(h.WorkdirPath, idToDir(id, h.Delimiter)), dstPath)
}

func (h *FileHandler) CopyFrom(id string, srcPath string) error {
	dstPath := path.Join(h.WorkdirPath, idToDir(id, h.Delimiter))
	if ok, err := checkIfExist(dstPath); err != nil {
		return newErr(h.WorkdirPath, err)
	} else if ok {
		return errors.New("already exists")
	}
	err := copyDir(srcPath, dstPath)
	if err != nil {
		if e := os.RemoveAll(dstPath); e != nil {
			srv_base.Logger.Errorf("cleanup failed: %s", e)
		}
		return fmt.Errorf("copy returned: %s", err)
	}
	return nil
}

func copyDir(src string, dst string) error {
	cmd := exec.Command("cp", "-R", "--no-dereference", "--preserve=mode,timestamps", "--no-preserve=context,links,xattr", src, dst)
	return cmd.Run()
}

func idToDir(id string, delimiter string) string {
	return strings.Replace(id, "/", delimiter, -1)
}

func dirToId(dir string, delimiter string) string {
	return strings.Replace(dir, delimiter, "/", -1)
}

func checkIfExist(p string) (ok bool, err error) {
	_, err = os.Stat(p)
	if err == nil {
		ok = true
	} else if err != nil && os.IsNotExist(err) {
		err = nil
	}
	return
}

func detectModFile(p string) (string, error) {
	p = path.Join(p, modfile.FileName)
	if ok, err := checkIfExist(p); err != nil || ok {
		return p, err
	}
	for _, ext := range modfile.FileExtensions {
		tp := p + "." + ext
		if ok, err := checkIfExist(tp); err != nil || ok {
			return tp, err
		}
	}
	return "", errors.New("modfile not found")
}

type FileHandlerErr struct {
	str string
	err error
}

func newErr(str string, err error) error {
	return &FileHandlerErr{
		str: str,
		err: err,
	}
}

func (e *FileHandlerErr) Error() string {
	return strings.Replace(e.err.Error(), e.str, "", -1)
}

func (e *FileHandlerErr) Unwrap() error {
	return e.err
}
