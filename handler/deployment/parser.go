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

package deployment

import (
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"math"
)

func parseCfgVal(val any, dataType module.DataType) (v any, err error) {
	switch dataType {
	case module.StringType:
		v, err = parseString(val)
	case module.BoolType:
		v, err = parseBool(val)
	case module.Int64Type:
		v, err = parseInt64(val)
	case module.Float64Type:
		v, err = parseFloat64(val)
	default:
		return nil, fmt.Errorf("unknown data type '%s'", dataType)
	}
	return
}

func parseCfgValSlice(val any, dataType module.DataType) (v any, err error) {
	vSl, ok := val.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid data type '%T'", val)
	}
	switch dataType {
	case module.StringType:
		v, err = toTSlice(vSl, parseString)
	case module.BoolType:
		v, err = toTSlice(vSl, parseBool)
	case module.Int64Type:
		v, err = toTSlice(vSl, parseInt64)
	case module.Float64Type:
		v, err = toTSlice(vSl, parseFloat64)
	default:
		return nil, fmt.Errorf("unknown data type '%s'", dataType)
	}
	return
}

func toTSlice[T any](sl []any, pf func(any) (T, error)) ([]T, error) {
	var vSl []T
	for _, v := range sl {
		val, err := pf(v)
		if err != nil {
			return nil, err
		}
		vSl = append(vSl, val)
	}
	return vSl, nil
}

func parseString(val any) (string, error) {
	v, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("invalid data type '%T'", val)
	}
	return v, nil
}

func parseBool(val any) (bool, error) {
	v, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("invalid data type '%T'", val)
	}
	return v, nil
}

func float64ToInt64(val float64) (int64, error) {
	i, fr := math.Modf(val)
	if fr > 0 {
		return 0, fmt.Errorf("invalid data type '%T'", val)
	}
	return int64(i), nil
}

func parseInt64(val any) (int64, error) {
	var i int64
	var err error
	switch v := val.(type) {
	case int:
		i = int64(v)
	case int8:
		i = int64(v)
	case int16:
		i = int64(v)
	case int32:
		i = int64(v)
	case int64:
		i = v
	case float32:
		i, err = float64ToInt64(float64(v))
	case float64:
		i, err = float64ToInt64(v)
	default:
		err = fmt.Errorf("invalid data type '%T'", val)
	}
	return i, err
}

func parseFloat64(val any) (float64, error) {
	var f float64
	switch v := val.(type) {
	case float32:
		f = float64(v)
	case float64:
		f = v
	default:
		return f, fmt.Errorf("invalid data type '%T'", val)
	}
	return f, nil
}
