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
	"strings"
)

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

func (i *ID) parse(s string) error {
	if !strings.Contains(s, "/") || strings.Contains(s, "//") || strings.HasPrefix(s, "/") {
		return fmt.Errorf("invalid module ID format '%s'", s)
	} else {
		*i = ID(s)
	}
	return nil
}

func (i *ID) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	return i.parse(s)
}

func (i *ID) UnmarshalYAML(yn *yaml.Node) (err error) {
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
