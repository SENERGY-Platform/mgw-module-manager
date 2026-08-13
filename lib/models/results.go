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

package models

// ErrorResult reports an error that occurred during an operation which is not aborted by the error.
// It is embedded in models that can be returned in a partially populated or failed state.
type ErrorResult struct {
	HasError bool   `json:"has_error"` // true if an error occurred, must be checked before relying on the enclosing model
	ErrorMsg string `json:"error_msg"` // error message, empty if HasError is false
}

// NewErrorResult returns an ErrorResult marked as failed with the given error message.
func NewErrorResult(msg string) ErrorResult {
	return ErrorResult{
		HasError: true,
		ErrorMsg: msg,
	}
}
