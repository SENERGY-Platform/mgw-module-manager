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

package config_def

import (
	"errors"
	"fmt"
	"regexp"
)

func RegexValidator(val any, parameter map[string]any) (bool, error) {
	v, ok := val.(string)
	if !ok {
		return false, fmt.Errorf("regex validation invalid data type: %T != string", val)
	}
	if parameter == nil {
		return false, errors.New("regex validation requires parameters")
	}
	pattern, ok := parameter["pattern"]
	if !ok {
		return false, errors.New("regex validation requires 'pattern' parameter")
	}
	p, ok := pattern.(string)
	if !ok {
		return false, fmt.Errorf("regex pattern invalid data type: %T != string", pattern)
	}
	re, err := regexp.Compile(p)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern '%s'", p)
	}
	return re.MatchString(v), nil
}

func NumberCompareValidator(val any, parameter map[string]any) (bool, error) {
	if parameter == nil {
		return false, errors.New("number compare validation requires parameters")
	}
	b, ok := parameter["b"]
	if !ok {
		return false, errors.New("number compare validation requires 'b' parameter")
	}
	o, ok := parameter["operator"]
	if !ok {
		return false, errors.New("number compare validation requires 'operator' parameter")
	}
	switch v := val.(type) {
	case int64:
		bv, ok := b.(int64)
		if !ok {
			return false, fmt.Errorf("number compare validation invalid data type: %T != int", b)
		}
		switch o {
		case ">":
			return v > bv, nil
		case "<":
			return v < bv, nil
		case "=":
			return v == bv, nil
		case ">=":
			return v >= bv, nil
		case "<=":
			return v <= bv, nil
		default:
			return false, fmt.Errorf("number compare validation invalid operator '%s'", o)
		}
	case float64:
		bv, ok := b.(float64)
		if !ok {
			return false, fmt.Errorf("number compare validation invalid data type: %T != float", b)
		}
		switch o {
		case ">":
			return v > bv, nil
		case "<":
			return v < bv, nil
		case "=":
			return v == bv, nil
		case ">=":
			return v >= bv, nil
		case "<=":
			return v <= bv, nil
		default:
			return false, fmt.Errorf("number compare validation invalid operator '%s'", o)
		}
	default:
		return false, fmt.Errorf("number compare validation invalid data type: %T != int | float", val)
	}
}
