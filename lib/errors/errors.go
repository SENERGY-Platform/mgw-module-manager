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

type ErrNotFound struct {
	errBase
}

func (e *ErrNotFound) init(msg string, err error) {
	if msg == "" {
		msg = "not found"
	}
	*e = ErrNotFound{errBase{msg: msg, err: err}}
}

type ErrExists struct {
	errBase
}

func (e *ErrExists) init(msg string, err error) {
	if msg == "" {
		msg = "exists"
	}
	*e = ErrExists{errBase{msg: msg, err: err}}
}

type ErrActiveJob struct {
	errBase
}

func (e *ErrActiveJob) init(msg string, err error) {
	if msg == "" {
		msg = "active job"
	}
	*e = ErrActiveJob{errBase{msg: msg, err: err}}
}

type ErrInvalidInput struct {
	errBase
}

func (e *ErrInvalidInput) init(msg string, err error) {
	if msg == "" {
		msg = "invalid input"
	}
	*e = ErrInvalidInput{errBase{msg: msg, err: err}}
}
