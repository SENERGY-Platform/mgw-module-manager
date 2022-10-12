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
	"golang.org/x/mod/semver"
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

func (v *SemVersion) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	if semver.IsValid(s) {
		*v = SemVersion(s)
	} else {
		err = fmt.Errorf("invalid version format '%s'", s)
	}
	return
}

func (v *SemVersion) String() string {
	return string(*v)
}

func (r *SemVersionRange) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	if _, _, err = parse(s); err != nil {
		return
	}
	*r = SemVersionRange(s)
	return
}

func (r *SemVersionRange) InRange(v SemVersion) (bool, error) {
	opr, ver, err := parse(string(*r))
	if err != nil {
		return false, err
	}
	ok := check(opr[0], ver[0], v)
	if ok && len(opr) > 1 && len(ver) > 1 {
		ok = check(opr[1], ver[1], v)
	}
	return ok, nil
}

func check(o VersionOperator, w, v SemVersion) bool {
	res := semver.Compare(v.String(), w.String())
	ok := false
	switch res {
	case -1:
		if o == LessEqual || o == Less {
			ok = true
		}
	case 0:
		if o == Equal || o == LessEqual || o == GreaterEqual {
			ok = true
		}
	case 1:
		if o == GreaterEqual || o == Greater {
			ok = true
		}
	}
	return ok
}

func parsePart(s string) (opr VersionOperator, ver SemVersion, err error) {
	pos := strings.Index(s, "v")
	if pos < 1 || pos > 2 {
		err = fmt.Errorf("invalid format '%s'", s)
		return
	}
	if o, ok := OperatorMap[s[:pos]]; ok {
		if !semver.IsValid(s[pos:]) {
			err = fmt.Errorf("invalid version format '%s'", s[pos:])
			return
		}
		opr = o
		ver = SemVersion(s[pos:])
		return
	}
	err = fmt.Errorf("invalid operator format '%s'", s[:pos])
	return
}

func parse(s string) (opr []VersionOperator, ver []SemVersion, err error) {
	partsStr := strings.Split(s, ";")
	if len(partsStr) > 2 {
		err = fmt.Errorf("invalid format '%s'", s)
		return
	}
	for _, p := range partsStr {
		o, v, e := parsePart(p)
		if e != nil {
			err = e
			return
		}
		opr = append(opr, o)
		ver = append(ver, v)
	}
	if len(opr) > 1 && len(ver) > 1 {
		if opr[0] == Less || opr[0] == LessEqual || opr[0] == Equal {
			err = fmt.Errorf("invalid operator order '%s' + '%s'", opr[0], opr[1])
			return
		}
		if opr[1] == Greater || opr[1] == GreaterEqual || opr[1] == Equal {
			err = fmt.Errorf("invalid operator order '%s' - '%s'", opr[0], opr[1])
			return
		}
		if semver.Compare(ver[0].String(), ver[1].String()) > -1 {
			err = fmt.Errorf("invalid version order '%s' - '%s'", ver[0], ver[1])
			return
		}
	}
	return
}
