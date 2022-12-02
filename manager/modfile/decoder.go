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
	"gopkg.in/yaml.v3"
	"module-manager/manager/modfile/v1"
)

func Decode[T Module](yn *yaml.Node) (Module, error) {
	var m T
	if err := yn.Decode(&m); err != nil {
		return m, err
	}
	return m, nil
}

var decoder = map[string]func(*yaml.Node) (Module, error){
	"1": Decode[v1.Module],
}
