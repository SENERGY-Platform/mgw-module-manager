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
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func ValueIsEqual(a, b pkg_models.Value) bool {
	if a.DataType != b.DataType {
		return false
	}
	if a.IsSlice != b.IsSlice {
		return false
	}
	if a.IsSlice {
		switch a.DataType {
		case constants.ValueDataTypeString:
			return slices.Equal(a.StringSlice, b.StringSlice)
		case constants.ValueDataTypeInt64:
			return slices.Equal(a.Int64Slice, b.Int64Slice)
		case constants.ValueDataTypeFloat64:
			return slices.Equal(a.Float64Slice, b.Float64Slice)
		case constants.ValueDataTypeBool:
			return slices.Equal(a.BoolSlice, b.BoolSlice)
		}
	} else {
		switch a.DataType {
		case constants.ValueDataTypeString:
			return a.String == b.String
		case constants.ValueDataTypeInt64:
			return a.Int64 == b.Int64
		case constants.ValueDataTypeFloat64:
			return a.Float64 == b.Float64
		case constants.ValueDataTypeBool:
			return a.Bool == b.Bool
		}
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
			return pkg_models.Value{}, fmt.Errorf("invalid data type '%T'", val) // TODO
		}
		switch dataType {
		case constants.ValueDataTypeString:
			for _, item := range anySlice {
				v, err := toString(item)
				if err != nil {
					return pkg_models.Value{}, err
				}
				config.StringSlice = append(config.StringSlice, v)
			}
		case constants.ValueDataTypeBool:
			for _, item := range anySlice {
				v, err := toBool(item)
				if err != nil {
					return pkg_models.Value{}, err
				}
				config.BoolSlice = append(config.BoolSlice, v)
			}
		case constants.ValueDataTypeInt64:
			for _, item := range anySlice {
				v, err := toInt64(item)
				if err != nil {
					return pkg_models.Value{}, err
				}
				config.Int64Slice = append(config.Int64Slice, v)
			}
		case constants.ValueDataTypeFloat64:
			for _, item := range anySlice {
				v, err := toFloat64(item)
				if err != nil {
					return pkg_models.Value{}, err
				}
				config.Float64Slice = append(config.Float64Slice, v)
			}
		default:
			return pkg_models.Value{}, fmt.Errorf("unknown data type '%s'", dataType) // TODO
		}
	} else {
		switch dataType {
		case constants.ValueDataTypeString:
			v, err := toString(val)
			if err != nil {
				return pkg_models.Value{}, err
			}
			config.String = v
		case constants.ValueDataTypeBool:
			v, err := toBool(val)
			if err != nil {
				return pkg_models.Value{}, err
			}
			config.Bool = v
		case constants.ValueDataTypeInt64:
			v, err := toInt64(val)
			if err != nil {
				return pkg_models.Value{}, err
			}
			config.Int64 = v
		case constants.ValueDataTypeFloat64:
			v, err := toFloat64(val)
			if err != nil {
				return pkg_models.Value{}, err
			}
			config.Float64 = v
		default:
			return pkg_models.Value{}, fmt.Errorf("unknown data type '%s'", dataType) // TODO
		}
	}
	return config, nil
}

func GetValueWithValidation(val any, moduleConfig external_models.ModuleLibConfigValue) (pkg_models.Value, error) {
	config := pkg_models.Value{
		IsSlice: moduleConfig.IsSlice,
	}
	if moduleConfig.IsSlice {
		anySlice, ok := val.([]any)
		if !ok {
			return pkg_models.Value{}, fmt.Errorf("invalid data type '%T'", val) // TODO
		}
		switch moduleConfig.DataType {
		case external_models.ModuleLibStringType:
			config.DataType = constants.ValueDataTypeString
			for _, item := range anySlice {
				v, err := toString(item)
				if err != nil {
					return pkg_models.Value{}, err
				}
				err = validateValue(v, moduleConfig)
				if err != nil {
					return pkg_models.Value{}, err
				}
				config.StringSlice = append(config.StringSlice, v)
			}
		case external_models.ModuleLibBoolType:
			config.DataType = constants.ValueDataTypeBool
			for _, item := range anySlice {
				v, err := toBool(item)
				if err != nil {
					return pkg_models.Value{}, err
				}
				err = validateValue(v, moduleConfig)
				if err != nil {
					return pkg_models.Value{}, err
				}
				config.BoolSlice = append(config.BoolSlice, v)
			}
		case external_models.ModuleLibInt64Type:
			config.DataType = constants.ValueDataTypeInt64
			for _, item := range anySlice {
				v, err := toInt64(item)
				if err != nil {
					return pkg_models.Value{}, err
				}
				err = validateValue(v, moduleConfig)
				if err != nil {
					return pkg_models.Value{}, err
				}
				config.Int64Slice = append(config.Int64Slice, v)
			}
		case external_models.ModuleLibFloat64Type:
			config.DataType = constants.ValueDataTypeFloat64
			for _, item := range anySlice {
				v, err := toFloat64(item)
				if err != nil {
					return pkg_models.Value{}, err
				}
				err = validateValue(v, moduleConfig)
				if err != nil {
					return pkg_models.Value{}, err
				}
				config.Float64Slice = append(config.Float64Slice, v)
			}
		default:
			return pkg_models.Value{}, fmt.Errorf("unknown data type '%s'", moduleConfig.DataType) // TODO
		}
	} else {
		switch moduleConfig.DataType {
		case external_models.ModuleLibStringType:
			config.DataType = constants.ValueDataTypeString
			v, err := toString(val)
			if err != nil {
				return pkg_models.Value{}, err
			}
			config.String = v
			err = validateValue(v, moduleConfig)
			if err != nil {
				return pkg_models.Value{}, err
			}
		case external_models.ModuleLibBoolType:
			config.DataType = constants.ValueDataTypeBool
			v, err := toBool(val)
			if err != nil {
				return pkg_models.Value{}, err
			}
			config.Bool = v
			err = validateValue(v, moduleConfig)
			if err != nil {
				return pkg_models.Value{}, err
			}
		case external_models.ModuleLibInt64Type:
			config.DataType = constants.ValueDataTypeInt64
			v, err := toInt64(val)
			if err != nil {
				return pkg_models.Value{}, err
			}
			config.Int64 = v
			err = validateValue(v, moduleConfig)
			if err != nil {
				return pkg_models.Value{}, err
			}
		case external_models.ModuleLibFloat64Type:
			config.DataType = constants.ValueDataTypeFloat64
			v, err := toFloat64(val)
			if err != nil {
				return pkg_models.Value{}, err
			}
			config.Float64 = v
			err = validateValue(v, moduleConfig)
			if err != nil {
				return pkg_models.Value{}, err
			}
		default:
			return pkg_models.Value{}, fmt.Errorf("unknown data type '%s'", moduleConfig.DataType) // TODO
		}
	}
	return config, nil
}

