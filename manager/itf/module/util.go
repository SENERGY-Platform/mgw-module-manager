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
	"code.cloudfoundry.org/bytefmt"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"math"
	"strconv"
	"strings"
)

type tmpConfigValue ConfigValue

type dataTypeValidator func(i any) bool

var dataTypeValidatorMap = map[DataType]dataTypeValidator{
	TextData:  textDataValidator,
	BoolData:  boolDataValidator,
	IntData:   intDataValidator,
	FloatData: floatDataValidator,
}

var textDataValidator dataTypeValidator = func(i any) (ok bool) {
	_, ok = i.(string)
	return
}

var boolDataValidator dataTypeValidator = func(i any) (ok bool) {
	_, ok = i.(bool)
	return
}

var intDataValidator dataTypeValidator = func(i any) bool {
	if _, ok := i.(int); ok {
		return true
	}
	if v, ok := i.(float64); ok {
		if _, f := math.Modf(v); f == 0 {
			return true
		}
	}
	return false
}

var floatDataValidator dataTypeValidator = func(i any) bool {
	if _, ok := i.(float64); ok {
		return true
	}
	if _, ok := i.(int); ok {
		return true
	}
	return false
}

func (m *ModuleType) parse(s string) error {
	if t, ok := ModuleTypeMap[s]; ok {
		*m = t
	} else {
		return fmt.Errorf("unknown module type '%s'", s)
	}
	return nil
}

func (m *ModuleType) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	return m.parse(s)
}

func (m *ModuleType) UnmarshalYAML(yn *yaml.Node) (err error) {
	var s string
	if err = yn.Decode(&s); err != nil {
		return
	}
	return m.parse(s)
}

func (d *DeploymentType) parse(s string) error {
	if t, ok := DeploymentTypeMap[s]; ok {
		*d = t
	} else {
		return fmt.Errorf("unknown deployment type '%s'", s)
	}
	return nil
}

func (d *DeploymentType) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	return d.parse(s)
}

func (d *DeploymentType) UnmarshalYAML(yn *yaml.Node) (err error) {
	var s string
	if err = yn.Decode(&s); err != nil {
		return
	}
	return d.parse(s)
}

func (i *ModuleID) parse(s string) error {
	if !strings.Contains(s, "/") || strings.Contains(s, "//") || strings.HasPrefix(s, "/") {
		return fmt.Errorf("invalid module ID format '%s'", s)
	} else {
		*i = ModuleID(s)
	}
	return nil
}

func (i *ModuleID) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	return i.parse(s)
}

func (i *ModuleID) UnmarshalYAML(yn *yaml.Node) (err error) {
	var s string
	if err = yn.Decode(&s); err != nil {
		return
	}
	return i.parse(s)
}

func (c *SrvDepCondition) parse(s string) error {
	if t, ok := SrvDepConditionMap[s]; ok {
		*c = t
	} else {
		return fmt.Errorf("unknown condition type '%s'", s)
	}
	return nil
}

func (c *SrvDepCondition) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	return c.parse(s)
}

func (c *SrvDepCondition) UnmarshalYAML(yn *yaml.Node) (err error) {
	var s string
	if err = yn.Decode(&s); err != nil {
		return
	}
	return c.parse(s)
}

func (d *DataType) parse(s string) error {
	if t, ok := DataTypeMap[s]; ok {
		*d = t
	} else {
		return fmt.Errorf("unknown data type '%s'", s)
	}
	return nil
}

func (d *DataType) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	return d.parse(s)
}

func (d *DataType) UnmarshalYAML(yn *yaml.Node) (err error) {
	var s string
	if err = yn.Decode(&s); err != nil {
		return
	}
	return d.parse(s)
}

