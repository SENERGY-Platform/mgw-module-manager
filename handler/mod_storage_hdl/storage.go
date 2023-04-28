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

package mod_storage_hdl

import (
	"context"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-modfile-lib/modfile"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
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

type Handler struct {
	wrkSpacePath string
	delimiter    string
	mfDecoders   modfile.Decoders
	mfGenerators modfile.Generators
	perm         fs.FileMode
}

func New(workspacePath string, delimiter string, mfDecoders modfile.Decoders, mfGenerators modfile.Generators, perm fs.FileMode) (*Handler, error) {
	if !path.IsAbs(workspacePath) {
		return nil, fmt.Errorf("workspace path must be absolute")
	}
	return &Handler{
		wrkSpacePath: workspacePath,
		delimiter:    delimiter,
		mfDecoders:   mfDecoders,
		mfGenerators: mfGenerators,
		perm:         perm,
	}, nil
}

func (h *Handler) InitWorkspace() error {
	if err := os.MkdirAll(path.Join(h.wrkSpacePath, modDir), h.perm); err != nil {
		return err
	}
	if err := os.MkdirAll(path.Join(h.wrkSpacePath, inclDir), h.perm); err != nil {
		return err
	}
	return nil
}

func (h *Handler) List(ctx context.Context, filter model.ModFilter) ([]model.ModuleMeta, error) {
	dir, err := os.ReadDir(path.Join(h.wrkSpacePath, modDir))
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	var mm []model.ModuleMeta
	for _, entry := range dir {
		if entry.IsDir() {
			m, err := h.readModFile(path.Join(h.wrkSpacePath, modDir, entry.Name()))
			if err != nil {
				continue
			}
			if filterMod(filter, m) {
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
		}
		if ctx.Err() != nil {
			return nil, model.NewInternalError(err)
		}
	}
	return mm, nil
}

func (h *Handler) Get(_ context.Context, mID string) (*module.Module, error) {
	p := path.Join(h.wrkSpacePath, modDir, idToDir(mID, h.delimiter))
	if _, err := os.Stat(p); err != nil {
		return nil, model.NewNotFoundError(err)
	}
	m, err := h.readModFile(p)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	return m, nil
}

func (h *Handler) Add(ctx context.Context, dir util.DirFS) error {
	panic("not implemented")
}

func (h *Handler) Delete(_ context.Context, mID string) error {
	if err := os.RemoveAll(path.Join(h.wrkSpacePath, modDir, idToDir(mID, h.delimiter))); err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) MakeInclDir(_ context.Context, mID, iID string) (util.DirFS, error) {
	p := path.Join(h.wrkSpacePath, inclDir, iID)
	if err := copyDir(path.Join(h.wrkSpacePath, modDir, idToDir(mID, h.delimiter)), p); err != nil {
		_ = os.RemoveAll(p)
		return "", model.NewInternalError(err)
	}
	dir, err := util.NewDirFS(p)
	if err != nil {
		return "", model.NewInternalError(err)
	}
	return dir, nil
}

func (h *Handler) GetInclDir(_ context.Context, iID string) (util.DirFS, error) {
	dir, err := util.NewDirFS(path.Join(h.wrkSpacePath, inclDir, iID))
	if err != nil {
		return "", model.NewInternalError(err)
	}
	return dir, nil
}

func (h *Handler) RemoveInclDir(_ context.Context, iID string) error {
	if err := os.RemoveAll(path.Join(h.wrkSpacePath, inclDir, iID)); err != nil {
		return model.NewInternalError(err)
	}
	return nil
}

func (h *Handler) readModFile(p string) (*module.Module, error) {
	mfp, err := detectModFile(p)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(mfp)
	if err != nil {
		return nil, err
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

func filterMod(filter model.ModFilter, m *module.Module) bool {
	if filter.Name != "" {
		if m.Name != filter.Name {
			return false
		}
	}
	if filter.Type != "" {
		if m.Type != filter.Type {
			return false
		}
	}
	if filter.DeploymentType != "" {
		if m.DeploymentType != filter.DeploymentType {
			return false
		}
	}
	if filter.Author != "" {
		if m.Author != filter.Author {
			return false
		}
	}
	if len(filter.Tags) > 0 {
		var ok bool
		for tag := range filter.Tags {
			if _, ok = m.Tags[tag]; ok {
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(filter.InDependencies) > 0 {
		var ok bool
		for id := range filter.InDependencies {
			if _, ok = m.Dependencies[id]; ok {
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}
