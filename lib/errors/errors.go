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

func (e *ErrNotFound) set(msg string, err error, args []interface{}) {
	*e = ErrNotFound{
		errBase{
			msg:  msg,
			err:  err,
			args: args,
		},
	}
}

type ErrExists struct {
	errBase
}

func (e *ErrExists) set(msg string, err error, args []interface{}) {
	*e = ErrExists{
		errBase{
			msg:  msg,
			err:  err,
			args: args,
		},
	}
}

type ErrActiveJob struct {
	errBase
}

func (e *ErrActiveJob) set(msg string, err error, args []interface{}) {
	*e = ErrActiveJob{
		errBase{
			msg:  msg,
			err:  err,
			args: args,
		},
	}
}

type ErrInvalidInput struct {
	errBase
}

func (e *ErrInvalidInput) set(msg string, err error, args []interface{}) {
	*e = ErrInvalidInput{
		errBase{
			msg:  msg,
			err:  err,
			args: args,
		},
	}
}
