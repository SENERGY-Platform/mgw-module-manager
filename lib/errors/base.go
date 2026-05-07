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
)

func New[
	T any,
	PT interface {
		*T
		error
		init(msg string, err error)
	},
](msg string) *T {
	pt := PT(new(T))
	pt.init(msg, nil)
	return pt
}

func Wrap[
	T any,
	PT interface {
		*T
		error
		init(msg string, err error)
	},
](err error) *T {
	pt := PT(new(T))
	pt.init("", err)
	return pt
}

func IsOf[
	T any,
	PT interface {
		*T
		error
	},
](err error) bool {
	for {
		_, ok := err.(PT)
		if ok {
			return true
		}
		err = errors.Unwrap(err)
		if err == nil {
			break
		}
	}
	return false
}

type errBase struct {
	msg string
	err error
}

func (e *errBase) Error() string {
	if e.err == nil {
		return e.msg
	}
	return e.err.Error()
}

func (e *errBase) Unwrap() error {
	return e.err
}

func (e *errBase) init(msg string, err error) {
	*e = errBase{
		msg: msg,
		err: err,
	}
}
