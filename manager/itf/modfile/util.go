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
	"module-manager/manager/itf/modfile/v1"
	"module-manager/manager/itf/module"
)

var decoders = map[string]func(*yaml.Node) (itf.ModFileModule, error){
	"1": v1.Decode,
}

func (mf *ModFile) UnmarshalYAML(yn *yaml.Node) error {
	var base modFileBase
	if err := yn.Decode(&base); err != nil {
		return err
	}
	d, ok := decoders[base.Version]
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

func (mf *ModFile) ParseModule() (module.Module, error) {
	return mf.module.Parse()
}
