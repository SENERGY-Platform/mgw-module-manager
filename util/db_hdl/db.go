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

package db_hdl

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/go-sql-driver/mysql"
	"io"
	"os"
	"strings"
	"time"
)

type Migration interface {
	Required(ctx context.Context, tx *sql.Tx) (bool, error)
	Run(ctx context.Context, tx *sql.Tx) error
}

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

func InitDB(ctx context.Context, db *sql.DB, schemaPath string, delay time.Duration, migrations ...Migration) error {
	err := waitForDB(ctx, db, delay)
	if err != nil {
		return err
	}
	file, err := os.Open(schemaPath)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	var stmts []string
	for {
		stmt, err := reader.ReadString(';')
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		stmts = append(stmts, strings.TrimSuffix(stmt, ";"))
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, stmt := range stmts {
		_, err = tx.ExecContext(ctx, stmt)
		if err != nil {
			return err
		}
	}
	for _, migration := range migrations {
		ok, err := migration.Required(ctx, tx)
		if err != nil {
			return err
		}
		if ok {
			if err = migration.Run(ctx, tx); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func waitForDB(ctx context.Context, db *sql.DB, delay time.Duration) error {
	err := db.PingContext(ctx)
	if err == nil {
		return nil
	} else {
		util.Logger.Error(err)
	}
	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err = db.PingContext(ctx)
			if err == nil {
				return nil
			} else {
				util.Logger.Error(err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
