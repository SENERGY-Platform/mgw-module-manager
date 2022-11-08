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

package module

import (
	"fmt"
	"golang.org/x/mod/semver"
	"strings"
)

func IsValidModuleType(s string) bool {
	if _, ok := ModuleTypeMap[s]; ok {
		return true
	}
	return false
}

func IsValidDeploymentType(s string) bool {
	if _, ok := DeploymentTypeMap[s]; ok {
		return true
	}
	return false
}

func IsValidModuleID(s string) bool {
	if !strings.Contains(s, "/") || strings.Contains(s, "//") || strings.HasPrefix(s, "/") {
		return false
	}
	return true
}

func IsValidSrvDepCondition(s string) bool {
	if _, ok := SrvDepConditionMap[s]; ok {
		return true
	}
	return false
}

func IsValidDataType(s string) bool {
	if _, ok := DataTypeMap[s]; ok {
		return true
	}
	return false
}

func IsValidSemVer(s string) bool {
	if semver.IsValid(s) {
		return true
	}
	return false
}

const (
	Greater      = ">"
	Less         = "<"
	Equal        = "="
	GreaterEqual = ">="
	LessEqual    = "<="
)

var OperatorMap = map[string]struct{}{
	Greater:      {},
	Less:         {},
	Equal:        {},
	GreaterEqual: {},
	LessEqual:    {},
}

func IsValidOperator(s string) bool {
	if _, ok := OperatorMap[s]; ok {
		return true
	}
	return false
}

func IsValidSemVerRange(s string) (bool, error) {
	if _, _, err := parse(s); err != nil {
		return false, err
	}
	return true, nil
}

func InSemVerRange(r string, v string) (bool, error) {
	opr, ver, err := parse(r)
	if err != nil {
		return false, err
	}
	ok := check(opr[0], ver[0], v)
	if ok && len(opr) > 1 && len(ver) > 1 {
		ok = check(opr[1], ver[1], v)
	}
	return ok, nil
}

func check(o string, w, v string) bool {
	res := semver.Compare(v, w)
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

func parsePart(s string) (string, string, error) {
	pos := strings.Index(s, "v")
	if pos < 1 || pos > 2 {
		return "", "", fmt.Errorf("invalid format '%s'", s)
	}
	if IsValidOperator(s[:pos]) {
		if !semver.IsValid(s[pos:]) {
			return "", "", fmt.Errorf("invalid version format '%s'", s[pos:])
		}
		return s[:pos], s[pos:], nil
	}
	return "", "", fmt.Errorf("invalid operator format '%s'", s[:pos])
}

func parse(s string) (opr []string, ver []string, err error) {
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
		if semver.Compare(ver[0], ver[1]) > -1 {
			err = fmt.Errorf("invalid version order '%s' - '%s'", ver[0], ver[1])
			return
		}
	}
	return
}