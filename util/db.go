/*
 * Copyright 2023 InfAI (CC SES)
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

package util

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"time"
)

func NewDB(addr string, port uint, user string, pw string, name string) (*sql.DB, error) {
	cfg := mysql.NewConfig()
	cfg.Addr = fmt.Sprintf("%s:%d", addr, port)
	cfg.User = user
	cfg.Passwd = pw
	cfg.DBName = name
	dc, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(dc)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}
