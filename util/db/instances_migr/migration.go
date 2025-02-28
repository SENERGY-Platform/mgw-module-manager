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

package instances_migr

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/SENERGY-Platform/mgw-go-service-base/context-hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/go-sql-driver/mysql"
	"time"
)

type Migration struct {
	Addr string
	Port uint
	User string
	PW   string
}

type instContainer struct {
	SrvRef string
	Order  int
	CtrID  string
}

type depContainer struct {
	instContainer
	Alias string
}

func (m *Migration) Required(ctx context.Context, db *sql.DB, timeout time.Duration) (bool, error) {
	cfg := mysql.NewConfig()
	cfg.Addr = fmt.Sprintf("%s:%d", m.Addr, m.Port)
	cfg.User = m.User
	cfg.Passwd = m.PW
	cfg.DBName = "information_schema"
	dc, err := mysql.NewConnector(cfg)
	if err != nil {
		return false, err
	}
	tmpDB := sql.OpenDB(dc)
	defer tmpDB.Close()
	ch := context_hdl.New()
	defer ch.CancelAll()
	row := tmpDB.QueryRowContext(ch.Add(context.WithTimeout(ctx, timeout)), "SELECT COUNT(*) FROM `tables` WHERE `table_name` = ?", "instances")
	var c int
	if err = row.Scan(&c); err != nil {
		return false, err
	}
	if c > 0 {
		row2 := db.QueryRowContext(ch.Add(context.WithTimeout(ctx, timeout)), "SELECT COUNT(*) FROM `instances`")
		var c2 int
		if err = row2.Scan(&c2); err != nil {
			return false, err
		}
		if c2 > 0 {
			row3 := db.QueryRowContext(ch.Add(context.WithTimeout(ctx, timeout)), "SELECT COUNT(*) FROM `containers`")
			var c3 int
			if err = row3.Scan(&c3); err != nil {
				return false, err
			}
			if c3 == 0 {
				return true, nil
			}
		}
	}
	return false, nil
}

func (m *Migration) Run(ctx context.Context, db *sql.DB, timeout time.Duration) error {
	util.Logger.Warning("Migrating Instances ...")
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	instContainers, err := m.getInstContainers(ctx, tx, timeout)
	if err != nil {
		return err
	}
	depContainers, err := m.getDepContainers(ctx, tx, instContainers, timeout)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO `containers` (`dep_id`, `ctr_id`, `srv_ref`, `alias`, `order`) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	ctxWt, cf := context.WithTimeout(ctx, timeout)
	defer cf()
	for dID, containers := range depContainers {
		for _, ctr := range containers {
			if _, err = stmt.ExecContext(ctxWt, dID, ctr.CtrID, ctr.SrvRef, ctr.Alias, ctr.Order); err != nil {
				return err
			}
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if err = m.cleanup(ctx, db, timeout); err != nil {
		return err
	}
	util.Logger.Warning("Instance migration successful")
	return nil
}

func (m *Migration) getInstContainers(ctx context.Context, tx *sql.Tx, timeout time.Duration) (map[string][]instContainer, error) {
	ctxWt, cf := context.WithTimeout(ctx, timeout)
	defer cf()
	rows, err := tx.QueryContext(ctxWt, "SELECT `inst_id`, `srv_ref`, `order`, `ctr_id` FROM `inst_containers` ORDER BY `inst_id`, `order` ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	containers := make(map[string][]instContainer)
	for rows.Next() {
		var id string
		var ctr instContainer
		if err = rows.Scan(&id, &ctr.SrvRef, &ctr.Order, &ctr.CtrID); err != nil {
			return nil, err
		}
		containers[id] = append(containers[id], ctr)
	}
	return containers, nil
}

func (m *Migration) getDepContainers(ctx context.Context, tx *sql.Tx, instContainers map[string][]instContainer, timeout time.Duration) (map[string][]depContainer, error) {
	ctxWt, cf := context.WithTimeout(ctx, timeout)
	defer cf()
	rows, err := tx.QueryContext(ctxWt, "SELECT `id`, `dep_id` FROM `instances`")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	depContainers := make(map[string][]depContainer)
	for rows.Next() {
		var id, dID string
		if err = rows.Scan(&id, &dID); err != nil {
			return nil, err
		}
		containers, ok := instContainers[id]
		if !ok {
			return nil, fmt.Errorf("deployment '%s' instance '%s' not found", dID, id)
		}
		for _, ctr := range containers {
			depContainers[dID] = append(depContainers[dID], depContainer{instContainer: ctr, Alias: getSrvName(dID, ctr.SrvRef)})
		}
	}
	return depContainers, nil
}

func (m *Migration) cleanup(ctx context.Context, db *sql.DB, timeout time.Duration) error {
	ctxWt, cf := context.WithTimeout(ctx, timeout)
	defer cf()
	if _, err := db.ExecContext(ctxWt, "DROP TABLE `inst_containers`"); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctxWt, "DROP TABLE `instances`"); err != nil {
		return err
	}
	return nil
}

func getSrvName(s, r string) string {
	return "mgw-inst-" + genHash(s, r)
}

func genHash(str ...string) string {
	hash := sha1.New()
	for _, s := range str {
		hash.Write([]byte(s))
	}
	return hex.EncodeToString(hash.Sum(nil))
}