func (v *ConfigValue) parse(tcv tmpConfigValue) error {
	validator := dataTypeValidatorMap[tcv.Type]
	if tcv.Value != nil && !validator(tcv.Value) {
		return fmt.Errorf("invalid type: config 'data' must be of '%s'", tcv.Type)
	}
	if tcv.Options != nil && len(tcv.Options) > 0 {
		for _, option := range tcv.Options {
			if !validator(option) {
				return fmt.Errorf("invalid type: config 'options' must contain values of '%s'", tcv.Type)
			}
		}
	}
	if tcv.Type == IntData {
		if tcv.Value != nil {
			tcv.Value = toInt(tcv.Value)
		}
		if tcv.Options != nil && len(tcv.Options) > 0 {
			for i := 0; i < len(tcv.Options); i++ {
				tcv.Options[i] = toInt(tcv.Options[i])
			}
		}
	}
	*v = ConfigValue(tcv)
	return nil
}

func (v *ConfigValue) UnmarshalJSON(b []byte) (err error) {
	var tcv tmpConfigValue
	if err = json.Unmarshal(b, &tcv); err != nil {
		return
	}
	return v.parse(tcv)
}

func (v *ConfigValue) UnmarshalYAML(yn *yaml.Node) (err error) {
	var tcv tmpConfigValue
	if err = yn.Decode(&tcv); err != nil {
		return
	}
	return v.parse(tcv)
}

func (p *Port) parse(itf any) error {
	switch v := itf.(type) {
	case int:
		*p = Port(strconv.FormatInt(int64(v), 10))
	case float64:
		if _, f := math.Modf(v); f > 0 {
			return fmt.Errorf("invlid port: %v", v)
		}
		*p = Port(strconv.FormatInt(int64(v), 10))
	case string:
		parts := strings.Split(v, "-")
		if len(parts) > 2 {
			return fmt.Errorf("invalid port range: %s", v)
		}
		for i := 0; i < len(parts); i++ {
			_, err := strconv.ParseInt(parts[i], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid port: %s", v)
			}
		}
		*p = Port(v)
	default:
		return fmt.Errorf("invlid port: %v", v)
	}
	return nil
}

func (p *Port) IsRange() bool {
	if strings.Contains(string(*p), "-") {
		return true
	}
	return false
}

func (p *Port) Range() (ports []int) {
	parts := strings.Split(string(*p), "-")
	start, _ := strconv.ParseInt(parts[0], 10, 64)
	if len(parts) > 1 {
		end, _ := strconv.ParseInt(parts[1], 10, 64)
		for i := start; i <= end; i++ {
			ports = append(ports, int(i))
		}
	} else {
		ports = append(ports, int(start))
	}
	return
}

func (p *Port) Int() int {
	i, _ := strconv.ParseInt(string(*p), 10, 64)
	return int(i)
}

func (p *Port) UnmarshalJSON(b []byte) (err error) {
	var itf any
	if err = json.Unmarshal(b, &itf); err != nil {
		return
	}
	return p.parse(itf)
}

func (p *Port) UnmarshalYAML(yn *yaml.Node) (err error) {
	var itf any
	if err = yn.Decode(&itf); err != nil {
		return
	}
	return p.parse(itf)
}

func (fb *ByteFmt) parse(itf any) error {
	switch v := itf.(type) {
	case int:
		*fb = ByteFmt(v)
	case float64:
		if _, f := math.Modf(v); f > 0 {
			return fmt.Errorf("invalid size: %v", v)
		}
		*fb = ByteFmt(v)
	case string:
		bytes, err := bytefmt.ToBytes(v)
		if err != nil {
			return fmt.Errorf("invalid size: %s", err)
		}
		*fb = ByteFmt(bytes)
	default:
		return fmt.Errorf("invalid size: %v", v)
	}
	return nil
}

func (fb *ByteFmt) UnmarshalJSON(b []byte) (err error) {
	var itf any
	if err = json.Unmarshal(b, &itf); err != nil {
		return
	}
	return fb.parse(itf)
}

func (fb *ByteFmt) UnmarshalYAML(yn *yaml.Node) (err error) {
	var itf any
	if err = yn.Decode(&itf); err != nil {
		return
	}
	return fb.parse(itf)
}

func (rtb ResourceTargetBase) Values() (values []string) {
	values = append(values, rtb.MountPoint)
	if rtb.Services != nil {
		values = append(values, rtb.Services...)
	}
	return
}

func (ct ConfigTarget) Values() (values []string) {
	values = append(values, ct.RefVar)
	if ct.Services != nil {
		values = append(values, ct.Services...)
	}
	return
}
