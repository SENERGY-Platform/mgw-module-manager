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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/mod/semver"
	"reflect"
	"sort"
	"strconv"
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

func (c Configs) SetString(r string, d *string, o ...string) {
	var sl []any
	if o != nil {
		for _, s := range o {
			sl = append(sl, s)
		}
	}
	cv := configValue{Options: sl, t: reflect.String}
	if d != nil {
		cv.Default = *d
	} else {
		cv.Default = d
	}
	c[r] = cv
}

func (c Configs) SetBool(r string, d *bool, o ...bool) {
	var sl []any
	if o != nil {
		for _, b := range o {
			sl = append(sl, b)
		}
	}
	cv := configValue{Options: sl, t: reflect.Bool}
	if d != nil {
		cv.Default = *d
	} else {
		cv.Default = d
	}
	c[r] = cv
}

func (c Configs) SetInt64(r string, d *int64, o ...int64) {
	var sl []any
	if o != nil {
		for _, i := range o {
			sl = append(sl, i)
		}
	}
	cv := configValue{Options: sl, t: reflect.Int64}
	if d != nil {
		cv.Default = *d
	} else {
		cv.Default = d
	}
	c[r] = cv
}

func (c Configs) SetFloat64(r string, d *float64, o ...float64) {
	var sl []any
	if o != nil {
		for _, f := range o {
			sl = append(sl, f)
		}
	}
	cv := configValue{Options: sl, t: reflect.Float64}
	if d != nil {
		cv.Default = *d
	} else {
		cv.Default = d
	}
	c[r] = cv
}

func (s *Set[T]) UnmarshalJSON(b []byte) error {
	var sl []T
	if err := json.Unmarshal(b, &sl); err != nil {
		return err
	}
	set := make(Set[T])
	for _, item := range sl {
		set[item] = struct{}{}
	}
	*s = set
	return nil
}

func (s Set[T]) MarshalJSON() ([]byte, error) {
	var sl []T
	for item := range s {
		sl = append(sl, item)
	}
	return json.Marshal(sl)
}

func (s Set[T]) Slice() []T {
	var sl []T
	for item := range s {
		sl = append(sl, item)
	}
	return sl
}

func (p PortMappings) Add(name *string, port []uint, hostPort []uint, protocol *string) error {
	var s []string
	if port == nil || !IsValidPort(port) {
		return fmt.Errorf("invalid port '%v'", port)
	}
	for _, n := range port {
		s = append(s, strconv.FormatInt(int64(n), 10))
	}
	if hostPort != nil {
		if !IsValidPort(hostPort) {
			return fmt.Errorf("invalid host port '%v'", hostPort)
		}
		var lp int
		var lhp int
		if len(port) > 1 {
			lp = int(port[1]-port[0]) + 1
		} else {
			lp = 1
		}
		if len(hostPort) > 1 {
			lhp = int(hostPort[1]-hostPort[0]) + 1
		} else {
			lhp = 1
		}
		if lp != lhp {
			if lp > lhp {
				return errors.New("range mismatch: ports > host ports")
			}
			if lp > 1 && lp < lhp {
				return errors.New("range mismatch: ports < host ports")
			}
		}
		for _, n := range hostPort {
			s = append(s, strconv.FormatInt(int64(n), 10))
		}
	}
	if protocol != nil {
		if !IsValidPortType(*protocol) {
			return fmt.Errorf("invalid protocol '%s'", *protocol)
		}
		s = append(s, *protocol)
	}
	key, err := hashStrings(s)
	if err != nil {
		return err
	}
	p[key] = portMapping{
		Name:     name,
		Port:     port,
		HostPort: hostPort,
		Protocol: protocol,
	}
	return nil
}

func (p PortMappings) MarshalJSON() ([]byte, error) {
	var sl []portMapping
	for _, pm := range p {
		sl = append(sl, pm)
	}
	return json.Marshal(sl)
}

func hashStrings(str []string) (string, error) {
	if str == nil || len(str) == 0 {
		return "", fmt.Errorf("failed to hash strings: no entries to write")
	}
	sort.Strings(str)
	h := sha256.New()
	for i := 0; i < len(str); i++ {
		_, err := h.Write([]byte(str[i]))
		if err != nil {
			return "", fmt.Errorf("failed to hash strings: %s", err)
		}
	}
	return base64.URLEncoding.EncodeToString(h.Sum(nil)), nil
}
