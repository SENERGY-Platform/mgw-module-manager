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

package slog_attr

import "github.com/SENERGY-Platform/go-service-base/struct-logger/attributes"

const (
	ErrorKey                   = attributes.ErrorKey
	RequestIDKey               = "request_id"
	StackTraceKey              = "stack_trace"
	IDKey                      = "id"
	DirNameKey                 = "dir_name"
	SignalKey                  = "signal"
	VersionKey                 = "version"
	ConfigValuesKey            = "config_values"
	ComponentKey               = "component"
	LogRecordTypeKey           = attributes.LogRecordTypeKey
	HttpAccessLogRecordTypeVal = attributes.HttpAccessLogRecordTypeVal
	MethodKey                  = attributes.MethodKey
	PathKey                    = attributes.PathKey
)

var Provider = attributes.Provider
