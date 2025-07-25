/*
 * Copyright 2025 InfAI (CC SES)
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

package error

import "errors"

var NotFoundErr = errors.New("not found")

type MultiError struct {
	errs []error
}

func NewMultiError(errs []error) *MultiError {
	return &MultiError{errs: errs}
}

func (e *MultiError) Error() string {
	var str string
	errsLen := len(e.errs)
	for i, err := range e.errs {
		str += err.Error()
		if i < errsLen-1 {
			str += "\n"
		}
	}
	return str
}

func (e *MultiError) Errors() []error {
	return e.errs
}

type RepoErr struct {
	Source string
	err    error
}

func NewRepoErr(source string, err error) *RepoErr {
	return &RepoErr{
		Source: source,
		err:    err,
	}
}

func (e *RepoErr) Error() string {
	return e.err.Error()
}

func (e *RepoErr) Unwrap() error {
	return e.err
}

type RepoModuleErr struct {
	Source  string
	Channel string
	err     error
}

func NewRepoModuleErr(source, channel string, err error) *RepoModuleErr {
	return &RepoModuleErr{
		Source:  source,
		Channel: channel,
		err:     err,
	}
}

func (e *RepoModuleErr) Error() string {
	return e.err.Error()
}

func (e *RepoModuleErr) Unwrap() error {
	return e.err
}
