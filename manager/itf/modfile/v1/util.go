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

package v1

import (
	"code.cloudfoundry.org/bytefmt"
	"fmt"
	"gopkg.in/yaml.v3"
	"math"
	"strconv"
	"strings"
)

//type tmpConfigValue ConfigValue
//
//type dataTypeValidator func(i any) bool
//
//var dataTypeValidatorMap = map[module.DataType]dataTypeValidator{
//	module.TextData:  textDataValidator,
//	module.BoolData:  boolDataValidator,
//	module.IntData:   intDataValidator,
//	module.FloatData: floatDataValidator,
//}
//
//var textDataValidator dataTypeValidator = func(i any) (ok bool) {
//	_, ok = i.(string)
//	return
//}
//
//var boolDataValidator dataTypeValidator = func(i any) (ok bool) {
//	_, ok = i.(bool)
//	return
//}
//
//var intDataValidator dataTypeValidator = func(i any) bool {
//	if _, ok := i.(int); ok {
//		return true
//	}
//	if v, ok := i.(float64); ok {
//		if _, f := math.Modf(v); f == 0 {
//			return true
//		}
//	}
//	return false
//}
//
//var floatDataValidator dataTypeValidator = func(i any) bool {
//	if _, ok := i.(float64); ok {
//		return true
//	}
//	if _, ok := i.(int); ok {
//		return true
//	}
//	return false
//}
//
//func (v *ConfigValue) parse(tcv tmpConfigValue) error {
//	validator := dataTypeValidatorMap[tcv.Type]
//	if tcv.Value != nil && !validator(tcv.Value) {
//		return fmt.Errorf("invalid type: config 'value' must be of '%s'", tcv.Type)
//	}
//	if tcv.Options != nil && len(tcv.Options) > 0 {
//		for _, option := range tcv.Options {
//			if !validator(option) {
//				return fmt.Errorf("invalid type: config 'options' must contain values of '%s'", tcv.Type)
//			}
//		}
//	}
//	if tcv.Type == module.IntData {
//		if tcv.Value != nil {
//			if f, ok := tcv.Value.(float64); ok {
//				tcv.Value = int64(f)
//			}
//		}
//		if tcv.Options != nil {
//			for i := 0; i < len(tcv.Options); i++ {
//				if f, ok := tcv.Options[i].(float64); ok {
//					tcv.Options[i] = int64(f)
//				}
//			}
//		}
//	}
//	if tcv.Type == module.FloatData {
//		if tcv.Value != nil {
//			if f, ok := tcv.Value.(int); ok {
//				tcv.Value = float64(f)
//			}
//		}
//		if tcv.Options != nil {
//			for i := 0; i < len(tcv.Options); i++ {
//				if f, ok := tcv.Options[i].(int); ok {
//					tcv.Options[i] = float64(f)
//				}
//			}
//		}
//	}
//	*v = ConfigValue(tcv)
//	return nil
//}
//
//func (v *ConfigValue) UnmarshalJSON(b []byte) (err error) {
//	var tcv tmpConfigValue
//	if err = json.Unmarshal(b, &tcv); err != nil {
//		return
//	}
//	return v.parse(tcv)
//}
//
//func (v *ConfigValue) UnmarshalYAML(yn *yaml.Node) (err error) {
//	var tcv tmpConfigValue
//	if err = yn.Decode(&tcv); err != nil {
//		return
//	}
//	return v.parse(tcv)
//}

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

func (p *Port) UnmarshalYAML(yn *yaml.Node) error {
	var itf any
	if err := yn.Decode(&itf); err != nil {
		return err
	}
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

func (fb *ByteFmt) UnmarshalYAML(yn *yaml.Node) (err error) {
	var itf any
	if err = yn.Decode(&itf); err != nil {
		return
	}
	return fb.parse(itf)
}
