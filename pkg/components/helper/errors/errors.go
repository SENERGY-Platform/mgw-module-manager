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

package errors

import "fmt"

func Join(errs ...error) error {
	return Joinf("", "Err%d: %s", errs...)
}

func Joinp(prefixMsg string, errs ...error) error {
	return Joinf(prefixMsg, "Err%d: %s", errs...)
}

func Joinf(prefixMsg, format string, errs ...error) error {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	e := &joinErr{
		prefixMsg: prefixMsg,
		format:    format,
		errs:      make([]error, n),
	}
	for i, err := range errs {
		if err != nil {
			e.errs[i] = err
		}
	}
	return e
}

type joinErr struct {
	prefixMsg string
	format    string
	errs      []error
}

func (e *joinErr) Error() string {
	var msg string
	if e.prefixMsg != "" {
		msg += e.prefixMsg + " "
	}
	if len(e.errs) == 1 {
		return msg + e.errs[0].Error()
	}
	msg += fmt.Sprintf(e.format, 0, e.errs[0].Error())
	for i, err := range e.errs[1:] {
		msg += ", " + fmt.Sprintf(e.format, i+1, err.Error())
	}
	return msg
}

func (e *joinErr) Unwrap() []error {
	return e.errs
}
