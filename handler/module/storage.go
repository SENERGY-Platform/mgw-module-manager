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
	"github.com/SENERGY-Platform/mgw-modfile-lib/modfile"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	fileName = "Modfile"
	modDir   = "modules"
	inclDir  = "deployments"
)

var mfExtensions = []string{"yaml", "yml"}

type StorageHandler struct {
	wrkSpacePath string
	delimiter    string
	mfDecoders   modfile.Decoders
	mfGenerators modfile.Generators
	perm         fs.FileMode
}

func NewStorageHandler(workspacePath string, delimiter string, mfDecoders modfile.Decoders, mfGenerators modfile.Generators, perm fs.FileMode) (*StorageHandler, error) {
	if !path.IsAbs(workspacePath) {
		return nil, fmt.Errorf("workspace path must be absolute")
	}
	return &StorageHandler{
		wrkSpacePath: workspacePath,
		delimiter:    delimiter,
		mfDecoders:   mfDecoders,
		mfGenerators: mfGenerators,
		perm:         perm,
	}, nil
}

func (h *StorageHandler) InitWorkspace() error {
	if err := os.MkdirAll(path.Join(h.wrkSpacePath, modDir), h.perm); err != nil {
		return err
	}
	if err := os.MkdirAll(path.Join(h.wrkSpacePath, inclDir), h.perm); err != nil {
		return err
	}
	return nil
}

func (h *StorageHandler) List(ctx context.Context) ([]model.ModuleMeta, error) {
	dir, err := os.ReadDir(path.Join(h.wrkSpacePath, modDir))
	if err != nil {
		return nil, model.NewInternalError(wrapErr(err, h.wrkSpacePath))
	}
	var mm []model.ModuleMeta
	for _, entry := range dir {
		if entry.IsDir() {
			m, err := h.readModFile(path.Join(h.wrkSpacePath, modDir, entry.Name()))
			if err != nil {
				continue
			}
			mm = append(mm, model.ModuleMeta{
				ID:             m.ID,
				Name:           m.Name,
				Description:    m.Description,
				Tags:           m.Tags,
				License:        m.License,
				Author:         m.Author,
				Version:        m.Version,
				Type:           m.Type,
				DeploymentType: m.DeploymentType,
			})
		}
		if ctx.Err() != nil {
			return nil, model.NewInternalError(err)
		}
	}
	return mm, nil
}

func (h *StorageHandler) Get(_ context.Context, mID string) (*module.Module, error) {
	p := path.Join(h.wrkSpacePath, modDir, idToDir(mID, h.delimiter))
	if _, err := os.Stat(p); err != nil {
		return nil, model.NewNotFoundError(wrapErr(err, h.wrkSpacePath))
	}
	m, err := h.readModFile(p)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	return m, nil
}

func (h *StorageHandler) Add(ctx context.Context, mID string) error {
	panic("not implemented")
}

func (h *StorageHandler) Delete(_ context.Context, mID string) error {
	if err := os.RemoveAll(path.Join(h.wrkSpacePath, modDir, idToDir(mID, h.delimiter))); err != nil {
		return model.NewInternalError(wrapErr(err, h.wrkSpacePath))
	}
	return nil
}

func (h *StorageHandler) CreateInclDir(_ context.Context, mID, iID string) (string, error) {
	p := path.Join(h.wrkSpacePath, inclDir, iID)
	if err := copyDir(path.Join(h.wrkSpacePath, modDir, mID), p); err != nil {
		_ = os.RemoveAll(p)
		return "", model.NewInternalError(wrapErr(err, h.wrkSpacePath))
	}
	return p, nil
}

func (h *StorageHandler) DeleteInclDir(_ context.Context, iID string) error {
	if err := os.RemoveAll(path.Join(h.wrkSpacePath, inclDir, iID)); err != nil {
		return model.NewInternalError(wrapErr(err, h.wrkSpacePath))
	}
	return nil
}

func (h *StorageHandler) readModFile(p string) (*module.Module, error) {
	mfp, err := detectModFile(p)
	if err != nil {
		return nil, wrapErr(err, h.wrkSpacePath)
	}
	f, err := os.Open(mfp)
	if err != nil {
		return nil, wrapErr(err, h.wrkSpacePath)
	}
	yd := yaml.NewDecoder(f)
	mf := modfile.New(h.mfDecoders, h.mfGenerators)
	err = yd.Decode(&mf)
	if err != nil {
		return nil, err
	}
	m, err := mf.GetModule()
	if err != nil {
		return nil, err
	}
	return m, nil
}

func copyDir(src string, dst string) error {
	cmd := exec.Command("cp", "-R", "--no-dereference", "--preserve=mode,timestamps", "--no-preserve=context,links,xattr", src, dst)
	return cmd.Run()
}

func idToDir(id string, delimiter string) string {
	return strings.Replace(id, "/", delimiter, -1)
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
	p = path.Join(p, fileName)
	if ok, err := checkIfExist(p); err != nil || ok {
		return p, err
	}
	for _, ext := range mfExtensions {
		tp := p + "." + ext
		if ok, err := checkIfExist(tp); err != nil || ok {
			return tp, err
		}
	}
	return "", errors.New("modfile not found")
}

type FileHandlerError struct {
	msg string
	err error
}

func wrapErr(err error, s string) error {
	return &FileHandlerError{
		msg: strings.Replace(err.Error(), s, "", -1),
		err: err,
	}
}

func (e *FileHandlerError) Error() string {
	return e.msg
}

func (e *FileHandlerError) Unwrap() error {
	return e.err
}
