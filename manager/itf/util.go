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
	"math"
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

type TmpConfigValue ConfigValue

func (v *ConfigValue) UnmarshalJSON(b []byte) (err error) {
	var tcv TmpConfigValue
	if err = json.Unmarshal(b, &tcv); err != nil {
		return
	}
	validator := dataTypeValidatorMap[tcv.Type]
	if tcv.Data != nil && !validator(tcv.Data) {
		return fmt.Errorf("invalid type: config 'data' must be of '%s'", tcv.Type)
	}
	if tcv.Options != nil && len(tcv.Options) > 0 {
		for _, option := range tcv.Options {
			if !validator(option) {
				return fmt.Errorf("invalid type: config 'options' must contain values of '%s'", tcv.Type)
			}
		}
	}
	if tcv.Constraints != nil {
		if tcv.Type == BoolData {
			return fmt.Errorf("type '%s' does not support constraints", tcv.Type)
		}
		errFmt := "invalid type: config value constraint '%s' must be of '%s'"
		if !validator(tcv.Constraints.Max) {
			return fmt.Errorf(errFmt, "max", tcv.Type)
		}
		if !validator(tcv.Constraints.Min) {
			return fmt.Errorf(errFmt, "min", tcv.Type)
		}
		if tcv.Constraints.Step != nil {
			if !validator(tcv.Constraints.Step) {
				return fmt.Errorf(errFmt, "step", tcv.Type)
			}
		}
	}
	if tcv.Type == IntData {
		if tcv.Data != nil {
			tcv.Data = toInt(tcv.Data)
		}
		if tcv.Options != nil && len(tcv.Options) > 0 {
			for i := 0; i < len(tcv.Options); i++ {
				tcv.Options[i] = toInt(tcv.Options[i])
			}
		}
		if tcv.Constraints != nil {
			tcv.Constraints.Min = toInt(tcv.Constraints.Min)
			tcv.Constraints.Max = toInt(tcv.Constraints.Max)
			if tcv.Constraints.Step != nil {
				tcv.Constraints.Step = toInt(tcv.Constraints.Step)
			}
		}
	}
	*v = ConfigValue(tcv)
	return
}

func toInt(i any) int64 {
	return int64(i.(float64))
}

type dataTypeValidator func(i any) bool

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

var dataTypeValidatorMap = map[DataType]dataTypeValidator{
	TextData:  textDataValidator,
	BoolData:  boolDataValidator,
	IntData:   intDataValidator,
	FloatData: floatDataValidator,
}
