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

package modules_migr

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"io"
	"io/fs"
	"os"
	"path"
	"time"
)

type Migration struct {
	ModFileHandler ModFileHandler
	WrkSpcPath     string
	missing        []moduleWrapper
}

type item struct {
	ID       string    `json:"id"`
	Dir      string    `json:"dir"`
	ModFile  string    `json:"modfile"`
	Indirect bool      `json:"indirect"`
	Added    time.Time `json:"added"`
	Updated  time.Time `json:"updated"`
}

type moduleWrapper struct {
	*module.Module
	Dir     string
	ModFile string
	Added   time.Time
	Updated time.Time
}

func (m *Migration) Required(ctx context.Context, db *sql.DB, timeout time.Duration) (bool, error) {
	file, err := os.Open(path.Join(m.WrkSpcPath, "index"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		} else {
			return false, err
		}
	}
	defer file.Close()
	index, err := readIndex(file)
	if err != nil {
		return false, err
	}
	indexModules, err := m.loadModules(index)
	if err != nil {
		return false, err
	}
	ctxWt, cf := context.WithTimeout(ctx, timeout)
	defer cf()
	rows, err := db.QueryContext(ctxWt, "SELECT `id`, `dir`, `modfile` FROM `modules`")
	if err != nil {
		return false, err
	}
	defer rows.Close()
	dbModules := make(map[string]moduleWrapper)
	for rows.Next() {
		var id string
		var mod moduleWrapper
		if err = rows.Scan(&id, &mod.Dir, &mod.ModFile); err != nil {
			return false, err
		}
		dbModules[id] = mod
	}
	for mID, mod := range indexModules {
		if _, ok := dbModules[mID]; !ok {
			m.missing = append(m.missing, mod)
		}
	}
	return len(m.missing) > 0, nil
}

func (m *Migration) Run(ctx context.Context, db *sql.DB, timeout time.Duration) error {
	util.Logger.Warning("Migrating Modules ...")
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO `modules` (`id`, `dir`, `modfile`, `added`, `updated`) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	stmt2, err := tx.PrepareContext(ctx, "INSERT INTO `mod_dependencies` (`mod_id`, `req_id`) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt2.Close()
	ctxWt, cf := context.WithTimeout(ctx, timeout)
	defer cf()
	for _, mod := range m.missing {
		if _, err = stmt.ExecContext(ctxWt, mod.ID, mod.Dir, mod.ModFile, mod.Added, mod.Updated); err != nil {
			return err
		}
		if len(mod.Dependencies) > 0 {
			for mID := range mod.Dependencies {
				if _, err = stmt2.ExecContext(ctxWt, mod.ID, mID); err != nil {
					return err
				}
			}
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if _, err = db.ExecContext(ctxWt, "ALTER TABLE `deployments` ADD FOREIGN KEY (`mod_id`) REFERENCES `modules` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT"); err != nil {
		return err
	}
	err = os.RemoveAll(path.Join(m.WrkSpcPath, "index"))
	if err != nil {
		return err
	}
	util.Logger.Warning("Module migration successful")
	return nil
}

func (m *Migration) loadModules(index map[string]item) (map[string]moduleWrapper, error) {
	modules := make(map[string]moduleWrapper)
	for _, i := range index {
		f, err := os.Open(path.Join(m.WrkSpcPath, i.Dir, i.ModFile))
		if err != nil {
			return nil, err
		}
		mod, err := m.ModFileHandler.GetModule(f)
		if err != nil {
			return nil, err
		}
		modules[mod.ID] = moduleWrapper{
			Module:  mod,
			Dir:     i.Dir,
			ModFile: i.ModFile,
			Added:   i.Added,
			Updated: i.Updated,
		}
	}
	return modules, nil
}

func readIndex(file *os.File) (map[string]item, error) {
	index := make(map[string]item)
	jd := json.NewDecoder(file)
	for {
		var i item
		if err := jd.Decode(&i); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		index[i.ID] = i
	}
	return index, nil
}
