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

package validator

import "fmt"

func NumberCompare(params map[string]any) (bool, error) {
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
