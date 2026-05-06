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

package lib_errors

import (
	"errors"
	"fmt"
)

type iErrBase interface {
	error
	Unwrap() error
}

func New[
	T iErrBase,
	PT interface {
		*T
		set(msg string, err error, args []interface{})
	},
](msg string, args ...interface{}) *T {
	pt := PT(new(T))
	pt.set(msg, nil, args)
	return pt
}

func Wrap[
	T iErrBase,
	PT interface {
		*T
		set(msg string, err error, args []interface{})
	},
](err error, args ...interface{}) *T {
	pt := PT(new(T))
	pt.set("", err, args)
	return pt
}

func IsOf[
	T error,
	PT interface {
		*T
	},
](err error) bool {
	pt := PT(new(T))
	return errors.As(err, &pt)
}

type errBase struct {
	msg  string
	err  error
	args []interface{}
}

func (e errBase) Error() string {
	if e.err != nil {
		return genErrString(e.err.Error(), e.args)
	}
	return genErrString(e.msg, e.args)
}

func (e errBase) Unwrap() error {
	return e.err
}

func genErrString(msg string, args []interface{}) string {
	if len(args) == 0 {
		return msg
	}
	s := "msg=" + msg
	isKey := true
	for _, arg := range args {
		if isKey {
			s += fmt.Sprintf(" %v=", arg)
			isKey = false
		} else {
			s += fmt.Sprintf("%v", arg)
			isKey = true
		}
	}
	return s
}
