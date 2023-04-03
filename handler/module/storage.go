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

package module

import (
	"context"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/go-service-base/srv-base"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
)

const FileName = "Modfile"

var FileExtensions = []string{"yaml", "yml"}

type StorageHandler struct {
	WorkdirPath string
	Delimiter   string
}

func NewStorageHandler(workdirPath string, delimiter string) (*StorageHandler, error) {
	if !path.IsAbs(workdirPath) {
		return nil, fmt.Errorf("workdir path must be absolute")
	}
	return &StorageHandler{
		WorkdirPath: workdirPath,
		Delimiter:   delimiter,
	}, nil
}

func (h *StorageHandler) List(ctx context.Context) ([]string, error) {
	dir, err := os.ReadDir(h.WorkdirPath)
	if err != nil {
		return nil, newErr(h.WorkdirPath, err)
	}
	var mIds []string
	for _, entry := range dir {
		if entry.IsDir() {
			mIds = append(mIds, dirToId(entry.Name(), h.Delimiter))
		}
	}
	return mIds, nil
}

func (h *StorageHandler) Open(ctx context.Context, id string) (io.ReadCloser, error) {
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

func (h *StorageHandler) Delete(ctx context.Context, id string) error {
	if err := os.RemoveAll(path.Join(h.WorkdirPath, idToDir(id, h.Delimiter))); err != nil {
		return newErr(h.WorkdirPath, err)
	}
	return nil
}

func (h *StorageHandler) CopyTo(ctx context.Context, id string, dstPath string) error {
	return copyDir(path.Join(h.WorkdirPath, idToDir(id, h.Delimiter)), dstPath)
}

func (h *StorageHandler) CopyFrom(ctx context.Context, id string, srcPath string) error {
	dstPath := path.Join(h.WorkdirPath, idToDir(id, h.Delimiter))
	if ok, err := checkIfExist(dstPath); err != nil {
		return newErr(h.WorkdirPath, err)
	} else if ok {
		return errors.New("already exists")
	}
	if err := copyDir(srcPath, dstPath); err != nil {
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
	p = path.Join(p, FileName)
	if ok, err := checkIfExist(p); err != nil || ok {
		return p, err
	}
	for _, ext := range FileExtensions {
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
