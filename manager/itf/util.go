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

package itf

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (m *ModuleType) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	if t, ok := ModuleTypeMap[s]; ok {
		*m = t
	} else {
		err = fmt.Errorf("unknown module type '%s'", s)
	}
	return
}

func (d *DeploymentType) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	if t, ok := DeploymentTypeMap[s]; ok {
		*d = t
	} else {
		err = fmt.Errorf("unknown deployment type '%s'", s)
	}
	return
}

func (r *ResourceType) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	if t, ok := MountResourceTypeMap[s]; ok {
		*r = t
	} else if t, ok = LinkResourceTypeMap[s]; ok {
		*r = t
	} else {
		err = fmt.Errorf("unknown resurce type '%s'", s)
	}
	return
}

func (r *ResourceType) IsMount() bool {
	_, ok := MountResourceTypeMap[string(*r)]
	return ok
}

func (r *ResourceType) IsLink() bool {
	_, ok := LinkResourceTypeMap[string(*r)]
	return ok
}

func (i *ModuleID) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	if !strings.Contains(s, "/") || strings.Contains(s, "//") || strings.HasPrefix(s, "/") {
		err = fmt.Errorf("invalid module ID format '%s'", s)
	} else {
		*i = ModuleID(s)
	}
	return
}
