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

package api

import (
	"log/slog"

	sb_slog_attributes "github.com/SENERGY-Platform/go-service-base/struct-logger/attributes"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
)

var logger *slog.Logger
var accessLogger *slog.Logger

func InitLogger(sl *slog.Logger) {
	logger = sl.With(slog_keys.Component, "http-api")
	accessLogger = sl.With(slog_keys.Component, "http-api", sb_slog_attributes.LogRecordTypeKey, sb_slog_attributes.HttpAccessLogRecordTypeVal)
}

func init() {
	InitLogger(slog.Default())
}
