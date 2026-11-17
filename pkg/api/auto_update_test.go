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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
)

type autoUpdateTestDB struct {
	row pkg_models.AutoUpdate
}

func (d *autoUpdateTestDB) Ping(_ context.Context) error { return nil }

func (d *autoUpdateTestDB) ReadAutoUpdate(_ context.Context) (pkg_models.AutoUpdate, error) {
	return d.row, nil
}

func (d *autoUpdateTestDB) WriteAutoUpdateSettings(_ context.Context, settings lib_models.AutoUpdateSettings) error {
	d.row.AutoUpdateSettings = settings
	return nil
}

func (d *autoUpdateTestDB) WriteAutoUpdateRunDate(_ context.Context, date string) error {
	d.row.LastRunDate = date
	return nil
}

func (d *autoUpdateTestDB) WriteAutoUpdateLastRun(_ context.Context, run lib_models.AutoUpdateRun) error {
	d.row.LastRun = &run
	return nil
}

func newAutoUpdateTestHandler(t *testing.T, db *autoUpdateTestDB) http.Handler {
	t.Helper()
	handler, err := CreateHandler(service.New(nil, nil, nil, nil, nil, nil, db, nil, nil), "test", "test", false)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestAutoUpdateStatus_ListsAreNeverNull(t *testing.T) {
	db := &autoUpdateTestDB{row: pkg_models.AutoUpdate{
		AutoUpdateSettings: lib_models.AutoUpdateSettings{Level: "patch", WindowStart: "02:00", WindowEnd: "05:00", TimeZone: "UTC"},
		LastRun: &lib_models.AutoUpdateRun{
			Skipped: []lib_models.AutoUpdateSkippedModule{{Id: "mod-a", Reason: "level_exceeded"}},
		},
	}}
	rec := httptest.NewRecorder()
	newAutoUpdateTestHandler(t, db).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auto-update/status", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "null") {
		t.Errorf("response contains null: %s", body)
	}
	var status struct {
		LastRun struct {
			Updated []json.RawMessage `json:"updated"`
			Failed  []json.RawMessage `json:"failed"`
			Skipped []struct {
				Missing []json.RawMessage `json:"missing"`
			} `json:"skipped"`
		} `json:"last_run"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.LastRun.Updated == nil || status.LastRun.Failed == nil || len(status.LastRun.Skipped) != 1 || status.LastRun.Skipped[0].Missing == nil {
		t.Errorf("expected empty lists, got %s", body)
	}
}

func TestAutoUpdateSettings_InvalidTimeZoneIsBadRequest(t *testing.T) {
	db := &autoUpdateTestDB{}
	handler := newAutoUpdateTestHandler(t, db)
	body := `{"enabled":true,"level":"patch","window_start":"02:00","window_end":"05:00","time_zone":"Europe/Atlantis"}`
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/auto-update", strings.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if db.row.TimeZone != "" {
		t.Errorf("invalid time zone stored: %+v", db.row)
	}
	body = strings.Replace(body, "Europe/Atlantis", "Asia/Tokyo", 1)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/auto-update", strings.NewReader(body)))
	if rec.Code != http.StatusOK || db.row.TimeZone != "Asia/Tokyo" {
		t.Errorf("valid time zone rejected: status %d, stored %+v", rec.Code, db.row.AutoUpdateSettings)
	}
}

func TestAutoUpdateStatus_ReachableFromModules(t *testing.T) {
	handler := newAutoUpdateTestHandler(t, &autoUpdateTestDB{row: pkg_models.AutoUpdate{
		AutoUpdateSettings: lib_models.AutoUpdateSettings{Level: "patch", WindowStart: "02:00", WindowEnd: "05:00", TimeZone: "UTC"},
	}})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/restricted/auto-update/status", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("GET /restricted/auto-update/status: expected 200, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/restricted/auto-update", strings.NewReader(`{}`)))
	if rec.Code == http.StatusOK {
		t.Errorf("PUT /restricted/auto-update: modules must not change the settings")
	}
}
