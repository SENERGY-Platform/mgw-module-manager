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

func NumberCompare(params map[string]any) error {
	o, err := getParamValue[string](params, "operator")
	if err != nil {
		return err
	}
	av, err := getParamValue[any](params, "a")
	if err != nil {
		return err
	}
	switch a := av.(type) {
	case int64:
		b, err := getParamValue[int64](params, "b")
		if err != nil {
			return err
		}
		ok, err := compareNumber(a, b, o)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%d %s %d", a, o, b)
		}
	case float64:
		b, err := getParamValue[float64](params, "b")
		if err != nil {
			return err
		}
		ok, err := compareNumber(a, b, o)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%f %s %f", a, o, b)
		}
	default:
		return fmt.Errorf("invalid data type: %T != int64 | float64", a)
	}
	return nil
}
