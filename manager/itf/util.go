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

type TmpValue Value

func (v *Value) UnmarshalJSON(b []byte) (err error) {
	var iv TmpValue
	if err = json.Unmarshal(b, &iv); err != nil {
		return
	}
	errFmt := "invalid type: value must be of '%s'"
	switch iv.Type {
	case TextData:
		if _, ok := iv.Data.(string); !ok {
			return fmt.Errorf(errFmt, TextData)
		}
	case BoolData:
		if _, ok := iv.Data.(bool); !ok {
			return fmt.Errorf(errFmt, BoolData)
		}
	case IntData:
		if _, ok := iv.Data.(float64); !ok {
			return fmt.Errorf(errFmt, IntData)
		}
		i, f := math.Modf(iv.Data.(float64))
		if f > 0 {
			return fmt.Errorf(errFmt, IntData)
		}
		iv.Data = i
	case FloatData:
		if _, ok := iv.Data.(float64); !ok {
			return fmt.Errorf(errFmt, FloatData)
		}
	}
	*v = Value(iv)
	return
}
