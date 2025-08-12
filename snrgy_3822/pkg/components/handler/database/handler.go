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

package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"github.com/go-sql-driver/mysql"
	"strings"
)

const timeLayout = "2006-01-02 15:04:05.000000"

func NewConnector(config Config) (driver.Connector, error) {
	cfg := mysql.NewConfig()
	cfg.Addr = config.Address
	cfg.User = config.User
	cfg.Passwd = config.Password
	cfg.DBName = config.Database
	cfg.Timeout = config.Timeout
	cfg.ReadTimeout = cfg.Timeout
	cfg.WriteTimeout = cfg.Timeout
	return mysql.NewConnector(cfg)
}

type Handler struct {
	sqlDB *sql.DB
}

func New(sqlDB *sql.DB) *Handler {
	return &Handler{sqlDB: sqlDB}
}

func (h *Handler) Migrate(ctx context.Context, migrations ...migration) error {
	for _, m := range migrations {
		ok, err := m.Required(ctx, h.sqlDB)
		if err != nil {
			return err
		}
		if ok {
			if err = m.Run(ctx, h.sqlDB); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) Ping(ctx context.Context) error {
	return h.sqlDB.PingContext(ctx)
}

func removeDuplicates[S ~[]E, E comparable](sl S) []E {
	if len(sl) < 2 {
		return sl
	}
	set := make(map[E]struct{})
	var sl2 []E
	for _, s := range sl {
		if _, ok := set[s]; !ok {
			sl2 = append(sl2, s)
		}
		set[s] = struct{}{}
	}
	return sl2
}

func toAnySlice[S ~[]E, E any](sl S) []any {
	anySl := make([]any, len(sl))
	for i, v := range sl {
		anySl[i] = v
	}
	return anySl
}

func genQuestionMarks(numCol int) string {
	if numCol <= 0 {
		return ""
	}
	if numCol >= 2 {
		return strings.Repeat("?, ", numCol-1) + "?"
	}
	return "?"
}
