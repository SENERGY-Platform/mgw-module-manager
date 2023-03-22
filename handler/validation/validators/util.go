/*
 * Copyright 2023 InfAI (CC SES)
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

package validators

import (
	"fmt"
)

func getParamValue[T any](params map[string]any, pKey string) (T, error) {
	v, ok := params[pKey]
	if !ok {
		return *new(T), fmt.Errorf("parameter '%s' not defined", pKey)
	}
	pVal, ok := v.(T)
	if !ok {
		return *new(T), fmt.Errorf("parameter '%s' invalid data type: %T != %T", pKey, v, *new(T))
	}
	return pVal, nil
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
