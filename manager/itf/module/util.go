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
	"reflect"
	"strings"
)

func InSemVerRange(r string, v string) (bool, error) {
	opr, ver, err := semVerRangeParse(r)
	if err != nil {
		return false, err
	}
	ok := semVerRangeCheck(opr[0], ver[0], v)
	if ok && len(opr) > 1 && len(ver) > 1 {
		ok = semVerRangeCheck(opr[1], ver[1], v)
	}
	return ok, nil
}

func semVerRangeCheck(o string, w, v string) bool {
	res := semver.Compare(v, w)
	switch res {
	case -1:
		if o == LessEqual || o == Less {
			return true
		}
	case 0:
		if o == Equal || o == LessEqual || o == GreaterEqual {
			return true
		}
	case 1:
		if o == GreaterEqual || o == Greater {
			return true
		}
	}
	return false
}

func semVerRangeParsePart(s string) (string, string, error) {
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

func semVerRangeParse(s string) (opr []string, ver []string, err error) {
	partsStr := strings.Split(s, ";")
	if len(partsStr) > 2 {
		err = fmt.Errorf("invalid format '%s'", s)
		return
	}
	for _, p := range partsStr {
		o, v, e := semVerRangeParsePart(p)
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

func (cv *configValue) Kind() reflect.Kind {
	return cv.t
}

func (cv *configValue) Is(t reflect.Kind) bool {
	return cv.t == t
}

func (c Configs) set(r string, d any, o any, t reflect.Kind) {
	c[r] = configValue{
		Default: d,
		Options: o,
		t:       t,
	}
}

func (c Configs) SetString(r string, d *string, o ...string) {
	if d != nil {
		c.set(r, *d, o, reflect.String)
		return
	}
	c.set(r, d, o, reflect.String)
}

func (c Configs) SetBool(r string, d *bool, o ...bool) {
	if d != nil {
		c.set(r, *d, o, reflect.Bool)
		return
	}
	c.set(r, d, o, reflect.Bool)
}

func (c Configs) SetInt64(r string, d *int64, o ...int64) {
	if d != nil {
		c.set(r, *d, o, reflect.Int64)
		return
	}
	c.set(r, d, o, reflect.Int64)
}

func (c Configs) SetFloat64(r string, d *float64, o ...float64) {
	if d != nil {
		c.set(r, *d, o, reflect.Float64)
		return
	}
	c.set(r, d, o, reflect.Float64)
}
