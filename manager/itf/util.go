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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"module-manager/manager/itf/misc"
	"sort"
	"strconv"
)

func newConfigValue[T any](def *T, opt []T, dType misc.DataType, optExt bool, cType string, cTypeOpt ConfigTypeOptions) configValue {
	cv := configValue{
		OptExt:   optExt,
		Type:     cType,
		DataType: dType,
	}
	if def != nil {
		cv.Default = *def
	}
	if opt != nil && len(opt) > 0 {
		cv.Options = opt
	}
	if cTypeOpt != nil && len(cTypeOpt) > 0 {
		cv.TypeOpt = cTypeOpt
	}
	return cv
}

func newConfigValueSlice[T any](def []T, opt []T, dType misc.DataType, optExt bool, cType string, cTypeOpt ConfigTypeOptions, delimiter *string) configValue {
	cv := configValue{
		OptExt:    optExt,
		Type:      cType,
		DataType:  dType,
		IsSlice:   true,
		Delimiter: delimiter,
	}
	if def != nil && len(def) > 0 {
		cv.Default = def
	}
	if opt != nil && len(opt) > 0 {
		cv.Options = opt
	}
	if cTypeOpt != nil && len(cTypeOpt) > 0 {
		cv.TypeOpt = cTypeOpt
	}
	return cv
}

func (c Configs) SetString(ref string, def *string, opt []string, optExt bool, cType string, cTypeOpt ConfigTypeOptions) {
	c[ref] = newConfigValue(def, opt, misc.String, optExt, cType, cTypeOpt)
}

func (c Configs) SetBool(ref string, def *bool, opt []bool, optExt bool, cType string, cTypeOpt ConfigTypeOptions) {
	c[ref] = newConfigValue(def, opt, misc.Bool, optExt, cType, cTypeOpt)
}

func (c Configs) SetInt64(ref string, def *int64, opt []int64, optExt bool, cType string, cTypeOpt ConfigTypeOptions) {
	c[ref] = newConfigValue(def, opt, misc.Int64, optExt, cType, cTypeOpt)
}

func (c Configs) SetFloat64(ref string, def *float64, opt []float64, optExt bool, cType string, cTypeOpt ConfigTypeOptions) {
	c[ref] = newConfigValue(def, opt, misc.Float64, optExt, cType, cTypeOpt)
}

func (c Configs) SetStringSlice(ref string, def []string, opt []string, optExt bool, cType string, cTypeOpt ConfigTypeOptions, delimiter *string) {
	c[ref] = newConfigValueSlice(def, opt, misc.String, optExt, cType, cTypeOpt, delimiter)
}

func (c Configs) SetBoolSlice(ref string, def []bool, opt []bool, optExt bool, cType string, cTypeOpt ConfigTypeOptions, delimiter *string) {
	c[ref] = newConfigValueSlice(def, opt, misc.Bool, optExt, cType, cTypeOpt, delimiter)
}

func (c Configs) SetInt64Slice(ref string, def []int64, opt []int64, optExt bool, cType string, cTypeOpt ConfigTypeOptions, delimiter *string) {
	c[ref] = newConfigValueSlice(def, opt, misc.Int64, optExt, cType, cTypeOpt, delimiter)
}

func (c Configs) SetFloat64Slice(ref string, def []float64, opt []float64, optExt bool, cType string, cTypeOpt ConfigTypeOptions, delimiter *string) {
	c[ref] = newConfigValueSlice(def, opt, misc.Float64, optExt, cType, cTypeOpt, delimiter)
}

func (o ConfigTypeOptions) SetString(ref string, val string) {
	o[ref] = configTypeOption{
		Value:    val,
		DataType: misc.String,
	}
}

func (o ConfigTypeOptions) SetBool(ref string, val bool) {
	o[ref] = configTypeOption{
		Value:    val,
		DataType: misc.Bool,
	}
}

func (o ConfigTypeOptions) SetInt64(ref string, val int64) {
	o[ref] = configTypeOption{
		Value:    val,
		DataType: misc.Int64,
	}
}

func (o ConfigTypeOptions) SetFloat64(ref string, val float64) {
	o[ref] = configTypeOption{
		Value:    val,
		DataType: misc.Float64,
	}
}

func (v configValue) OptionsLen() (l int) {
	switch o := v.Options.(type) {
	case []string:
		l = len(o)
	case []bool:
		l = len(o)
	case []int64:
		l = len(o)
	case []float64:
		l = len(o)
	}
	return
}

func isValidPort(p []uint) bool {
	return !(p == nil || len(p) == 0 || len(p) > 2 || (len(p) > 1 && p[0] == p[1]) || (len(p) > 1 && p[1] < p[0]))
}

func isValidPortType(s string) bool {
	_, ok := PortTypeMap[s]
	return ok
}

func (p PortMappings) Add(name *string, port []uint, hostPort []uint, protocol *string) error {
	var s []string
	if port == nil || !isValidPort(port) {
		return fmt.Errorf("invalid port '%v'", port)
	}
	for _, n := range port {
		s = append(s, strconv.FormatInt(int64(n), 10))
	}
	if hostPort != nil {
		if !isValidPort(hostPort) {
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
		if !isValidPortType(*protocol) {
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
