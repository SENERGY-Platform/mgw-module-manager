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

package parser

import (
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"strconv"
	"strings"
)

func DataTypeToString(val any, dataType module.DataType) (string, error) {
	switch dataType {
	case module.StringType:
		s, err := toString(val)
		if err != nil {
			return "", err
		}
		return s, nil
	case module.BoolType:
		b, err := toBool(val)
		if err != nil {
			return "", err
		}
		return strconv.FormatBool(b), nil
	case module.Int64Type:
		i, err := toInt64(val)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(i, 10), nil
	case module.Float64Type:
		f, err := toFloat64(val)
		if err != nil {
			return "", err
		}
		return strconv.FormatFloat(f, 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("unknown data type '%s'", dataType)
	}
}

func DataTypeToStringList(val any, delimiter string, dataType module.DataType) (string, error) {
	var sSl []string
	switch dataType {
	case module.StringType:
		sl, err := toSliceT[string](val)
		if err != nil {
			return "", err
		}
		sSl = sl
	case module.BoolType:
		sl, err := toSliceT[bool](val)
		if err != nil {
			return "", err
		}
		for _, b := range sl {
			sSl = append(sSl, strconv.FormatBool(b))
		}
	case module.Int64Type:
		sl, err := toSliceT[int64](val)
		if err != nil {
			return "", err
		}
		for _, i := range sl {
			sSl = append(sSl, strconv.FormatInt(i, 10))
		}
	case module.Float64Type:
		sl, err := toSliceT[float64](val)
		if err != nil {
			return "", err
		}
		for _, f := range sl {
			sSl = append(sSl, strconv.FormatFloat(f, 'f', -1, 64))
		}
	default:
		return "", fmt.Errorf("unknown data type '%s'", dataType)
	}
	return strings.Join(sSl, delimiter), nil
}

func AnyToDataType(val any, dataType module.DataType) (v any, err error) {
	switch dataType {
	case module.StringType:
		v, err = toString(val)
	case module.BoolType:
		v, err = toBool(val)
	case module.Int64Type:
		v, err = toInt64(val)
	case module.Float64Type:
		v, err = toFloat64(val)
	default:
		return nil, fmt.Errorf("unknown data type '%s'", dataType)
	}
	return
}

func AnyToDataTypeSlice(val any, dataType module.DataType) (v any, err error) {
	vSl, ok := val.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid data type '%T'", val)
	}
	if len(vSl) == 0 {
		return nil, errors.New("no values to parse")
	}
	switch dataType {
	case module.StringType:
		v, err = sliceToSliceT(vSl, toString)
	case module.BoolType:
		v, err = sliceToSliceT(vSl, toBool)
	case module.Int64Type:
		v, err = sliceToSliceT(vSl, toInt64)
	case module.Float64Type:
		v, err = sliceToSliceT(vSl, toFloat64)
	default:
		return nil, fmt.Errorf("unknown data type '%s'", dataType)
	}
	return
}

func toSliceT[T any](val any) ([]T, error) {
	sl, ok := val.([]T)
	if !ok {
		return nil, fmt.Errorf("invalid data type '%T'", val)
	}
	return sl, nil
}

func sliceToSliceT[T any](sl []any, pf func(any) (T, error)) ([]T, error) {
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
