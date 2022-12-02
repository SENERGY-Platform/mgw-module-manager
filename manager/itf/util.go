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

func (c Configs) set(ref string, def any, opt any, dType misc.DataType, optExt bool, cType string, cTypeOpt ConfigTypeOptions) {
	c[ref] = configValue{
		Default:  def,
		Options:  opt,
		OptExt:   optExt,
		Type:     cType,
		TypeOpt:  cTypeOpt,
		DataType: dType,
	}
}

func (c Configs) SetString(ref string, def *string, opt []string, optExt bool, cType string, cTypeOpt ConfigTypeOptions) {
	if def != nil {
		c.set(ref, *def, opt, misc.String, optExt, cType, cTypeOpt)
	} else {
		c.set(ref, def, opt, misc.String, optExt, cType, cTypeOpt)
	}
}

func (c Configs) SetBool(ref string, def *bool, opt []bool, optExt bool, cType string, cTypeOpt ConfigTypeOptions) {
	if def != nil {
		c.set(ref, *def, opt, misc.Bool, optExt, cType, cTypeOpt)
	} else {
		c.set(ref, def, opt, misc.Bool, optExt, cType, cTypeOpt)
	}
}

func (c Configs) SetInt64(ref string, def *int64, opt []int64, optExt bool, cType string, cTypeOpt ConfigTypeOptions) {
	if def != nil {
		c.set(ref, *def, opt, misc.Int64, optExt, cType, cTypeOpt)
	} else {
		c.set(ref, def, opt, misc.Int64, optExt, cType, cTypeOpt)
	}
}

func (c Configs) SetFloat64(ref string, def *float64, opt []float64, optExt bool, cType string, cTypeOpt ConfigTypeOptions) {
	if def != nil {
		c.set(ref, *def, opt, misc.Float64, optExt, cType, cTypeOpt)
	} else {
		c.set(ref, def, opt, misc.Float64, optExt, cType, cTypeOpt)
	}
}

func (c Configs) setSlice(ref string, def any, opt any, dType misc.DataType, optExt bool, cType string, cTypeOpt ConfigTypeOptions, delimiter *string) {
	c[ref] = configValue{
		Default:   def,
		Options:   opt,
		OptExt:    optExt,
		Type:      cType,
		TypeOpt:   cTypeOpt,
		DataType:  dType,
		IsSlice:   true,
		Delimiter: delimiter,
	}
}

func (c Configs) SetStringSlice(ref string, def []string, opt []string, optExt bool, cType string, cTypeOpt ConfigTypeOptions, delimiter *string) {
	c.setSlice(ref, def, opt, misc.String, optExt, cType, cTypeOpt, delimiter)
}

func (c Configs) SetBoolSlice(ref string, def []bool, opt []bool, optExt bool, cType string, cTypeOpt ConfigTypeOptions, delimiter *string) {
	c.setSlice(ref, def, opt, misc.Bool, optExt, cType, cTypeOpt, delimiter)
}

func (c Configs) SetInt64Slice(ref string, def []int64, opt []int64, optExt bool, cType string, cTypeOpt ConfigTypeOptions, delimiter *string) {
	c.setSlice(ref, def, opt, misc.Int64, optExt, cType, cTypeOpt, delimiter)
}

func (c Configs) SetFloat64Slice(ref string, def []float64, opt []float64, optExt bool, cType string, cTypeOpt ConfigTypeOptions, delimiter *string) {
	c.setSlice(ref, def, opt, misc.Float64, optExt, cType, cTypeOpt, delimiter)
}

func (o ConfigTypeOptions) set(ref string, val any, dType misc.DataType) {
	o[ref] = configTypeOption{
		Value:    val,
		DataType: dType,
	}
}

func (o ConfigTypeOptions) SetString(ref string, val string) {
	o.set(ref, val, misc.String)
}

func (o ConfigTypeOptions) SetBool(ref string, val bool) {
	o.set(ref, val, misc.Bool)
}

func (o ConfigTypeOptions) SetInt64(ref string, val int64) {
	o.set(ref, val, misc.Int64)
}

func (o ConfigTypeOptions) SetFloat64(ref string, val float64) {
	o.set(ref, val, misc.Float64)
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
