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

import (
	"fmt"
	"regexp"
	"unicode/utf8"
)

func Regex(params map[string]any) (bool, error) {
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

func TextLenCompare(params map[string]any) (bool, error) {
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