func ValueToInterface(config pkg_models.Value) (v interface{}) {
	switch config.DataType {
	case constants.ValueDataTypeString:
		if config.IsSlice {
			return config.StringSlice
		}
		return config.String
	case constants.ValueDataTypeInt64:
		if config.IsSlice {
			return config.Int64Slice
		}
		return config.Int64
	case constants.ValueDataTypeFloat64:
		if config.IsSlice {
			return config.Float64Slice
		}
		return config.Float64
	case constants.ValueDataTypeBool:
		if config.IsSlice {
			return config.BoolSlice
		}
		return config.Bool
	}
	return
}

func ValueToString(config pkg_models.Value) string {
	switch config.DataType {
	case constants.ValueDataTypeString:
		return config.String
	case constants.ValueDataTypeInt64:
		return strconv.FormatInt(config.Int64, 10)
	case constants.ValueDataTypeFloat64:
		return strconv.FormatFloat(config.Float64, 'f', -1, 64)
	case constants.ValueDataTypeBool:
		return strconv.FormatBool(config.Bool)
	}
	return ""
}

func SliceValueToString(config pkg_models.Value, delimiter string) string {
	var values []string
	switch config.DataType {
	case constants.ValueDataTypeString:
		values = config.StringSlice
	case constants.ValueDataTypeInt64:
		for _, i := range config.Int64Slice {
			values = append(values, strconv.FormatInt(i, 10))
		}
	case constants.ValueDataTypeFloat64:
		for _, f := range config.Float64Slice {
			values = append(values, strconv.FormatFloat(f, 'f', -1, 64))
		}
	case constants.ValueDataTypeBool:
		for _, b := range config.BoolSlice {
			values = append(values, strconv.FormatBool(b))
		}
	}
	return strings.Join(values, delimiter)
}

func validateValue[T comparable](val T, moduleConfig external_models.ModuleLibConfigValue) error {
	err := module_lib_validation_configs.ValidateValue(moduleConfig.Type, moduleConfig.TypeOpt, val)
	if err != nil {
		return err
	}
	if moduleConfig.Options != nil && !moduleConfig.OptExt {
		ok, err := module_lib_validation_configs.ValidateValueInOptions(val, moduleConfig.Options)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("not in options") // TODO
		}
	}
	return nil
}

func toString(val any) (string, error) {
	v, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("invalid data type '%T'", val) // TODO
	}
	return v, nil
}

func toBool(val any) (bool, error) {
	v, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("invalid data type '%T'", val) // TODO
	}
	return v, nil
}

func float64ToInt64(val float64) (int64, error) {
	i, fr := math.Modf(val)
	if fr > 0 {
		return 0, fmt.Errorf("invalid data type '%T'", val) // TODO
	}
	return int64(i), nil
}

func toInt64(val any) (int64, error) {
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
		err = fmt.Errorf("invalid data type '%T'", val) // TODO
	}
	return i, err
}

func toFloat64(val any) (float64, error) {
	var f float64
	switch v := val.(type) {
	case float32:
		f = float64(v)
	case float64:
		f = v
	default:
		return f, fmt.Errorf("invalid data type '%T'", val) // TODO
	}
	return f, nil
}
