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
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/context_hdl"
	"github.com/go-sql-driver/mysql"
	"time"
)

type Migration struct {
	Addr    string
	Port    uint
	User    string
	PW      string
	Timeout time.Duration
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

func (m *Migration) Required(ctx context.Context, tx *sql.Tx) (bool, error) {
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
	row := tmpDB.QueryRowContext(ch.Add(context.WithTimeout(ctx, m.Timeout)), "SELECT COUNT(*) FROM `tables` WHERE `table_name` = ?", "instances")
	var c int
	if err = row.Scan(&c); err != nil {
		return false, err
	}
	if c > 0 {
		row2 := tx.QueryRowContext(ch.Add(context.WithTimeout(ctx, m.Timeout)), "SELECT COUNT(*) FROM `instances`")
		var c2 int
		if err = row2.Scan(&c2); err != nil {
			return false, err
		}
		if c2 > 0 {
			row3 := tx.QueryRowContext(ch.Add(context.WithTimeout(ctx, m.Timeout)), "SELECT COUNT(*) FROM `containers`")
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

func (m *Migration) Run(ctx context.Context, tx *sql.Tx) error {
	util.Logger.Warning("Migrating Instances ...")
	instContainers, err := m.getInstContainers(ctx, tx)
	if err != nil {
		return err
	}
	depContainers, err := m.getDepContainers(ctx, tx, instContainers)
	if err != nil {
		return err
	}
	ctxWt, cf := context.WithTimeout(ctx, m.Timeout)
	defer cf()
	stmt, err := tx.PrepareContext(ctxWt, "INSERT INTO `containers` (`dep_id`, `ctr_id`, `srv_ref`, `alias`, `order`) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for dID, containers := range depContainers {
		for _, ctr := range containers {
			if _, err = stmt.ExecContext(ctx, dID, ctr.CtrID, ctr.SrvRef, ctr.Alias, ctr.Order); err != nil {
				return err
			}
		}
	}
	if err = m.cleanup(ctx, tx); err != nil {
		return err
	}
	util.Logger.Warning("Instance migration finished")
	return nil
}

func (m *Migration) getInstContainers(ctx context.Context, tx *sql.Tx) (map[string][]instContainer, error) {
	ctxWt, cf := context.WithTimeout(ctx, m.Timeout)
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

func (m *Migration) getDepContainers(ctx context.Context, tx *sql.Tx, instContainers map[string][]instContainer) (map[string][]depContainer, error) {
	ctxWt, cf := context.WithTimeout(ctx, m.Timeout)
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

func (m *Migration) cleanup(ctx context.Context, tx *sql.Tx) error {
	ctxWt, cf := context.WithTimeout(ctx, m.Timeout)
	defer cf()
	if _, err := tx.ExecContext(ctxWt, "DROP TABLE `inst_containers`"); err != nil {
		return err
	}
	ctxWt2, cf2 := context.WithTimeout(ctx, m.Timeout)
	defer cf2()
	if _, err := tx.ExecContext(ctxWt2, "DROP TABLE `instances`"); err != nil {
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
