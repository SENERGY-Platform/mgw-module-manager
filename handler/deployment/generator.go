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
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"path"
	"strconv"
	"strings"
)

func genVolumeName(s, v string) string {
	return "MGW_" + genHash(s, v)
}

func genBindMountPath(s, p string) string {
	return path.Join(s, p)
}

func genSrvName(s, r string) string {
	return "MGW_" + genHash(s, r)
}

func genHash(str ...string) string {
	hash := sha1.New()
	for _, s := range str {
		hash.Write([]byte(s))
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash.Sum(nil))
}

func genConfigEnvValues(mConfigs module.Configs, dConfigs map[string]any) (map[string]string, error) {
	envVals := make(map[string]string)
	for ref, mConfig := range mConfigs {
		val, ok := dConfigs[ref]
		if !ok {
			if mConfig.Required {
				if mConfig.Default != nil {
					val = mConfig.Default
				} else {
					return nil, fmt.Errorf("config '%s' required", ref)
				}
			} else {
				if mConfig.Default != nil {
					val = mConfig.Default
				} else {
					continue
				}
			}
		}
		var s string
		var err error
		if mConfig.IsSlice {
			s, err = toStringList(val, mConfig.Delimiter, mConfig.DataType)
		} else {
			s, err = toString(val, mConfig.DataType)
		}
		if err != nil {
			return nil, err
		}
		envVals[ref] = s
	}
	return envVals, nil
}

func toStringList(val any, d string, dataType module.DataType) (string, error) {
	var sSl []string
	switch dataType {
	case module.StringType:
		sl, err := toSlice[string](val)
		if err != nil {
			return "", err
		}
		sSl = sl
	case module.BoolType:
		sl, err := toSlice[bool](val)
		if err != nil {
			return "", err
		}
		for _, b := range sl {
			sSl = append(sSl, strconv.FormatBool(b))
		}
	case module.Int64Type:
		sl, err := toSlice[int64](val)
		if err != nil {
			return "", err
		}
		for _, i := range sl {
			sSl = append(sSl, strconv.FormatInt(i, 10))
		}
	case module.Float64Type:
		sl, err := toSlice[float64](val)
		if err != nil {
			return "", err
		}
		for _, f := range sl {
			sSl = append(sSl, strconv.FormatFloat(f, 'f', -1, 64))
		}
	default:
		return "", fmt.Errorf("unknown data type '%s'", dataType)
	}
	return strings.Join(sSl, d), nil
}

func toSlice[T any](val any) ([]T, error) {
	sl, ok := val.([]T)
	if !ok {
		return nil, fmt.Errorf("invalid data type '%T'", val)
	}
	return sl, nil
}

func toString(val any, dataType module.DataType) (string, error) {
	switch dataType {
	case module.StringType:
		s, err := parseString(val)
		if err != nil {
			return "", err
		}
		return s, nil
	case module.BoolType:
		b, err := parseBool(val)
		if err != nil {
			return "", err
		}
		return strconv.FormatBool(b), nil
	case module.Int64Type:
		i, err := parseInt64(val)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(i, 10), nil
	case module.Float64Type:
		f, err := parseFloat64(val)
		if err != nil {
			return "", err
		}
		return strconv.FormatFloat(f, 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("unknown data type '%s'", dataType)
	}
}
