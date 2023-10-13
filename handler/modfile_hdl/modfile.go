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
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"gopkg.in/yaml.v3"
	"io/fs"
	"regexp"
)

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

func (h *Handler) GetModule(file fs.File) (*module.Module, error) {
	defer file.Close()
	yd := yaml.NewDecoder(file)
	mf := modfile.New(h.mfDecoders, h.mfGenerators)
	err := yd.Decode(&mf)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	m, err := mf.GetModule()
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	return m, nil
}

func (h *Handler) GetModFile(dir dir_fs.DirFS) (fs.File, string, error) {
	dirEntries, err := fs.ReadDir(dir, ".")
	if err != nil {
		return nil, "", err
	}
	re := regexp.MustCompile(`^Modfile\.(?:yml|yaml)$`)
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			eName := entry.Name()
			if re.MatchString(eName) {
				f, err := dir.Open(eName)
				if err != nil {
					return nil, "", err
				}
				return f, eName, nil
			}
		}
	}
	return nil, "", errors.New("missing modfile")
}
