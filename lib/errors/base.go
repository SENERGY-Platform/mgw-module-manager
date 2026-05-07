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

type iBaseErr interface {
	error
	Unwrap() error
}

type iBaseErrPtr[T iBaseErr] interface {
	*T
	set(msg string, err error)
}

func New[T iBaseErr, PT iBaseErrPtr[T]](msg string) *T {
	pt := PT(new(T))
	pt.set(msg, nil)
	return pt
}

func Wrap[T iBaseErr, PT iBaseErrPtr[T]](err error) *T {
	pt := PT(new(T))
	pt.set("", err)
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

type baseErr struct {
	msg string
	err error
}

func (e baseErr) Error() string {
	if e.err == nil {
		return e.msg
	}
	return e.err.Error()
}

func (e baseErr) Unwrap() error {
	return e.err
}
