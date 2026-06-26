/*
 * Copyright 2026 InfAI (CC SES)
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

package configs

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	module_lib_validation_configs "github.com/SENERGY-Platform/mgw-module-lib/validation/configs"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	pkg_constants "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func ValueIsEqual(a, b pkg_models.Value) bool {
	if a.DataType != b.DataType {
		return false
	}
	if a.IsSlice != b.IsSlice {
		return false
	}
	switch a.DataType {
	case pkg_constants.ValueDataTypeString:
		if a.IsSlice {
			return slices.Equal(a.StringSlice, b.StringSlice)
		}
		return a.String == b.String
	case pkg_constants.ValueDataTypeInt64:
		if a.IsSlice {
			return slices.Equal(a.Int64Slice, b.Int64Slice)
		}
		return a.Int64 == b.Int64
	case pkg_constants.ValueDataTypeFloat64:
		if a.IsSlice {
			return slices.Equal(a.Float64Slice, b.Float64Slice)
		}
		return a.Float64 == b.Float64
	case pkg_constants.ValueDataTypeBool:
		if a.IsSlice {
			return slices.Equal(a.BoolSlice, b.BoolSlice)
		}
		return a.Bool == b.Bool
	}
	return false
}

func GetValue(val any, dataType int, isSlice bool) (pkg_models.Value, error) {
	config := pkg_models.Value{
		DataType: dataType,
		IsSlice:  isSlice,
	}
	if isSlice {
		anySlice, ok := val.([]any)
		if !ok {
			return pkg_models.Value{}, errors.New(fmt.Sprintf("invalid data type: '%T'", val))
		}
		switch dataType {
		case pkg_constants.ValueDataTypeString:
			for _, item := range anySlice {
				v, err := toString(item)
				if err != nil {
					return pkg_models.Value{}, err
				}
				config.StringSlice = append(config.StringSlice, v)
			}
		case pkg_constants.ValueDataTypeBool:
			for _, item := range anySlice {
				v, err := toBool(item)
				if err != nil {
					return pkg_models.Value{}, err
				}
				config.BoolSlice = append(config.BoolSlice, v)
			}
		case pkg_constants.ValueDataTypeInt64:
			for _, item := range anySlice {
				v, err := toInt64(item)
				if err != nil {
					return pkg_models.Value{}, err
				}
				config.Int64Slice = append(config.Int64Slice, v)
			}
		case pkg_constants.ValueDataTypeFloat64:
			for _, item := range anySlice {
				v, err := toFloat64(item)
				if err != nil {
					return pkg_models.Value{}, err
				}
				config.Float64Slice = append(config.Float64Slice, v)
			}
		default:
			return pkg_models.Value{}, errors.New(fmt.Sprintf("unsuported data type: '%d'", dataType))
		}
	} else {
		switch dataType {
		case pkg_constants.ValueDataTypeString:
			v, err := toString(val)
			if err != nil {
				return pkg_models.Value{}, err
			}
			config.String = v
		case pkg_constants.ValueDataTypeBool:
			v, err := toBool(val)
			if err != nil {
				return pkg_models.Value{}, err
			}
			config.Bool = v
		case pkg_constants.ValueDataTypeInt64:
			v, err := toInt64(val)
			if err != nil {
				return pkg_models.Value{}, err
			}
			config.Int64 = v
		case pkg_constants.ValueDataTypeFloat64:
			v, err := toFloat64(val)
			if err != nil {
				return pkg_models.Value{}, err
			}
			config.Float64 = v
		default:
			return pkg_models.Value{}, errors.New(fmt.Sprintf("unsuported data type: '%d'", dataType))
		}
	}
	return config, nil
}

func GetDataType(moduleDataType string) int {
	return moduleDataTypeMap[moduleDataType]
}

func ValueToInterface(config pkg_models.Value) (v interface{}) {
	switch config.DataType {
	case pkg_constants.ValueDataTypeString:
		if config.IsSlice {
			return config.StringSlice
		}
		return config.String
	case pkg_constants.ValueDataTypeInt64:
		if config.IsSlice {
			return config.Int64Slice
		}
		return config.Int64
	case pkg_constants.ValueDataTypeFloat64:
		if config.IsSlice {
			return config.Float64Slice
		}
		return config.Float64
	case pkg_constants.ValueDataTypeBool:
		if config.IsSlice {
			return config.BoolSlice
		}
		return config.Bool
	}
	return
}

func ValueToString(config pkg_models.Value) string {
	switch config.DataType {
	case pkg_constants.ValueDataTypeString:
		return config.String
	case pkg_constants.ValueDataTypeInt64:
		return strconv.FormatInt(config.Int64, 10)
	case pkg_constants.ValueDataTypeFloat64:
		return strconv.FormatFloat(config.Float64, 'f', -1, 64)
	case pkg_constants.ValueDataTypeBool:
		return strconv.FormatBool(config.Bool)
	}
	return ""
}

func SliceValueToString(config pkg_models.Value, delimiter string) string {
	var values []string
	switch config.DataType {
	case pkg_constants.ValueDataTypeString:
		values = config.StringSlice
	case pkg_constants.ValueDataTypeInt64:
		for _, i := range config.Int64Slice {
			values = append(values, strconv.FormatInt(i, 10))
		}
	case pkg_constants.ValueDataTypeFloat64:
		for _, f := range config.Float64Slice {
			values = append(values, strconv.FormatFloat(f, 'f', -1, 64))
		}
	case pkg_constants.ValueDataTypeBool:
		for _, b := range config.BoolSlice {
			values = append(values, strconv.FormatBool(b))
		}
	}
	return strings.Join(values, delimiter)
}

func ValidateValue(value pkg_models.Value, moduleConfig external_models.ModuleLibConfigValue) error {
	modCfgType, ok := reverseModuleDataTypeMap[value.DataType]
	if !ok {
		return errors.New(fmt.Sprintf("unsuported data type: '%d'", value.DataType))
	}
	if modCfgType != moduleConfig.DataType {
		return errors.New(fmt.Sprintf("invalid data type: '%s' required", moduleConfig.DataType))
	}
	if value.IsSlice != moduleConfig.IsSlice {
		return errors.New("invalid slice declaration")
	}
	switch value.DataType {
	case pkg_constants.ValueDataTypeString:
		if value.IsSlice {
			return validateAndCheckValueSlice(value.StringSlice, moduleConfig)
		}
		return validateAndCheckValue(value.String, moduleConfig)
	case pkg_constants.ValueDataTypeInt64:
		if value.IsSlice {
			return validateAndCheckValueSlice(value.Int64Slice, moduleConfig)
		}
		return validateAndCheckValue(value.Int64, moduleConfig)
	case pkg_constants.ValueDataTypeFloat64:
		if value.IsSlice {
			return validateAndCheckValueSlice(value.Float64Slice, moduleConfig)
		}
		return validateAndCheckValue(value.Float64, moduleConfig)
	case pkg_constants.ValueDataTypeBool:
		if value.IsSlice {
			return validateAndCheckValueSlice(value.BoolSlice, moduleConfig)
		}
		return validateAndCheckValue(value.Bool, moduleConfig)
	default:
		return errors.New(fmt.Sprintf("unsuported data type: '%d'", value.DataType))
	}
}

func validateAndCheckValue[T comparable](val T, moduleConfig external_models.ModuleLibConfigValue) error {
	err := module_lib_validation_configs.ValidateValue(moduleConfig.Type, moduleConfig.TypeOpt, val)
	if err != nil {
		return err
	}
	if moduleConfig.Options != nil && !moduleConfig.OptExt {
		ok, err := module_lib_validation_configs.CheckValueInOptions(val, moduleConfig.Options)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New(fmt.Sprintf("value '%v' not in options %v", val, moduleConfig.Options))
		}
	}
	return nil
}

func validateAndCheckValueSlice[T comparable](valSl []T, moduleConfig external_models.ModuleLibConfigValue) error {
	err := module_lib_validation_configs.ValidateValueSlice(moduleConfig.Type, moduleConfig.TypeOpt, valSl)
	if err != nil {
		return err
	}
	if moduleConfig.Options != nil && !moduleConfig.OptExt {
		ok, err := module_lib_validation_configs.CheckValueSliceInOptions(valSl, moduleConfig.Options)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New(fmt.Sprintf("values %v not in options %v", valSl, moduleConfig.Options))
		}
	}
	return nil
}

func toString(val any) (string, error) {
	v, ok := val.(string)
	if !ok {
		return "", errors.New("invalid data type: 'string' required")
	}
	return v, nil
}

func toBool(val any) (bool, error) {
	v, ok := val.(bool)
	if !ok {
		return false, errors.New("invalid data type: 'boolean' required")
	}
	return v, nil
}

func float64ToInt64(val float64) (int64, bool) {
	i, fr := math.Modf(val)
	if fr > 0 {
		return 0, false
	}
	return int64(i), true
}

func toInt64(val any) (int64, error) {
	var i int64
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
		var ok bool
		i, ok = float64ToInt64(float64(v))
		if !ok {
			return 0, errors.New("invalid data type: 'integer' required")
		}
	case float64:
		var ok bool
		i, ok = float64ToInt64(v)
		if !ok {
			return 0, errors.New("invalid data type: 'integer' required")
		}
	default:
		return 0, errors.New("invalid data type: 'integer' required")
	}
	return i, nil
}

func toFloat64(val any) (float64, error) {
	var f float64
	switch v := val.(type) {
	case float32:
		f = float64(v)
	case float64:
		f = v
	default:
		return 0, errors.New("invalid data type: 'float' required")
	}
	return f, nil
}

var moduleDataTypeMap = map[string]int{
	external_models.ModuleLibStringType:  pkg_constants.ValueDataTypeString,
	external_models.ModuleLibInt64Type:   pkg_constants.ValueDataTypeInt64,
	external_models.ModuleLibFloat64Type: pkg_constants.ValueDataTypeFloat64,
	external_models.ModuleLibBoolType:    pkg_constants.ValueDataTypeBool,
}

var reverseModuleDataTypeMap = func() map[int]string {
	m := make(map[int]string)
	for s, i := range moduleDataTypeMap {
		m[i] = s
	}
	return m
}()
