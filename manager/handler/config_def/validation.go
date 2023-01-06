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

func getParamValue(params map[string]any, key string) (any, error) {
	if params == nil {
		return nil, errors.New("no parameters")
	}
	v, ok := params[key]
	if !ok {
		return nil, fmt.Errorf("parameter '%s' required", key)
	}
	return v, nil
}

func getParamValueGen[T any](params map[string]any, key string) (v T, err error) {
	if p, e := getParamValue(params, key); e != nil {
		err = e
	} else {
		var ok bool
		if v, ok = p.(T); !ok {
			err = fmt.Errorf("parameter '%s' invalid data type: %T != %T", key, p, *new(T))
		}
	}
	return
}

func RegexValidator(params map[string]any) (bool, error) {
	str, err := getParamValueGen[string](params, "string")
	if err != nil {
		return false, err
	}
	p, err := getParamValueGen[string](params, "pattern")
	if err != nil {
		return false, err
	}
	re, err := regexp.Compile(p)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern '%s'", p)
	}
	return re.MatchString(str), nil
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

func NumberCompareValidator(params map[string]any) (bool, error) {
	o, err := getParamValueGen[string](params, "operator")
	if err != nil {
		return false, err
	}
	av, err := getParamValue(params, "a")
	if err != nil {
		return false, err
	}
	switch a := av.(type) {
	case int64:
		b, err := getParamValueGen[int64](params, "b")
		if err != nil {
			return false, err
		}
		return compareNumber(a, b, o)
	case float64:
		b, err := getParamValueGen[float64](params, "b")
		if err != nil {
			return false, err
		}
		return compareNumber(a, b, o)
	default:
		return false, fmt.Errorf("invalid data type: %T != int64 | float64", a)
	}
}

func TextLenCompareValidator(params map[string]any) (bool, error) {
	o, err := getParamValueGen[string](params, "operator")
	if err != nil {
		return false, err
	}
	s, err := getParamValueGen[string](params, "string")
	if err != nil {
		return false, err
	}
	l, err := getParamValueGen[int64](params, "length")
	if err != nil {
		return false, err
	}
	return compareNumber(int64(utf8.RuneCountInString(s)), l, o)
}
