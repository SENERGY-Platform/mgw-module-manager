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
	"encoding/json"
	"fmt"
	"math"
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
	if _, ok := i.(float64); !ok {
		return false
	}
	if _, f := math.Modf(i.(float64)); f > 0 {
		return false
	}
	return true
}

var floatDataValidator dataTypeValidator = func(i any) (ok bool) {
	_, ok = i.(float64)
	return
}

func toInt(i any) int64 {
	return int64(i.(float64))
}

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

func (c *SrvDepCondition) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	if t, ok := SrvDepConditionMap[s]; ok {
		*c = t
	} else {
		err = fmt.Errorf("unknown condition type '%s'", s)
	}
	return
}

func (d *DataType) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return
	}
	if t, ok := DataTypeMap[s]; ok {
		*d = t
	} else {
		err = fmt.Errorf("unknown data type '%s'", s)
	}
	return
}

func (v *ConfigValue) UnmarshalJSON(b []byte) (err error) {
	var tcv tmpConfigValue
	if err = json.Unmarshal(b, &tcv); err != nil {
		return
	}
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
	return
}
