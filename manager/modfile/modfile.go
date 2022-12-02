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

package modfile

import (
	"errors"
	"gopkg.in/yaml.v3"
	"module-manager/manager/itf"
)

const FileName = "Modfile"

var FileExtensions = []string{"yaml", "yml"}

type Module interface {
	Parse(confDefHandler itf.ConfDefHandler) (itf.Module, error)
}

type modFileBase struct {
	Version string `yaml:"modfileVersion"`
}

type ModFile struct {
	modFileBase `yaml:",inline"`
	module      Module
}

func (mf *ModFile) UnmarshalYAML(yn *yaml.Node) error {
	var base modFileBase
	if err := yn.Decode(&base); err != nil {
		return err
	}
	d, ok := decoder[base.Version]
	if !ok {
		return errors.New("unsupported modfile version")
	}
	m, err := d(yn)
	if err != nil {
		return err
	}
	*mf = ModFile{
		modFileBase: base,
		module:      m,
	}
	return nil
}

func (mf *ModFile) ParseModule(confDefHandler itf.ConfDefHandler) (itf.Module, error) {
	return mf.module.Parse(confDefHandler)
}
