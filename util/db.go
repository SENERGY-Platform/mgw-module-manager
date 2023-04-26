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
	"context"
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"time"
)

func InitDB(ctx context.Context, addr string, port uint, user string, pw string, name string, moc int, mic int, timeout time.Duration) (*sql.DB, error) {
	tmpDB, err := newDB(addr, port, user, pw, "", 3*time.Minute, 1, 1)
	if err != nil {
		return nil, err
	}
	defer tmpDB.Close()
	ctxWT, cf := context.WithTimeout(ctx, timeout)
	defer cf()
	if err = createDB(tmpDB, ctxWT, name); err != nil {
		return nil, err
	}
	db, err := newDB(addr, port, user, pw, name, 3*time.Minute, moc, mic)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func createDB(db *sql.DB, ctx context.Context, name string) error {
	result, err := db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+name)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		Logger.Infof("created database '%s'", name)
	}
	return nil
}

func newDB(addr string, port uint, user string, pw string, name string, cml time.Duration, moc int, mic int) (*sql.DB, error) {
	cfg := mysql.NewConfig()
	cfg.Addr = fmt.Sprintf("%s:%d", addr, port)
	cfg.User = user
	cfg.Passwd = pw
	if name != "" {
		cfg.DBName = name
	}
	dc, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(dc)
	db.SetConnMaxLifetime(cml)
	db.SetMaxOpenConns(moc)
	db.SetMaxIdleConns(mic)
	return db, nil
}
