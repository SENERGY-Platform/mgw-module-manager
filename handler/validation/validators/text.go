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
	"errors"
	"fmt"
	"regexp"
	"unicode/utf8"
)

func Regex(params map[string]any) error {
	str, err := getParamValue[string](params, "string")
	if err != nil {
		return err
	}
	p, err := getParamValue[string](params, "pattern")
	if err != nil {
		return err
	}
	re, err := regexp.Compile(p)
	if err != nil {
		return fmt.Errorf("invalid pattern '%s'", p)
	}
	if !re.MatchString(str) {
		return errors.New("no match")
	}
	return nil
}

func TextLenCompare(params map[string]any) error {
	o, err := getParamValue[string](params, "operator")
	if err != nil {
		return err
	}
	s, err := getParamValue[string](params, "string")
	if err != nil {
		return err
	}
	l, err := getParamValue[int64](params, "length")
	if err != nil {
		return err
	}
	ok, err := compareNumber(int64(utf8.RuneCountInString(s)), l, o)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("invalid length")
	}
	return nil
}
