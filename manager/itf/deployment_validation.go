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
	"fmt"
	"math"
	"reflect"
)

//func ValidateBase(b Base) error {
//	if !module.IsValidModuleID(b.ModuleID) {
//		return fmt.Errorf("invalid module ID format '%s'", t.ModuleID)
//	}
//	if !module.IsValidSemVer(b.ModuleVersion) {
//		return fmt.Errorf("invalid version format '%s'", t.ModuleVersion)
//	}
//	for ref, val := range b.Resources {
//		if ref == "" {
//			return errors.New("invalid resource reference")
//		}
//		if err := validateInput(input.Input); err != nil {
//			return fmt.Errorf("invalid input for resource '%s': %s", ref, err)
//		}
//	}
//	for ref, val := range b.Secrets {
//		if ref == "" {
//			return errors.New("invalid secret reference")
//		}
//		if err := validateInput(input.Input); err != nil {
//			return fmt.Errorf("invalid input for secret '%s': %s", ref, err)
//		}
//	}
//	for ref, val := range b.Configs {
//		if ref == "" {
//			return errors.New("invalid config reference")
//		}
//		if err := validateInput(input.Input); err != nil {
//			return fmt.Errorf("invalid input for config '%s': %s", ref, err)
//		}
//	}
//	return nil
//}

func ValidateInput(val any, t reflect.Kind) error {
	switch t {
	case reflect.String:
		if _, ok := val.(string); !ok {
			return fmt.Errorf("type missmatch: string != %T", val)
		}
	case reflect.Bool:
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("type missmatch: bool != %T", val)
		}
	case reflect.Int64:
		switch v := val.(type) {
		case uint, uint8, uint16, uint32, uint64:
		case int, int8, int16, int32, int64:
		case float32:
			if _, fr := math.Modf(float64(v)); fr != 0 {
				return fmt.Errorf("type missmatch: integer != %T", val)
			}
		case float64:
			if _, fr := math.Modf(v); fr != 0 {
				return fmt.Errorf("type missmatch: integer != %T", val)
			}
		default:
			return fmt.Errorf("type missmatch: integer != %T", val)
		}
	case reflect.Float64:
		switch val.(type) {
		case float32, float64:
		default:
			return fmt.Errorf("type missmatch: float != %T", val)
		}
	default:
		return fmt.Errorf("unknown data type '%s'", t)
	}
	return nil
}
