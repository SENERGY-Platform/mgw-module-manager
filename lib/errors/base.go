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

type iErrBase interface {
	Error() string
	Unwrap() error
}

func New[
	T iErrBase,
	PT interface {
		*T
		set(msg string, err error)
	},
](msg string) *T {
	pt := PT(new(T))
	pt.set(msg, nil)
	return pt
}

func Wrap[
	T iErrBase,
	PT interface {
		*T
		set(msg string, err error)
	},
](err error) *T {
	pt := PT(new(T))
	pt.set("", err)
	return pt
}

type errBase struct {
	msg string
	err error
}

func (e errBase) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return e.msg
}

func (e errBase) Unwrap() error {
	return e.err
}
