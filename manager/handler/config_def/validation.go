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
	"unicode/utf8"
)

func RegexValidator(val any, parameter map[string]any) (bool, error) {
	v, ok := val.(string)
	if !ok {
		return false, fmt.Errorf("invalid data type: %T != string", val)
	}
	if parameter == nil {
		return false, errors.New("missing parameters")
	}
	pattern, ok := parameter["pattern"]
	if !ok {
		return false, errors.New("missing 'pattern' parameter")
	}
	p, ok := pattern.(string)
	if !ok {
		return false, fmt.Errorf("invalid data type: %T != string", pattern)
	}
	re, err := regexp.Compile(p)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern '%s'", p)
	}
	return re.MatchString(v), nil
}

type number interface {
	int64 | float64
}

func compareNumber[T number](a T, b T, o string) (bool, error) {
	switch o {
	case ">":
		return a > b, nil
	case "<":
		return a < b, nil
	case "=":
		return a == b, nil
	case ">=":
		return a >= b, nil
	case "<=":
		return a <= b, nil
	default:
		return false, fmt.Errorf("invalid operator '%s'", o)
	}
}

func getCompareParameter(parameter map[string]any) (b any, o string, err error) {
	if parameter == nil {
		return nil, "", errors.New("missing parameters")
	}
	op, ok := parameter["operator"]
	if !ok {
		err = errors.New("missing 'operator' parameter")
		return
	}
	o, ok = op.(string)
	if !ok {
		err = fmt.Errorf("invalid data type: %T != string", op)
		return
	}
	b, ok = parameter["b"]
	if !ok {
		err = errors.New("missing 'b' parameter")
		return
	}
	return
}

func NumberCompareValidator(val any, parameter map[string]any) (bool, error) {
	bp, o, err := getCompareParameter(parameter)
	if err != nil {
		return false, err
	}
	switch a := val.(type) {
	case int64:
		b, k := bp.(int64)
		if !k {
			return false, fmt.Errorf("invalid data type: %T != int", bp)
		}
		return compareNumber(a, b, o)
	case float64:
		b, k := bp.(float64)
		if !k {
			return false, fmt.Errorf("invalid data type: %T != float", bp)
		}
		return compareNumber(a, b, o)
	default:
		return false, fmt.Errorf("invalid data type: %T != int | float", a)
	}
}

func TextLenCompareValidator(val any, parameter map[string]any) (bool, error) {
	v, ok := val.(string)
	if !ok {
		return false, fmt.Errorf("invalid data type: %T != string", val)
	}
	bp, o, err := getCompareParameter(parameter)
	if err != nil {
		return false, err
	}
	b, k := bp.(int64)
	if !k {
		return false, fmt.Errorf("invalid data type: %T != int", b)
	}
	return compareNumber(int64(utf8.RuneCountInString(v)), b, o)
}
