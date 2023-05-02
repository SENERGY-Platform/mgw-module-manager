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

package modfile_hdl

import (
	"errors"
	"github.com/SENERGY-Platform/mgw-modfile-lib/modfile"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"gopkg.in/yaml.v3"
	"io/fs"
	"strings"
)

const fileName = "Modfile"

type Handler struct {
	mfDecoders   modfile.Decoders
	mfGenerators modfile.Generators
}

func New(mfDecoders modfile.Decoders, mfGenerators modfile.Generators) *Handler {
	return &Handler{
		mfDecoders:   mfDecoders,
		mfGenerators: mfGenerators,
	}
}

func (h *Handler) GetModule(dir util.DirFS) (*module.Module, error) {
	f, err := getModFile(dir)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	yd := yaml.NewDecoder(f)
	mf := modfile.New(h.mfDecoders, h.mfGenerators)
	err = yd.Decode(&mf)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	m, err := mf.GetModule()
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	return m, nil
}

func getModFile(dir util.DirFS) (fs.File, error) {
	dirEntries, err := fs.ReadDir(dir, ".")
	if err != nil {
		return nil, err
	}
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			if strings.Contains(entry.Name(), fileName) {
				f, err := dir.Open(entry.Name())
				if err != nil {
					return nil, err
				}
				return f, nil
			}
		}
	}
	return nil, errors.New("missing modfile")
}
