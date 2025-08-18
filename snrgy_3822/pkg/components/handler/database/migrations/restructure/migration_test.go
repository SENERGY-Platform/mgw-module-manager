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

package restructure

import (
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/database"
	helper_sql_db "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/sql_db"
	"log/slog"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	InitLogger(slog.Default())
	c, err := database.NewConnector(database.Config{
		Address:  "10.0.0.3:3306",
		Database: "module_manager_old",
		User:     "dev",
		Password: "dev123",
		Timeout:  time.Second * 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB := helper_sql_db.NewSQLDatabase(c, helper_sql_db.Config{
		MaxOpenConns:    25,
		MaxIdleConns:    25,
		ConnMaxLifetime: time.Minute * 5,
	})
	defer sqlDB.Close()
	err = migrateAuxConfigsTab(t.Context(), sqlDB)
	if err != nil {
		t.Fatal(err)
	}
}
