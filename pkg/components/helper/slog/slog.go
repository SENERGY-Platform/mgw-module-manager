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

package slog

import (
	"context"
	"io"
	"iter"
	"log/slog"

	struct_logger "github.com/SENERGY-Platform/go-service-base/struct-logger"
	sb_slog_attributes "github.com/SENERGY-Platform/go-service-base/struct-logger/attributes"
	"github.com/SENERGY-Platform/go-service-base/struct-logger/handlers"
)

var ContextAttributeKeys []string

func GetContextAttributes(ctx context.Context) iter.Seq[slog.Attr] {
	return func(yield func(slog.Attr) bool) {
		for _, key := range ContextAttributeKeys {
			val := ctx.Value(key)
			if val != nil {

				if !yield(slog.Any(key, val)) {
					return
				}
			}
		}
	}
}

func New(c struct_logger.Config, out io.Writer, organization, project string) *slog.Logger {
	recordTime := struct_logger.NewRecordTime(c.TimeFormat, c.TimeUtc)
	options := &slog.HandlerOptions{
		AddSource:   c.AddSource,
		Level:       struct_logger.GetLevel(c.Level, slog.LevelInfo),
		ReplaceAttr: recordTime.ReplaceAttr,
	}
	handler := handlers.NewContextHandler(
		struct_logger.GetHandler(c.Handler, out, options, slog.Default().Handler()),
		GetContextAttributes,
	)
	if c.AddMeta {
		var attr []slog.Attr
		if organization != "" {
			attr = append(attr, slog.String(sb_slog_attributes.OrganizationKey, organization))
		}
		if project != "" {
			attr = append(attr, slog.String(sb_slog_attributes.ProjectKey, project))
		}
		handler = handler.WithAttrs(attr)
	}
	return slog.New(handler)
}
